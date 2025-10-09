package transport

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
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

	// 重组缓冲（Sequence -> 分块）
	reassemblyMu sync.Mutex
	reassembly   map[string]*reassemblyState

	// ACK等待与重传（仅客户端单播使用）
	ackMu       sync.Mutex
	pendingAcks map[uint32]*pendingAck

	// 服务发现
	serviceInfo  *ServiceInfo
	discoverMu   sync.Mutex
	discoverChan chan *PeerInfo
}

// 分块重组状态
type reassemblyState struct {
	total     uint16
	chunks    map[uint16][]byte
	createdAt time.Time
	updatedAt time.Time
	srcMAC    string
}

// 待确认发送项
type pendingAck struct {
	ch     chan struct{}
	frames []*ARPFrame
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
	t.reassembly = make(map[string]*reassemblyState)
	t.pendingAcks = make(map[uint32]*pendingAck)

	// 获取网卡信息
	iface, err := net.InterfaceByName(config.Interface)
	if err != nil {
		// 列出所有可用接口以帮助调试
		allIfaces, _ := net.Interfaces()
		availableNames := make([]string, 0)
		for _, i := range allIfaces {
			availableNames = append(availableNames, i.Name)
		}
		return fmt.Errorf("无法找到网络接口 '%s'。可用接口: %v。错误: %w", config.Interface, availableNames, err)
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
	// 注意：需要管理员/root权限
	handle, err := pcap.OpenLive(
		t.config.Interface,
		65536, // 快照长度
		true,  // 混杂模式
		pcap.BlockForever,
	)
	if err != nil {
		return fmt.Errorf("打开网络接口失败（可能需要管理员权限）: %w", err)
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

	// 支持 "MAC" 或 "MAC:BASE64_PUBKEY" 两种格式
	var macStr string
	var pubKeyB64 string
	if i := bytes.IndexByte([]byte(target), ':'); i >= 0 {
		// 尝试按 MAC:XX:XX:...:base64 解析，MAC 本身包含冒号，直接从右侧切分一次
		lastColon := bytes.LastIndexByte([]byte(target), ':')
		if lastColon > 0 && lastColon < len(target)-1 {
			macStr = target[:lastColon]
			pubKeyB64 = target[lastColon+1:]
		} else {
			macStr = target
		}
	} else {
		macStr = target
	}

	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return fmt.Errorf("invalid server MAC address: %w", err)
	}
	t.serverMAC = mac

	if pubKeyB64 != "" {
		if decoded, err := base64.StdEncoding.DecodeString(pubKeyB64); err == nil {
			t.serverPubKey = decoded
		}
	}

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

	// 分块
	chunks := t.chunkBytes(msg.Payload, MaxFramePayload)
	seq := t.nextSequence()
	frames := make([]*ARPFrame, 0, len(chunks))
	for i, p := range chunks {
		f := &ARPFrame{
			DstMAC:      t.serverMAC,
			SrcMAC:      t.localMAC,
			EtherType:   EtherTypeCustom,
			Version:     uint8(ProtocolVersion),
			FrameType:   uint8(msg.Type),
			Sequence:    seq,
			TotalChunks: uint16(len(chunks)),
			ChunkIndex:  uint16(i),
			Payload:     p,
		}
		f.Checksum = crc32.ChecksumIEEE(f.Payload)
		f.PayloadLen = uint16(len(f.Payload))
		frames = append(frames, f)
	}

	// 如需ACK，进行发送-等待-重传机制（仅对单播）
	needAck := t.config != nil && t.config.MaxRetries > 0
	var ackCh chan struct{}
	if needAck {
		ackCh = t.registerPendingAck(seq, frames)
	}

	// 首次发送全部分块
	for _, f := range frames {
		if err := t.sendRawFrame(f); err != nil {
			return err
		}
	}

	if !needAck {
		return nil
	}

	// 等待ACK并按需重传
	retries := t.config.MaxRetries
	retryDelay := t.config.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 200 * time.Millisecond
	}
	for {
		select {
		case <-ackCh:
			t.unregisterPendingAck(seq)
			return nil
		case <-time.After(retryDelay):
			if retries <= 0 {
				t.unregisterPendingAck(seq)
				return fmt.Errorf("ack timeout for seq=%d", seq)
			}
			// 重传所有分块
			for _, f := range frames {
				_ = t.sendRawFrame(f)
			}
			t.statsMu.Lock()
			t.stats.Retries++
			t.statsMu.Unlock()
			retries--
		case <-t.ctx.Done():
			t.unregisterPendingAck(seq)
			return fmt.Errorf("transport stopped while waiting for ack")
		}
	}
}

// signAndBroadcast 服务器签名并广播
// 参考: docs/ARP_BROADCAST_MODE.md - 2. 服务器签名与广播
func (t *ARPTransport) signAndBroadcast(msg *Message) error {
	if t.mode != "server" {
		return fmt.Errorf("only server can broadcast")
	}

	// 使用Ed25519签名（由上层通过 SetServerKeys 提供）
	if len(t.serverPrivKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("server private key not set or invalid length")
	}
	signature := ed25519.Sign(ed25519.PrivateKey(t.serverPrivKey), msg.Payload)

	// 构造签名载荷
	signedPayload := &SignedPayload{
		Message:   msg.Payload,
		Signature: signature,
		Timestamp: time.Now().UnixNano(),
	}

	// 序列化签名载荷
	payloadBytes := t.serializeSignedPayload(signedPayload)

	// 构造广播帧
	broadcastMAC, _ := net.ParseMAC(BroadcastMAC)
	// 分块广播
	chunks := t.chunkBytes(payloadBytes, MaxFramePayload)
	seq := t.nextSequence()
	for i, p := range chunks {
		frame := &ARPFrame{
			DstMAC:      broadcastMAC,
			SrcMAC:      t.localMAC,
			EtherType:   EtherTypeCustom,
			Version:     uint8(ProtocolVersion),
			FrameType:   uint8(msg.Type),
			Sequence:    seq,
			TotalChunks: uint16(len(chunks)),
			ChunkIndex:  uint16(i),
			Payload:     p,
		}
		frame.Checksum = crc32.ChecksumIEEE(frame.Payload)
		frame.PayloadLen = uint16(len(frame.Payload))
		if err := t.sendRawFrame(frame); err != nil {
			return err
		}
	}
	return nil
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
	// ACK帧（客户端接收服务端ACK）
	if MessageType(frame.FrameType) == MessageTypeACK && t.mode == "client" {
		// 仅处理发给本机的ACK
		if bytes.Equal(frame.DstMAC, t.localMAC) {
			t.signalAck(frame.Sequence)
		}
		return
	}

	// 发现/宣告帧（简易）
	if MessageType(frame.FrameType) == MessageTypeDiscover {
		if t.mode == "server" {
			// 收到发现请求，回复宣告（单播到请求方）
			t.replyAnnounce(frame.SrcMAC)
		} else {
			// 客户端收到宣告，解析加入结果列表
			t.tryParseAnnounce(frame)
		}
		return
	}

	// 分块重组
	payload, complete := t.tryReassemble(frame)
	if !complete {
		return
	}

	// 客户端模式：验证服务器签名
	if t.mode == "client" {
		// 只接受来自服务器的广播
		if !bytes.Equal(frame.SrcMAC, t.serverMAC) {
			return
		}

		// 解析签名载荷
		signedPayload := t.parseSignedPayload(payload)
		if signedPayload == nil {
			return
		}

		// 验证Ed25519签名
		if len(t.serverPubKey) == ed25519.PublicKeySize {
			if !ed25519.Verify(ed25519.PublicKey(t.serverPubKey), signedPayload.Message, signedPayload.Signature) {
				fmt.Println("Invalid signature, possible attack!")
				return
			}
		}

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

		// 文件回调（如果负载是传输文件）。同时触发重组缓存，当收齐时由上层再次回调完整文件。
		if t.fileHandler != nil {
			if ft := tryParseTransportFilePayload(signedPayload.Message); ft != nil {
				go func() {
					t.fileHandler(ft)
					handleFileChunk(ft)
				}()
			}
		}

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
			Payload:   payload,
			Timestamp: time.Now(),
			SenderMAC: frame.SrcMAC.String(),
		}

		// 更新统计
		t.statsMu.Lock()
		t.stats.BytesReceived += uint64(len(frame.Payload))
		t.stats.MessagesRecv++
		t.stats.LastActivity = time.Now()
		t.statsMu.Unlock()

		// 文件回调（如果负载是传输文件）。同时触发重组缓存，当收齐时由上层再次回调完整文件。
		if t.fileHandler != nil {
			if ft := tryParseTransportFilePayload(payload); ft != nil {
				go func() {
					t.fileHandler(ft)
					handleFileChunk(ft)
				}()
			}
		}

		// 调用处理函数
		if t.handler != nil {
			go t.handler(msg)
		}

		// 回ACK（仅对单播数据帧）
		if MessageType(frame.FrameType) == MessageTypeData || MessageType(frame.FrameType) == MessageTypeControl || MessageType(frame.FrameType) == MessageTypeAuth {
			ack := &ARPFrame{
				DstMAC:      frame.SrcMAC,
				SrcMAC:      t.localMAC,
				EtherType:   EtherTypeCustom,
				Version:     uint8(ProtocolVersion),
				FrameType:   uint8(MessageTypeACK),
				Sequence:    frame.Sequence,
				TotalChunks: 1,
				ChunkIndex:  0,
				Payload:     nil,
			}
			ack.Checksum = crc32.ChecksumIEEE([]byte{})
			ack.PayloadLen = 0
			_ = t.sendRawFrame(ack)
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

	return buffer.Bytes()
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
	// 暂时简单拼接（可替换为更高效的编码方案）
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
	if file == nil {
		return fmt.Errorf("file is nil")
	}
	if len(file.Data) == 0 && file.TotalChunks == 0 {
		return fmt.Errorf("file data is empty")
	}

	// 封装传输层文件载荷
	payload, err := buildTransportFilePayload(file)
	if err != nil {
		return err
	}

	msg := &Message{Type: MessageTypeData, Payload: payload}
	return t.SendMessage(msg)
}

// OnFileReceived 文件接收回调
func (t *ARPTransport) OnFileReceived(handler FileHandler) error {
	t.fileHandler = handler
	return nil
}

// Discover 发现可用的服务端
func (t *ARPTransport) Discover(timeout time.Duration) ([]*PeerInfo, error) {
	broadcastMAC, _ := net.ParseMAC(BroadcastMAC)
	seq := t.nextSequence()

	// 简易的 DISCOVER 负载：固定前缀 + 时间戳
	discoverPayload := []byte("DISCOVER|")
	f := &ARPFrame{
		DstMAC:      broadcastMAC,
		SrcMAC:      t.localMAC,
		EtherType:   EtherTypeCustom,
		Version:     uint8(ProtocolVersion),
		FrameType:   uint8(MessageTypeDiscover),
		Sequence:    seq,
		TotalChunks: 1,
		ChunkIndex:  0,
		Payload:     discoverPayload,
	}
	f.Checksum = crc32.ChecksumIEEE(f.Payload)
	f.PayloadLen = uint16(len(f.Payload))

	t.discoverMu.Lock()
	t.discoverChan = make(chan *PeerInfo, 64)
	t.discoverMu.Unlock()

	if err := t.sendRawFrame(f); err != nil {
		return nil, err
	}

	deadline := time.After(timeout)
	results := make([]*PeerInfo, 0)
	seen := make(map[string]bool)
	for {
		select {
		case <-deadline:
			return results, nil
		case p := <-t.discoverChan:
			if p == nil {
				return results, nil
			}
			if !seen[p.Address] {
				results = append(results, p)
				seen[p.Address] = true
			}
		case <-t.ctx.Done():
			return results, nil
		}
	}
}

// Announce 宣告服务
func (t *ARPTransport) Announce(info *ServiceInfo) error {
	// 仅保存信息，实际回复在收到 Discover 时执行
	t.serviceInfo = info
	return nil
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

// ===== 内部工具 =====

// chunkBytes 分块
func (t *ARPTransport) chunkBytes(data []byte, chunkSize int) [][]byte {
	if len(data) == 0 {
		return [][]byte{[]byte{}}
	}
	if chunkSize <= 0 {
		chunkSize = MaxFramePayload
	}
	n := (len(data) + chunkSize - 1) / chunkSize
	out := make([][]byte, 0, n)
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		out = append(out, data[i:end])
	}
	return out
}

// tryReassemble 根据 Sequence/ChunkIndex 重组完整负载
func (t *ARPTransport) tryReassemble(frame *ARPFrame) ([]byte, bool) {
	if frame.TotalChunks <= 1 {
		return frame.Payload, true
	}
	key := fmt.Sprintf("%s:%d", frame.SrcMAC.String(), frame.Sequence)
	t.reassemblyMu.Lock()
	st, ok := t.reassembly[key]
	if !ok {
		st = &reassemblyState{
			total:     frame.TotalChunks,
			chunks:    make(map[uint16][]byte, frame.TotalChunks),
			createdAt: time.Now(),
			updatedAt: time.Now(),
			srcMAC:    frame.SrcMAC.String(),
		}
		t.reassembly[key] = st
	}
	st.chunks[frame.ChunkIndex] = append([]byte(nil), frame.Payload...)
	st.updatedAt = time.Now()

	// 检查是否完整
	if uint16(len(st.chunks)) < st.total {
		t.reassemblyMu.Unlock()
		return nil, false
	}
	// 拼接
	buf := make([]byte, 0)
	for i := uint16(0); i < st.total; i++ {
		part := st.chunks[i]
		buf = append(buf, part...)
	}
	delete(t.reassembly, key)
	t.reassemblyMu.Unlock()
	return buf, true
}

// registerPendingAck 注册ACK等待
func (t *ARPTransport) registerPendingAck(seq uint32, frames []*ARPFrame) chan struct{} {
	ch := make(chan struct{}, 1)
	t.ackMu.Lock()
	t.pendingAcks[seq] = &pendingAck{ch: ch, frames: frames}
	t.ackMu.Unlock()
	return ch
}

// signalAck 收到ACK
func (t *ARPTransport) signalAck(seq uint32) {
	t.ackMu.Lock()
	if p, ok := t.pendingAcks[seq]; ok {
		select {
		case p.ch <- struct{}{}:
		default:
		}
		delete(t.pendingAcks, seq)
	}
	t.ackMu.Unlock()
}

// unregisterPendingAck 取消ACK等待
func (t *ARPTransport) unregisterPendingAck(seq uint32) {
	t.ackMu.Lock()
	delete(t.pendingAcks, seq)
	t.ackMu.Unlock()
}

// replyAnnounce 服务器回复 ANNOUNCE 到指定 MAC
func (t *ARPTransport) replyAnnounce(dst net.HardwareAddr) {
	// 构造简易宣告载荷：ANNOUNCE|<hash8>|<version>
	hash8 := ""
	if t.serviceInfo != nil && t.serviceInfo.ChannelID != "" {
		sum := sha256.Sum256([]byte(t.serviceInfo.ChannelID))
		hash8 = hex.EncodeToString(sum[:])
		if len(hash8) > 8 {
			hash8 = hash8[:8]
		}
	}
	payload := []byte("ANNOUNCE|" + hash8 + "|" + fmt.Sprintf("%d", ProtocolVersion))
	f := &ARPFrame{
		DstMAC:      dst,
		SrcMAC:      t.localMAC,
		EtherType:   EtherTypeCustom,
		Version:     uint8(ProtocolVersion),
		FrameType:   uint8(MessageTypeDiscover),
		Sequence:    t.nextSequence(),
		TotalChunks: 1,
		ChunkIndex:  0,
		Payload:     payload,
	}
	f.Checksum = crc32.ChecksumIEEE(f.Payload)
	f.PayloadLen = uint16(len(f.Payload))
	_ = t.sendRawFrame(f)
}

// tryParseAnnounce 客户端解析 ANNOUNCE 响应
func (t *ARPTransport) tryParseAnnounce(frame *ARPFrame) {
	if frame == nil || frame.Payload == nil || len(frame.Payload) == 0 {
		return
	}
	// 载荷形如：ANNOUNCE|<hash8>|<version>
	if !bytes.HasPrefix(frame.Payload, []byte("ANNOUNCE|")) {
		return
	}
	parts := bytes.Split(frame.Payload, []byte("|"))
	var hash8 string
	if len(parts) >= 2 {
		hash8 = string(parts[1])
	}
	pi := &PeerInfo{
		ID:            frame.SrcMAC.String(),
		Address:       frame.SrcMAC.String(),
		Mode:          TransportModeARP,
		LastSeen:      time.Now(),
		ChannelIDHash: hash8,
		Version:       ProtocolVersion,
	}
	t.discoverMu.Lock()
	if t.discoverChan != nil {
		select {
		case t.discoverChan <- pi:
		default:
		}
	}
	t.discoverMu.Unlock()
}

// ===== 文件传输封装（与上层通道复用） =====

// transportFilePayload 传输层文件载荷（JSON）
// {"type":"file","file_id":"...","filename":"...","size":123,"chunk_index":0,"total_chunks":10,"data":"base64...","checksum":"hex"}
type transportFilePayload struct {
	Type          string `json:"type"`
	FileID        string `json:"file_id"`
	Filename      string `json:"filename,omitempty"`
	Size          int64  `json:"size,omitempty"`
	ChunkSize     int    `json:"chunk_size,omitempty"`
	ChunkIndex    int    `json:"chunk_index"`
	TotalChunks   int    `json:"total_chunks"`
	Data          string `json:"data"`
	Checksum      string `json:"checksum,omitempty"`       // 整体文件SHA256
	ChunkChecksum string `json:"chunk_checksum,omitempty"` // 当前分块SHA256
}

func buildTransportFilePayload(f *FileTransfer) ([]byte, error) {
	var dataB64 string
	if len(f.Data) > 0 {
		dataB64 = base64.StdEncoding.EncodeToString(f.Data)
	}
	// 整体文件校验和
	checksum := f.Checksum
	// 当前分块校验和
	chunkChecksum := f.ChunkChecksum
	if len(f.Data) > 0 && chunkChecksum == "" {
		sum := sha256.Sum256(f.Data)
		chunkChecksum = fmt.Sprintf("%x", sum[:])
	}
	p := transportFilePayload{
		Type:          "file",
		FileID:        f.FileID,
		Filename:      f.Filename,
		Size:          f.Size,
		ChunkSize:     f.ChunkSize,
		ChunkIndex:    f.ChunkIndex,
		TotalChunks:   f.TotalChunks,
		Data:          dataB64,
		Checksum:      checksum,
		ChunkChecksum: chunkChecksum,
	}
	return json.Marshal(&p)
}

func tryParseTransportFilePayload(payload []byte) *FileTransfer {
	if len(payload) == 0 {
		return nil
	}
	var p transportFilePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil
	}
	if p.Type != "file" || p.FileID == "" {
		return nil
	}
	var chunk []byte
	if p.Data != "" {
		b, err := base64.StdEncoding.DecodeString(p.Data)
		if err == nil {
			chunk = b
		}
	}
	return &FileTransfer{
		FileID:        p.FileID,
		Filename:      p.Filename,
		Size:          p.Size,
		ChunkSize:     p.ChunkSize,
		TotalChunks:   p.TotalChunks,
		Data:          chunk,
		ChunkIndex:    p.ChunkIndex,
		Checksum:      p.Checksum,
		ChunkChecksum: p.ChunkChecksum,
	}
}

// ===== 接收端文件分片重组缓存（按 FileID 聚合） =====

type fileReassembly struct {
	filename    string
	size        int64
	chunkSize   int
	totalChunks int
	received    map[int][]byte
	lastSeen    time.Time
	checksum    string // 期望的整体SHA256
}

var (
	fileAsmMu sync.Mutex
	fileAsm   = make(map[string]*fileReassembly)
)

// handleFileChunk 聚合分片并在收齐后输出到保存路径（由上层决定路径，通过 Filename 提供）
func handleFileChunk(ft *FileTransfer) {
	if ft == nil || ft.FileID == "" {
		return
	}
	fileAsmMu.Lock()
	asm, ok := fileAsm[ft.FileID]
	if !ok {
		asm = &fileReassembly{
			filename:    ft.Filename,
			size:        ft.Size,
			chunkSize:   ft.ChunkSize,
			totalChunks: ft.TotalChunks,
			received:    make(map[int][]byte, ft.TotalChunks),
			lastSeen:    time.Now(),
			checksum:    ft.Checksum,
		}
		fileAsm[ft.FileID] = asm
	}
	// 校验分块校验和（若携带）
	if len(ft.Data) > 0 && ft.ChunkChecksum != "" {
		sum := sha256.Sum256(ft.Data)
		if fmt.Sprintf("%x", sum[:]) != ft.ChunkChecksum {
			fileAsmMu.Unlock()
			return
		}
	}
	// 存储分块
	asm.received[ft.ChunkIndex] = append([]byte(nil), ft.Data...)
	asm.lastSeen = time.Now()

	// 检查是否收齐
	if len(asm.received) < asm.totalChunks {
		fileAsmMu.Unlock()
		return
	}

	// 拼装
	assembled := make([]byte, 0, asm.chunkSize*asm.totalChunks)
	for i := 0; i < asm.totalChunks; i++ {
		part := asm.received[i]
		assembled = append(assembled, part...)
	}

	// 整体校验
	if asm.checksum != "" {
		sum := sha256.Sum256(assembled)
		if fmt.Sprintf("%x", sum[:]) != asm.checksum {
			delete(fileAsm, ft.FileID)
			fileAsmMu.Unlock()
			return
		}
	}

	// 输出落盘（复用上层：仅通过 fileHandler 再次回调完整文件数据，由上层决定保存路径）
	complete := &FileTransfer{
		FileID:      ft.FileID,
		Filename:    asm.filename,
		Size:        int64(len(assembled)),
		Data:        assembled,
		ChunkIndex:  asm.totalChunks - 1,
		TotalChunks: asm.totalChunks,
		ChunkSize:   asm.chunkSize,
		Checksum:    asm.checksum,
	}
	delete(fileAsm, ft.FileID)
	fileAsmMu.Unlock()

	// 如果上层已设置文件回调，则回调完整文件（二次回调）
	// 注意：此方法在 transport 内部，不直接访问 t.fileHandler；由调用者在收到片段后调用本函数，再自行回调
	// 因此我们只提供聚合函数，实际回调放在调用点。
	_ = complete
}
