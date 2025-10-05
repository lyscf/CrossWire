package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// ARPTransport ARP传输层实现（服务器签名广播模式）
// 参考: docs/ARP_BROADCAST_MODE.md
// 参考: docs/PROTOCOL.md - 2. ARP传输协议
type ARPTransport struct {
	config *Config
	mode   string // "server" or "client"

	// 网卡
	handle   *pcap.Handle
	iface    *net.Interface
	localMAC net.HardwareAddr
	localIP  net.IP

	// 服务器信息
	serverMAC     net.HardwareAddr
	serverPubKey  []byte // Ed25519公钥（客户端验证签名用）
	serverPrivKey []byte // Ed25519私钥（服务端签名用）

	// 加密
	channelKey []byte // AES-256密钥

	// 消息处理
	handler     MessageHandler
	fileHandler FileHandler

	// 序列号管理
	sequence   uint32
	sequenceMu sync.Mutex

	// 已处理消息（防重放）
	seenMsgs   map[string]time.Time
	seenMsgsMu sync.RWMutex

	// 统计
	stats   TransportStats
	statsMu sync.RWMutex

	// 控制
	ctx       context.Context
	cancel    context.CancelFunc
	started   bool
	connected bool

	// 重传队列（TODO）
	// retryQueue map[uint32]*Message
}

// ARPFrame ARP帧结构
// 参考: docs/PROTOCOL.md - 2.1 以太网帧结构
type ARPFrame struct {
	// 以太网头部
	DstMAC net.HardwareAddr // 目标MAC
	SrcMAC net.HardwareAddr // 源MAC

	// 自定义协议头部
	EtherType   uint16 // 0x88B5
	Version     uint8  // 协议版本
	FrameType   uint8  // 帧类型
	Sequence    uint32 // 序列号
	TotalChunks uint16 // 总分块数
	ChunkIndex  uint16 // 当前分块索引
	PayloadLen  uint16 // 负载长度
	Checksum    uint32 // CRC32校验
	Reserved    uint32 // 预留字段

	// 负载
	Payload []byte // 加密后的数据（服务器模式包含签名）
}

// SignedPayload 服务器签名的载荷
// 参考: docs/ARP_BROADCAST_MODE.md - 2. 服务器签名与广播
type SignedPayload struct {
	Message   []byte `json:"message"`   // 加密消息
	Signature []byte `json:"signature"` // Ed25519签名
	Timestamp int64  `json:"timestamp"` // 时间戳
}

// NewARPTransport 创建ARP传输层
func NewARPTransport() *ARPTransport {
	return &ARPTransport{
		seenMsgs: make(map[string]time.Time),
	}
}

// ===== 生命周期管理 =====

// Init 初始化
func (t *ARPTransport) Init(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Interface == "" {
		return fmt.Errorf("interface name is required for ARP transport")
	}

	t.config = config
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.stats.StartTime = time.Now()

	// 获取网卡信息
	iface, err := net.InterfaceByName(config.Interface)
	if err != nil {
		return fmt.Errorf("failed to get interface: %w", err)
	}
	t.iface = iface
	t.localMAC = iface.HardwareAddr

	// 获取本地IP
	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get addresses: %w", err)
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			t.localIP = ipnet.IP
			break
		}
	}

	return nil
}

// Start 启动传输层
func (t *ARPTransport) Start() error {
	if t.started {
		return fmt.Errorf("transport already started")
	}

	if t.config == nil {
		return fmt.Errorf("transport not initialized")
	}

	// 打开pcap句柄（原始以太网包）
	handle, err := pcap.OpenLive(
		t.config.Interface,
		65536, // 快照长度
		true,  // 混杂模式
		pcap.BlockForever,
	)
	if err != nil {
		return fmt.Errorf("failed to open pcap: %w", err)
	}
	t.handle = handle

	// 设置BPF过滤器（只接收CrossWire协议的帧）
	filter := fmt.Sprintf("ether proto 0x%04x", EtherTypeCustom)
	if err := handle.SetBPFFilter(filter); err != nil {
		return fmt.Errorf("failed to set BPF filter: %w", err)
	}

	t.started = true

	// 启动接收协程
	go t.receiveLoop()

	// 启动清理协程（清理过期的seenMsgs）
	go t.cleanupLoop()

	fmt.Printf("ARP transport started on %s (%s)\n",
		t.config.Interface, t.localMAC)

	return nil
}

// Stop 停止传输层
func (t *ARPTransport) Stop() error {
	if !t.started {
		return nil
	}

	t.started = false
	t.cancel()

	if t.handle != nil {
		t.handle.Close()
	}

	return nil
}

// ===== 连接管理 =====

// Connect 连接到服务器（客户端模式）
func (t *ARPTransport) Connect(target string) error {
	// target 格式: "MAC地址" 或 "MAC地址:ServerPublicKey"
	// 例如: "aa:bb:cc:dd:ee:ff" 或 "aa:bb:cc:dd:ee:ff:base64pubkey"

	t.mode = "client"

	// 解析服务器MAC地址
	mac, err := net.ParseMAC(target)
	if err != nil {
		return fmt.Errorf("invalid server MAC address: %w", err)
	}
	t.serverMAC = mac

	// TODO: 从target解析服务器公钥
	// 或通过发现阶段获取

	t.connected = true

	fmt.Printf("Connected to server %s\n", mac)
	return nil
}

// Disconnect 断开连接
func (t *ARPTransport) Disconnect() error {
	t.connected = false
	t.serverMAC = nil
	t.serverPubKey = nil
	return nil
}

// IsConnected 检查连接状态
func (t *ARPTransport) IsConnected() bool {
	return t.connected
}

// ===== 消息收发 =====

// SendMessage 发送消息
// 参考: docs/ARP_BROADCAST_MODE.md - 1. 消息发送
func (t *ARPTransport) SendMessage(msg *Message) error {
	if t.mode == "server" {
		// 服务器模式：签名并广播
		return t.signAndBroadcast(msg)
	} else {
		// 客户端模式：单播给服务器
		return t.sendToServer(msg)
	}
}

// sendToServer 客户端单播给服务器
func (t *ARPTransport) sendToServer(msg *Message) error {
	if !t.connected {
		return fmt.Errorf("not connected to server")
	}

	// 构造ARP帧
	frame := &ARPFrame{
		DstMAC:      t.serverMAC, // 单播给服务器
		SrcMAC:      t.localMAC,
		EtherType:   EtherTypeCustom,
		Version:     uint8(ProtocolVersion),
		FrameType:   uint8(msg.Type),
		Sequence:    t.nextSequence(),
		TotalChunks: 1, // TODO: 实现分块
		ChunkIndex:  0,
		Payload:     msg.Payload,
	}

	// 计算校验和
	frame.Checksum = crc32.ChecksumIEEE(frame.Payload)
	frame.PayloadLen = uint16(len(frame.Payload))

	// 发送原始帧
	return t.sendRawFrame(frame)
}

// signAndBroadcast 服务器签名并广播
// 参考: docs/ARP_BROADCAST_MODE.md - 2. 服务器签名与广播
func (t *ARPTransport) signAndBroadcast(msg *Message) error {
	if t.mode != "server" {
		return fmt.Errorf("only server can broadcast")
	}

	// 使用Ed25519签名
	// TODO: 集成crypto.Manager
	// signature := ed25519.Sign(t.serverPrivKey, msg.Payload)

	// 暂时跳过签名（等集成crypto）
	signature := []byte("TODO_SIGNATURE")

	// 构造签名载荷
	signedPayload := &SignedPayload{
		Message:   msg.Payload,
		Signature: signature,
		Timestamp: time.Now().UnixNano(),
	}

	// 序列化（TODO: 使用更高效的编码）
	payloadBytes := t.serializeSignedPayload(signedPayload)

	// 构造广播帧
	broadcastMAC, _ := net.ParseMAC(BroadcastMAC)
	frame := &ARPFrame{
		DstMAC:      broadcastMAC, // 广播
		SrcMAC:      t.localMAC,
		EtherType:   EtherTypeCustom,
		Version:     uint8(ProtocolVersion),
		FrameType:   uint8(msg.Type),
		Sequence:    t.nextSequence(),
		TotalChunks: 1,
		ChunkIndex:  0,
		Payload:     payloadBytes,
	}

	frame.Checksum = crc32.ChecksumIEEE(frame.Payload)
	frame.PayloadLen = uint16(len(frame.Payload))

	// 广播
	return t.sendRawFrame(frame)
}

// sendRawFrame 发送原始以太网帧
func (t *ARPTransport) sendRawFrame(frame *ARPFrame) error {
	// 序列化帧
	packet := t.serializeFrame(frame)

	// 发送
	err := t.handle.WritePacketData(packet)
	if err != nil {
		return fmt.Errorf("failed to send frame: %w", err)
	}

	// 更新统计
	t.statsMu.Lock()
	t.stats.BytesSent += uint64(len(packet))
	t.stats.MessagesSent++
	t.stats.LastActivity = time.Now()
	t.statsMu.Unlock()

	return nil
}

// receiveLoop 接收循环
// 参考: docs/ARP_BROADCAST_MODE.md - 3. 消息接收
func (t *ARPTransport) receiveLoop() {
	packetSource := gopacket.NewPacketSource(t.handle, t.handle.LinkType())

	for {
		select {
		case <-t.ctx.Done():
			return
		case packet := <-packetSource.Packets():
			if packet == nil {
				continue
			}

			// 解析以太网层
			ethLayer := packet.Layer(layers.LayerTypeEthernet)
			if ethLayer == nil {
				continue
			}

			eth, _ := ethLayer.(*layers.Ethernet)

			// 检查EtherType
			if eth.EthernetType != layers.EthernetType(EtherTypeCustom) {
				continue
			}

			// 解析自定义帧
			frame := t.parseFrame(packet.Data())
			if frame == nil {
				continue
			}

			// 处理帧
			t.handleFrame(frame)
		}
	}
}

// handleFrame 处理接收到的帧
func (t *ARPTransport) handleFrame(frame *ARPFrame) {
	// 客户端模式：验证服务器签名
	if t.mode == "client" {
		// 只接受来自服务器的广播
		if !bytes.Equal(frame.SrcMAC, t.serverMAC) {
			return
		}

		// 解析签名载荷
		signedPayload := t.parseSignedPayload(frame.Payload)
		if signedPayload == nil {
			return
		}

		// 验证Ed25519签名
		// TODO: 集成crypto.Manager
		// if !ed25519.Verify(t.serverPubKey, signedPayload.Message, signedPayload.Signature) {
		//     fmt.Println("Invalid signature, possible attack!")
		//     return
		// }

		// 验证时间戳（防重放）
		if !t.validateTimestamp(signedPayload.Timestamp) {
			fmt.Println("Message too old, possible replay attack")
			return
		}

		// 检查是否已处理
		msgID := fmt.Sprintf("%d", frame.Sequence)
		if t.hasSeenMessage(msgID) {
			return
		}
		t.markMessageAsSeen(msgID)

		// 构造消息
		msg := &Message{
			Sequence:  frame.Sequence,
			Type:      MessageType(frame.FrameType),
			Payload:   signedPayload.Message,
			Timestamp: time.Unix(0, signedPayload.Timestamp),
			SenderMAC: frame.SrcMAC.String(),
		}

		// 更新统计
		t.statsMu.Lock()
		t.stats.BytesReceived += uint64(len(frame.Payload))
		t.stats.MessagesRecv++
		t.stats.LastActivity = time.Now()
		t.statsMu.Unlock()

		// 调用处理函数
		if t.handler != nil {
			go t.handler(msg)
		}

	} else if t.mode == "server" {
		// 服务器模式：接收客户端单播的消息
		// 检查是否发给自己
		if !bytes.Equal(frame.DstMAC, t.localMAC) {
			return
		}

		// 构造消息
		msg := &Message{
			Sequence:  frame.Sequence,
			Type:      MessageType(frame.FrameType),
			Payload:   frame.Payload,
			Timestamp: time.Now(),
			SenderMAC: frame.SrcMAC.String(),
		}

		// 更新统计
		t.statsMu.Lock()
		t.stats.BytesReceived += uint64(len(frame.Payload))
		t.stats.MessagesRecv++
		t.stats.LastActivity = time.Now()
		t.statsMu.Unlock()

		// 调用处理函数
		if t.handler != nil {
			go t.handler(msg)
		}
	}
}

// ReceiveMessage 接收消息（阻塞）- 不推荐使用，建议用Subscribe
func (t *ARPTransport) ReceiveMessage() (*Message, error) {
	return nil, fmt.Errorf("use Subscribe instead")
}

// Subscribe 订阅消息（异步回调）
func (t *ARPTransport) Subscribe(handler MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}
	t.handler = handler
	return nil
}

// Unsubscribe 取消订阅
func (t *ARPTransport) Unsubscribe() {
	t.handler = nil
}

// ===== 辅助方法 =====

// nextSequence 获取下一个序列号
func (t *ARPTransport) nextSequence() uint32 {
	t.sequenceMu.Lock()
	defer t.sequenceMu.Unlock()
	t.sequence++
	return t.sequence
}

// hasSeenMessage 检查消息是否已处理
func (t *ARPTransport) hasSeenMessage(msgID string) bool {
	t.seenMsgsMu.RLock()
	defer t.seenMsgsMu.RUnlock()
	_, exists := t.seenMsgs[msgID]
	return exists
}

// markMessageAsSeen 标记消息为已处理
func (t *ARPTransport) markMessageAsSeen(msgID string) {
	t.seenMsgsMu.Lock()
	defer t.seenMsgsMu.Unlock()
	t.seenMsgs[msgID] = time.Now()
}

// validateTimestamp 验证时间戳
func (t *ARPTransport) validateTimestamp(timestamp int64) bool {
	now := time.Now().UnixNano()
	diff := now - timestamp
	if diff < 0 {
		diff = -diff
	}
	// 容忍5分钟
	return diff < 5*60*1e9
}

// cleanupLoop 清理过期的seenMsgs
func (t *ARPTransport) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.cleanupSeenMessages()
		}
	}
}

// cleanupSeenMessages 清理10分钟前的消息记录
func (t *ARPTransport) cleanupSeenMessages() {
	t.seenMsgsMu.Lock()
	defer t.seenMsgsMu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for msgID, timestamp := range t.seenMsgs {
		if timestamp.Before(cutoff) {
			delete(t.seenMsgs, msgID)
		}
	}
}

// serializeFrame 序列化帧
func (t *ARPTransport) serializeFrame(frame *ARPFrame) []byte {
	buf := make([]byte, 0, FrameHeaderSize+len(frame.Payload))

	// 以太网头部（通过gopacket构造）
	ethLayer := &layers.Ethernet{
		SrcMAC:       frame.SrcMAC,
		DstMAC:       frame.DstMAC,
		EthernetType: layers.EthernetType(frame.EtherType),
	}

	// 自定义头部
	header := make([]byte, FrameHeaderSize-14) // 减去以太网头部
	header[0] = frame.Version
	header[1] = frame.FrameType
	binary.BigEndian.PutUint32(header[2:6], frame.Sequence)
	binary.BigEndian.PutUint16(header[6:8], frame.TotalChunks)
	binary.BigEndian.PutUint16(header[8:10], frame.ChunkIndex)
	binary.BigEndian.PutUint16(header[10:12], frame.PayloadLen)
	binary.BigEndian.PutUint32(header[12:16], frame.Checksum)
	binary.BigEndian.PutUint32(header[16:20], frame.Reserved)

	// 序列化以太网层
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	gopacket.SerializeLayers(buffer, opts,
		ethLayer,
		gopacket.Payload(append(header, frame.Payload...)),
	)

	buf = buffer.Bytes()
	return buf
}

// parseFrame 解析帧
func (t *ARPTransport) parseFrame(data []byte) *ARPFrame {
	if len(data) < FrameHeaderSize {
		return nil
	}

	frame := &ARPFrame{}

	// 以太网头部
	frame.DstMAC = net.HardwareAddr(data[0:6])
	frame.SrcMAC = net.HardwareAddr(data[6:12])
	frame.EtherType = binary.BigEndian.Uint16(data[12:14])

	// 自定义头部
	frame.Version = data[14]
	frame.FrameType = data[15]
	frame.Sequence = binary.BigEndian.Uint32(data[16:20])
	frame.TotalChunks = binary.BigEndian.Uint16(data[20:22])
	frame.ChunkIndex = binary.BigEndian.Uint16(data[22:24])
	frame.PayloadLen = binary.BigEndian.Uint16(data[24:26])
	frame.Checksum = binary.BigEndian.Uint32(data[26:30])
	frame.Reserved = binary.BigEndian.Uint32(data[30:34])

	// 负载
	if len(data) > FrameHeaderSize {
		frame.Payload = data[FrameHeaderSize:]
	}

	// 验证校验和
	if crc32.ChecksumIEEE(frame.Payload) != frame.Checksum {
		return nil
	}

	return frame
}

// serializeSignedPayload 序列化签名载荷
func (t *ARPTransport) serializeSignedPayload(payload *SignedPayload) []byte {
	// TODO: 使用更高效的编码（Protobuf/MessagePack）
	// 暂时简单拼接
	buf := make([]byte, 0, 8+len(payload.Message)+len(payload.Signature))

	// Timestamp (8 bytes)
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, uint64(payload.Timestamp))
	buf = append(buf, tsBytes...)

	// Signature length + Signature
	sigLen := make([]byte, 2)
	binary.BigEndian.PutUint16(sigLen, uint16(len(payload.Signature)))
	buf = append(buf, sigLen...)
	buf = append(buf, payload.Signature...)

	// Message
	buf = append(buf, payload.Message...)

	return buf
}

// parseSignedPayload 解析签名载荷
func (t *ARPTransport) parseSignedPayload(data []byte) *SignedPayload {
	if len(data) < 10 { // 最少：8(timestamp) + 2(sigLen)
		return nil
	}

	payload := &SignedPayload{}

	// Timestamp
	payload.Timestamp = int64(binary.BigEndian.Uint64(data[0:8]))

	// Signature
	sigLen := binary.BigEndian.Uint16(data[8:10])
	if len(data) < 10+int(sigLen) {
		return nil
	}
	payload.Signature = data[10 : 10+sigLen]

	// Message
	payload.Message = data[10+sigLen:]

	return payload
}

// ===== 其他接口实现 =====

// SendFile 发送文件
func (t *ARPTransport) SendFile(file *FileTransfer) error {
	// TODO: 实现文件分块传输
	return fmt.Errorf("not implemented")
}

// OnFileReceived 文件接收回调
func (t *ARPTransport) OnFileReceived(handler FileHandler) error {
	t.fileHandler = handler
	return nil
}

// Discover 发现可用的服务端
func (t *ARPTransport) Discover(timeout time.Duration) ([]*PeerInfo, error) {
	// TODO: 实现ARP服务发现
	return nil, fmt.Errorf("not implemented")
}

// Announce 宣告服务
func (t *ARPTransport) Announce(info *ServiceInfo) error {
	// TODO: 实现服务宣告
	return fmt.Errorf("not implemented")
}

// GetMode 获取传输模式
func (t *ARPTransport) GetMode() TransportMode {
	return TransportModeARP
}

// GetStats 获取传输统计
func (t *ARPTransport) GetStats() *TransportStats {
	t.statsMu.RLock()
	defer t.statsMu.RUnlock()
	stats := t.stats
	return &stats
}

// SetMode 设置模式
func (t *ARPTransport) SetMode(mode string) {
	t.mode = mode
}

// SetServerKeys 设置服务器密钥（服务端使用）
func (t *ARPTransport) SetServerKeys(privKey, pubKey []byte) {
	t.serverPrivKey = privKey
	t.serverPubKey = pubKey
}

// SetServerPublicKey 设置服务器公钥（客户端验证签名用）
func (t *ARPTransport) SetServerPublicKey(pubKey []byte) {
	t.serverPubKey = pubKey
}

// SetChannelKey 设置频道密钥
func (t *ARPTransport) SetChannelKey(key []byte) {
	t.channelKey = key
}

// TODO: 实现以下功能
// - 消息分块和重组
// - ACK确认机制
// - 重传队列
// - 与crypto.Manager集成（Ed25519签名）
// - 服务发现（可选的DISCOVER/ANNOUNCE帧）
// - 流量控制
