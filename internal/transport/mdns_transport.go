package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

// MDNSTransport mDNS传输层实现（服务器签名广播模式）
// 参考: docs/PROTOCOL.md - 4. mDNS传输协议
type MDNSTransport struct {
	config *Config
	mode   string // "server" or "client"

	// UDP连接
	conn *net.UDPConn

	// mDNS服务器
	mdnsServer *mdns.Server

	// 服务器信息
	serverAddr    *net.UDPAddr
	serverPubKey  []byte // Ed25519公钥（客户端验证签名用）
	serverPrivKey []byte // Ed25519私钥（服务端签名用）

	// 加密
	channelKey []byte // AES-256密钥

	// 频道信息
	channelID   string
	channelName string

	// 消息处理
	handler     MessageHandler
	fileHandler FileHandler

	// 消息重组器
	assembler *MessageAssembler

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

	// mDNS监听
	entriesCh chan *mdns.ServiceEntry
	closeCh   chan struct{}
}

// MessageAssembler 消息重组器
// 参考: docs/PROTOCOL.md - 4.3.4 消息重组器
type MessageAssembler struct {
	chunks map[string]*ChunkSet
	mutex  sync.Mutex
}

// ChunkSet 分块集合
type ChunkSet struct {
	data     map[int]string
	total    int
	lastSeen time.Time
}

// NewMDNSTransport 创建mDNS传输层
func NewMDNSTransport() *MDNSTransport {
	return &MDNSTransport{
		seenMsgs:  make(map[string]time.Time),
		assembler: NewMessageAssembler(),
		entriesCh: make(chan *mdns.ServiceEntry, 100),
		closeCh:   make(chan struct{}),
	}
}

// NewMessageAssembler 创建消息重组器
func NewMessageAssembler() *MessageAssembler {
	return &MessageAssembler{
		chunks: make(map[string]*ChunkSet),
	}
}

// ===== 生命周期管理 =====

// Init 初始化
func (t *MDNSTransport) Init(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	t.config = config
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.stats.StartTime = time.Now()

	return nil
}

// Start 启动传输层
func (t *MDNSTransport) Start() error {
	if t.started {
		return fmt.Errorf("transport already started")
	}

	if t.config == nil {
		return fmt.Errorf("transport not initialized")
	}

	// 创建UDP连接
	addr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0, // 随机端口
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	t.conn = conn

	t.started = true

	// 服务端模式：注册mDNS服务和启动接收循环
	if t.mode == "server" {
		if err := t.startServer(); err != nil {
			return err
		}
		go t.receiveFromClients()
	}

	// 启动消息监听（客户端和服务端都需要）
	go t.listenMessages()

	// 启动清理协程
	go t.cleanupLoop()

	fmt.Printf("mDNS transport started in %s mode\n", t.mode)
	return nil
}

// startServer 启动服务端
func (t *MDNSTransport) startServer() error {
	// 注册频道发现服务
	// 参考: docs/PROTOCOL.md - 4.2.1 服务注册
	port := t.conn.LocalAddr().(*net.UDPAddr).Port

	// 获取本地IP
	localIP := t.getLocalIP()

	// 编码服务器公钥
	pubKeyB64 := base64.StdEncoding.EncodeToString(t.serverPubKey)

	info := &mdns.MDNSService{
		Instance: t.channelName,
		Service:  "_crosswire._udp",
		Domain:   "local",
		HostName: "crosswire-server.local",
		Port:     port,
		IPs:      []net.IP{localIP},
		TXT: []string{
			"version=1",
			"protocol=server-signed",
			fmt.Sprintf("pubkey=%s", pubKeyB64),
			fmt.Sprintf("channel=%s", t.channelID[:8]), // 前8字符
		},
	}

	// 创建mDNS服务实例
	server, err := mdns.NewMDNSService(
		info.Instance,
		info.Service,
		info.Domain,
		info.HostName,
		info.Port,
		nil,
		info.TXT,
	)
	if err != nil {
		return fmt.Errorf("failed to create mDNS service: %w", err)
	}

	mdnsServer, err := mdns.NewServer(&mdns.Config{Zone: server})
	if err != nil {
		return fmt.Errorf("failed to start mDNS server: %w", err)
	}

	t.mdnsServer = mdnsServer

	fmt.Printf("mDNS server registered: %s (port %d)\n", t.channelName, port)
	return nil
}

// Stop 停止传输层
func (t *MDNSTransport) Stop() error {
	if !t.started {
		return nil
	}

	t.started = false
	t.cancel()
	close(t.closeCh)

	// 关闭mDNS服务器
	if t.mdnsServer != nil {
		t.mdnsServer.Shutdown()
	}

	// 关闭UDP连接
	if t.conn != nil {
		t.conn.Close()
	}

	return nil
}

// ===== 连接管理 =====

// Connect 连接到服务器（客户端模式）
func (t *MDNSTransport) Connect(target string) error {
	// target 格式: "IP:Port"
	t.mode = "client"

	addr, err := net.ResolveUDPAddr("udp4", target)
	if err != nil {
		return fmt.Errorf("invalid server address: %w", err)
	}
	t.serverAddr = addr

	t.connected = true

	fmt.Printf("Connected to mDNS server %s\n", target)
	return nil
}

// Disconnect 断开连接
func (t *MDNSTransport) Disconnect() error {
	t.connected = false
	t.serverAddr = nil
	return nil
}

// IsConnected 检查连接状态
func (t *MDNSTransport) IsConnected() bool {
	return t.connected
}

// ===== 消息收发 =====

// SendMessage 发送消息
// 参考: docs/PROTOCOL.md - 4.3 消息传输流程
func (t *MDNSTransport) SendMessage(msg *Message) error {
	if t.mode == "server" {
		// 服务器模式：签名并通过mDNS广播
		return t.signAndBroadcastViaMDNS(msg)
	} else {
		// 客户端模式：UDP单播给服务器
		return t.sendToServer(msg)
	}
}

// sendToServer 客户端单播给服务器
// 参考: docs/PROTOCOL.md - 4.3.1 客户端发送消息
func (t *MDNSTransport) sendToServer(msg *Message) error {
	if !t.connected {
		return fmt.Errorf("not connected to server")
	}

	// 发送加密的消息载荷
	_, err := t.conn.WriteToUDP(msg.Payload, t.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to send to server: %w", err)
	}

	// 更新统计
	t.statsMu.Lock()
	t.stats.BytesSent += uint64(len(msg.Payload))
	t.stats.MessagesSent++
	t.stats.LastActivity = time.Now()
	t.statsMu.Unlock()

	return nil
}

// receiveFromClients 服务器接收客户端消息
// 参考: docs/PROTOCOL.md - 4.3.2 服务器处理与签名
func (t *MDNSTransport) receiveFromClients() {
	buf := make([]byte, 4096)

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		n, clientAddr, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		// 构造消息
		msg := &Message{
			Payload:    buf[:n],
			Timestamp:  time.Now(),
			SenderAddr: clientAddr.String(),
		}

		// 更新统计
		t.statsMu.Lock()
		t.stats.BytesReceived += uint64(n)
		t.stats.MessagesRecv++
		t.stats.LastActivity = time.Now()
		t.statsMu.Unlock()

		// 调用处理函数（应该由Server层验证权限并调用signAndBroadcastViaMDNS）
		if t.handler != nil {
			go t.handler(msg)
		}
	}
}

// signAndBroadcastViaMDNS 服务器签名并通过mDNS广播
// 参考: docs/PROTOCOL.md - 4.3.3 服务实例名编码
func (t *MDNSTransport) signAndBroadcastViaMDNS(msg *Message) error {
	if t.mode != "server" {
		return fmt.Errorf("only server can broadcast via mDNS")
	}

	// 使用Ed25519签名
	// TODO: 集成crypto.Manager
	// signature := ed25519.Sign(t.serverPrivKey, msg.Payload)
	signature := []byte("TODO_SIGNATURE")

	// 构造签名载荷
	signedPayload := &SignedPayload{
		Message:   msg.Payload,
		Signature: signature,
		Timestamp: time.Now().UnixNano(),
	}

	payloadBytes, err := json.Marshal(signedPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal signed payload: %w", err)
	}

	// Base64URL编码
	encoded := base64.URLEncoding.EncodeToString(payloadBytes)

	// 分块（DNS标签最大63字符，我们用50保守）
	const chunkSize = 50
	chunks := splitIntoChunks(encoded, chunkSize)

	// 为每个块注册mDNS服务
	msgIDShort := msg.ID
	if len(msgIDShort) > 6 {
		msgIDShort = msgIDShort[:6]
	}

	for i, chunk := range chunks {
		instanceName := fmt.Sprintf("%s-%03d-%s", msgIDShort, i, chunk)

		info := &mdns.MDNSService{
			Instance: instanceName,
			Service:  "_crosswire-msg._udp",
			Domain:   "local",
			Port:     0,
			TXT: []string{
				fmt.Sprintf("total=%d", len(chunks)),
			},
		}

		// 创建mDNS服务实例
		server, err := mdns.NewMDNSService(
			info.Instance,
			info.Service,
			info.Domain,
			info.HostName,
			info.Port,
			nil,
			info.TXT,
		)
		if err != nil {
			continue
		}

		// 注册服务（5秒后自动注销）
		mdnsServer, err := mdns.NewServer(&mdns.Config{Zone: server})
		if err != nil {
			continue
		}

		// 自动清理
		time.AfterFunc(5*time.Second, func() {
			_ = mdnsServer.Shutdown()
		})
	}

	return nil
}

// listenMessages 监听mDNS消息服务
// 参考: docs/PROTOCOL.md - 4.3.4 客户端接收与验证签名
func (t *MDNSTransport) listenMessages() {
	// 持续查询消息服务
	go func() {
		for {
			select {
			case <-t.ctx.Done():
				return
			default:
			}

			// 查询消息服务
			params := &mdns.QueryParam{
				Service:             "_crosswire-msg._udp",
				Domain:              "local",
				Timeout:             1 * time.Second,
				Entries:             t.entriesCh,
				WantUnicastResponse: false,
			}

			mdns.Query(params)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 处理接收到的服务实例
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-t.closeCh:
			return
		case entry := <-t.entriesCh:
			t.handleMessageEntry(entry)
		}
	}
}

// handleMessageEntry 处理mDNS消息服务实例
func (t *MDNSTransport) handleMessageEntry(entry *mdns.ServiceEntry) {
	if entry == nil {
		return
	}

	// 解析服务实例名: <msgid>-<seq>-<data>
	parts := strings.Split(entry.Name, "-")
	if len(parts) < 3 {
		return
	}

	msgID := parts[0]
	seq, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}
	data := strings.Join(parts[2:], "-") // 数据可能包含"-"

	// 获取总分块数
	var total int
	for _, txt := range entry.InfoFields {
		if strings.HasPrefix(txt, "total=") {
			fmt.Sscanf(txt, "total=%d", &total)
			break
		}
	}

	// 添加到重组器
	if assembled := t.assembler.AddChunk(msgID, seq, data, total); assembled != "" {
		// 重组完成，验证并处理
		t.verifyAndProcess(assembled)
	}
}

// verifyAndProcess 验证签名并处理消息
func (t *MDNSTransport) verifyAndProcess(encodedPayload string) {
	// Base64URL解码
	payloadBytes, err := base64.URLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return
	}

	// 反序列化签名载荷
	var signedPayload SignedPayload
	if err := json.Unmarshal(payloadBytes, &signedPayload); err != nil {
		return
	}

	// 验证服务器签名
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
	msgID := fmt.Sprintf("%d", signedPayload.Timestamp)
	if t.hasSeenMessage(msgID) {
		return
	}
	t.markMessageAsSeen(msgID)

	// 构造消息
	msg := &Message{
		Payload:   signedPayload.Message,
		Timestamp: time.Unix(0, signedPayload.Timestamp),
	}

	// 调用处理函数
	if t.handler != nil {
		go t.handler(msg)
	}
}

// ReceiveMessage 接收消息（阻塞）- 不推荐使用，建议用Subscribe
func (t *MDNSTransport) ReceiveMessage() (*Message, error) {
	return nil, fmt.Errorf("use Subscribe instead")
}

// Subscribe 订阅消息（异步回调）
func (t *MDNSTransport) Subscribe(handler MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}
	t.handler = handler
	return nil
}

// Unsubscribe 取消订阅
func (t *MDNSTransport) Unsubscribe() {
	t.handler = nil
}

// ===== 辅助方法 =====

// hasSeenMessage 检查消息是否已处理
func (t *MDNSTransport) hasSeenMessage(msgID string) bool {
	t.seenMsgsMu.RLock()
	defer t.seenMsgsMu.RUnlock()
	_, exists := t.seenMsgs[msgID]
	return exists
}

// markMessageAsSeen 标记消息为已处理
func (t *MDNSTransport) markMessageAsSeen(msgID string) {
	t.seenMsgsMu.Lock()
	defer t.seenMsgsMu.Unlock()
	t.seenMsgs[msgID] = time.Now()
}

// validateTimestamp 验证时间戳
func (t *MDNSTransport) validateTimestamp(timestamp int64) bool {
	now := time.Now().UnixNano()
	diff := now - timestamp
	if diff < 0 {
		diff = -diff
	}
	// 容忍5分钟
	return diff < 5*60*1e9
}

// cleanupLoop 清理过期的seenMsgs
func (t *MDNSTransport) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.cleanupSeenMessages()
			t.assembler.Cleanup()
		}
	}
}

// cleanupSeenMessages 清理10分钟前的消息记录
func (t *MDNSTransport) cleanupSeenMessages() {
	t.seenMsgsMu.Lock()
	defer t.seenMsgsMu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for msgID, timestamp := range t.seenMsgs {
		if timestamp.Before(cutoff) {
			delete(t.seenMsgs, msgID)
		}
	}
}

// getLocalIP 获取本地IP
func (t *MDNSTransport) getLocalIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return net.IPv4(127, 0, 0, 1)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

// splitIntoChunks 分块
func splitIntoChunks(s string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

// ===== MessageAssembler方法 =====

// AddChunk 添加分块
func (a *MessageAssembler) AddChunk(msgID string, seq int, data string, total int) string {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, exists := a.chunks[msgID]; !exists {
		a.chunks[msgID] = &ChunkSet{
			data:     make(map[int]string),
			total:    total,
			lastSeen: time.Now(),
		}
	}

	set := a.chunks[msgID]
	set.data[seq] = data
	set.lastSeen = time.Now()

	// 更新总数（如果首次获取）
	if set.total == 0 && total > 0 {
		set.total = total
	}

	// 检查是否完整
	if a.isComplete(set) {
		assembled := a.assemble(set)
		delete(a.chunks, msgID)
		return assembled
	}

	return ""
}

// isComplete 检查分块是否完整
func (a *MessageAssembler) isComplete(set *ChunkSet) bool {
	if set.total == 0 {
		return false
	}
	return len(set.data) == set.total
}

// assemble 重组分块
func (a *MessageAssembler) assemble(set *ChunkSet) string {
	var assembled strings.Builder
	for i := 0; i < set.total; i++ {
		if chunk, exists := set.data[i]; exists {
			assembled.WriteString(chunk)
		}
	}
	return assembled.String()
}

// Cleanup 清理超时的分块
func (a *MessageAssembler) Cleanup() {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	for msgID, set := range a.chunks {
		if set.lastSeen.Before(cutoff) {
			delete(a.chunks, msgID)
		}
	}
}

// ===== 其他接口实现 =====

// SendFile 发送文件
func (t *MDNSTransport) SendFile(file *FileTransfer) error {
	// TODO: 实现文件传输
	return fmt.Errorf("not implemented")
}

// OnFileReceived 文件接收回调
func (t *MDNSTransport) OnFileReceived(handler FileHandler) error {
	t.fileHandler = handler
	return nil
}

// Discover 发现可用的服务端
// 参考: docs/PROTOCOL.md - 4.2.2 频道查询
func (t *MDNSTransport) Discover(timeout time.Duration) ([]*PeerInfo, error) {
	var peers []*PeerInfo
	entriesCh := make(chan *mdns.ServiceEntry, 10)

	// 查询CrossWire服务
	params := &mdns.QueryParam{
		Service:             "_crosswire._udp",
		Domain:              "local",
		Timeout:             timeout,
		Entries:             entriesCh,
		WantUnicastResponse: false,
	}

	go mdns.Query(params)

	// 收集结果
	deadline := time.After(timeout)
	for {
		select {
		case entry := <-entriesCh:
			if entry == nil {
				continue
			}

			// 解析服务器公钥和频道ID
			var channelHash string
			for _, txt := range entry.InfoFields {
				if strings.HasPrefix(txt, "pubkey=") {
					// pubKey = txt[7:] // TODO: 用于验证服务器签名
				} else if strings.HasPrefix(txt, "channel=") {
					channelHash = txt[8:]
				}
			}

			peer := &PeerInfo{
				ID:            entry.Name,
				Address:       fmt.Sprintf("%s:%d", entry.AddrV4, entry.Port),
				Mode:          TransportModeMDNS,
				LastSeen:      time.Now(),
				ChannelIDHash: channelHash,
				Version:       1,
			}
			peers = append(peers, peer)

		case <-deadline:
			return peers, nil
		}
	}
}

// Announce 宣告服务
func (t *MDNSTransport) Announce(info *ServiceInfo) error {
	// 在Start()中已实现
	t.channelID = info.ChannelID
	t.channelName = info.ChannelName
	return nil
}

// GetMode 获取传输模式
func (t *MDNSTransport) GetMode() TransportMode {
	return TransportModeMDNS
}

// GetStats 获取传输统计
func (t *MDNSTransport) GetStats() *TransportStats {
	t.statsMu.RLock()
	defer t.statsMu.RUnlock()
	stats := t.stats
	return &stats
}

// SetMode 设置模式
func (t *MDNSTransport) SetMode(mode string) {
	t.mode = mode
}

// SetServerKeys 设置服务器密钥（服务端使用）
func (t *MDNSTransport) SetServerKeys(privKey, pubKey []byte) {
	t.serverPrivKey = privKey
	t.serverPubKey = pubKey
}

// SetServerPublicKey 设置服务器公钥（客户端验证签名用）
func (t *MDNSTransport) SetServerPublicKey(pubKey []byte) {
	t.serverPubKey = pubKey
}

// SetChannelKey 设置频道密钥
func (t *MDNSTransport) SetChannelKey(key []byte) {
	t.channelKey = key
}

// SetChannelInfo 设置频道信息
func (t *MDNSTransport) SetChannelInfo(id, name string) {
	t.channelID = id
	t.channelName = name
}

// TODO: 实现以下功能
// - 与crypto.Manager集成（Ed25519签名验证）
// - 文件传输
// - 更高效的消息编码
// - 流量控制
