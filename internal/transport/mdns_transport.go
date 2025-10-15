package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/miekg/dns"
)

// MDNSTransport mDNS传输层实现（纯宣告包模式）
// 使用DNS Response包（Announcement）进行所有通信
// 参考: docs/PROTOCOL.md - 4. mDNS传输协议
type MDNSTransport struct {
	config *Config
	mode   string // "server" or "client"

	// UDP连接
	conn *net.UDPConn

	// mDNS服务器（仅用于服务发现）
	mdnsServer *mdns.Server

	// 服务器信息
	serverAddr    *net.UDPAddr
	serverPubKey  []byte // Ed25519公钥（验证签名用）
	serverPrivKey []byte // Ed25519私钥（签名用）

	// 加密
	channelKey []byte // AES-256密钥

	// 频道信息
	channelID   string
	channelName string
	hostname    string // 本机主机名
	localIP     net.IP // 本机IP

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

	// mDNS监听（仅用于服务发现）
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

	// 初始化本机信息
	t.localIP = t.getLocalIP()
	t.hostname = fmt.Sprintf("crosswire-%s", generateShortID())

	// 创建UDP连接（监听5353端口）
	addr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 5353,
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	t.conn = conn

	t.started = true

	// 服务端模式：注册mDNS服务发现
	if t.mode == "server" {
		if err := t.startServer(); err != nil {
			return err
		}
	}

	// 启动宣告包接收循环（客户端和服务端都需要）
	go t.receiveAnnouncements()

	// 启动清理协程
	go t.cleanupLoop()

	fmt.Printf("mDNS transport started in %s mode (IP: %s)\n", t.mode, t.localIP)
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

// SendMessage 发送消息（纯宣告包模式）
// 参考: docs/PROTOCOL.md - 4.3 消息传输流程
func (t *MDNSTransport) SendMessage(msg *Message) error {
	if t.mode == "server" {
		// 服务器模式：多播宣告给所有客户端
		return t.sendAnnouncement(msg, true)
	} else {
		// 客户端模式：广播宣告（因为可能不知道服务器地址）
		return t.sendAnnouncement(msg, true)
	}
}

// sendAnnouncement 发送宣告包（核心方法）
func (t *MDNSTransport) sendAnnouncement(msg *Message, multicast bool) error {
	// 创建宣告包
	announcement := t.createAnnouncementPacket(msg.Payload)

	// 序列化
	packet, err := announcement.Pack()
	if err != nil {
		return fmt.Errorf("failed to pack announcement: %w", err)
	}

	// 发送
	var targetAddr *net.UDPAddr
	if multicast {
		// 多播到 224.0.0.251:5353
		targetAddr, _ = net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	} else if t.serverAddr != nil {
		// 单播到服务器
		targetAddr = t.serverAddr
	} else {
		// 默认多播
		targetAddr, _ = net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	}

	_, err = t.conn.WriteToUDP(packet, targetAddr)
	if err != nil {
		return fmt.Errorf("failed to send announcement: %w", err)
	}

	// 更新统计
	t.statsMu.Lock()
	t.stats.BytesSent += uint64(len(packet))
	t.stats.MessagesSent++
	t.stats.LastActivity = time.Now()
	t.statsMu.Unlock()

	return nil
}

// createAnnouncementPacket 创建宣告包
func (t *MDNSTransport) createAnnouncementPacket(data []byte) *dns.Msg {
	msg := new(dns.Msg)
	msg.Response = true      // QR=1 (Response)
	msg.Authoritative = true // AA=1 (Authoritative)
	msg.RecursionDesired = false

	msgID := generateMessageID()
	serviceName := fmt.Sprintf("%s.msg._crosswire._tcp.local.", msgID)

	// 1. PTR记录（服务类型）
	ptr := &dns.PTR{
		Hdr: dns.RR_Header{
			Name:   "_crosswire._tcp.local.",
			Rrtype: dns.TypePTR,
			Class:  dns.ClassINET,
			Ttl:    10, // 短TTL，消息快速过期
		},
		Ptr: serviceName,
	}
	msg.Answer = append(msg.Answer, ptr)

	// 2. SRV记录（发送者信息）
	srv := &dns.SRV{
		Hdr: dns.RR_Header{
			Name:   serviceName,
			Rrtype: dns.TypeSRV,
			Class:  dns.ClassINET,
			Ttl:    10,
		},
		Priority: 0,
		Weight:   0,
		Port:     5353,
		Target:   dns.Fqdn(t.hostname + ".local"),
	}
	msg.Answer = append(msg.Answer, srv)

	// 3. A记录（发送者IP）
	a := &dns.A{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(t.hostname + ".local"),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    10,
		},
		A: t.localIP,
	}
	msg.Answer = append(msg.Answer, a)

	// 4. TXT记录-元数据
	metaTXT := &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   serviceName,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    10,
		},
		Txt: []string{
			"version=1.0",
			fmt.Sprintf("msgid=%s", msgID),
			fmt.Sprintf("ts=%d", time.Now().Unix()),
			fmt.Sprintf("size=%d", len(data)),
		},
	}
	msg.Answer = append(msg.Answer, metaTXT)

	// 5-N. TXT记录-数据载荷
	chunks := chunkData(data, 240) // 每块240字节
	for i, chunk := range chunks {
		dataTXT := &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   dns.Fqdn(fmt.Sprintf("data%d.%s.local", i, msgID)),
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    10,
			},
			Txt: []string{base64.StdEncoding.EncodeToString(chunk)},
		}
		msg.Answer = append(msg.Answer, dataTXT)
	}

	// 添加签名（如果有私钥）
	if len(t.serverPrivKey) == ed25519.PrivateKeySize {
		t.addSignature(msg)
	}

	return msg
}

// addSignature 添加签名到宣告包
func (t *MDNSTransport) addSignature(msg *dns.Msg) {
	// 计算Answer Section的哈希
	var data []byte
	for _, rr := range msg.Answer {
		// 将RR序列化为字符串
		rrStr := rr.String()
		data = append(data, []byte(rrStr)...)
	}
	hash := sha256.Sum256(data)

	// Ed25519签名
	signature := ed25519.Sign(ed25519.PrivateKey(t.serverPrivKey), hash[:])

	// 添加签名记录到Additional Section
	sigTXT := &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   "_sig.local.",
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    0,
		},
		Txt: []string{
			fmt.Sprintf("sig=%s", base64.StdEncoding.EncodeToString(signature)),
			fmt.Sprintf("ts=%d", time.Now().Unix()),
		},
	}
	msg.Extra = append(msg.Extra, sigTXT)
}

// receiveAnnouncements 接收宣告包
func (t *MDNSTransport) receiveAnnouncements() {
	buf := make([]byte, 2048)

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		n, addr, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		// 解析DNS消息
		msg := new(dns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil {
			continue
		}

		// 只处理响应包（宣告包）
		if !msg.Response {
			continue
		}

		// 过滤非CrossWire宣告
		if !t.isCrossWireAnnouncement(msg) {
			continue
		}

		// 提取数据
		data, metadata := t.extractDataFromAnnouncement(msg)
		if len(data) == 0 {
			continue
		}

		// 检查是否已处理（防重放）
		msgID := metadata["msgid"]
		if t.hasSeenMessage(msgID) {
			continue
		}
		t.markMessageAsSeen(msgID)

		// 验证签名
		if len(t.serverPubKey) == ed25519.PublicKeySize {
			if !t.verifySignature(msg) {
				fmt.Println("Invalid signature, possible attack!")
				continue
			}
		}

		// 更新统计
		t.statsMu.Lock()
		t.stats.BytesReceived += uint64(n)
		t.stats.MessagesRecv++
		t.stats.LastActivity = time.Now()
		t.statsMu.Unlock()

		// 构造消息
		message := &Message{
			Payload:    data,
			SenderAddr: addr.String(),
			Timestamp:  time.Now(),
		}

		// 调用处理函数
		if t.handler != nil {
			go t.handler(message)
		}
	}
}

// isCrossWireAnnouncement 判断是否是CrossWire宣告
func (t *MDNSTransport) isCrossWireAnnouncement(msg *dns.Msg) bool {
	for _, rr := range msg.Answer {
		if ptr, ok := rr.(*dns.PTR); ok {
			if strings.Contains(ptr.Hdr.Name, "_crosswire") {
				return true
			}
		}
	}
	return false
}

// extractDataFromAnnouncement 从宣告包提取数据
func (t *MDNSTransport) extractDataFromAnnouncement(msg *dns.Msg) ([]byte, map[string]string) {
	metadata := make(map[string]string)
	var dataChunks []string

	for _, rr := range msg.Answer {
		if txt, ok := rr.(*dns.TXT); ok {
			// 判断是元数据还是数据
			if strings.Contains(txt.Hdr.Name, "data") {
				// 数据块
				if len(txt.Txt) > 0 {
					dataChunks = append(dataChunks, txt.Txt[0])
				}
			} else {
				// 元数据
				for _, field := range txt.Txt {
					parts := strings.SplitN(field, "=", 2)
					if len(parts) == 2 {
						metadata[parts[0]] = parts[1]
					}
				}
			}
		}
	}

	// 重组数据
	var allData []byte
	for _, chunk := range dataChunks {
		decoded, _ := base64.StdEncoding.DecodeString(chunk)
		allData = append(allData, decoded...)
	}

	return allData, metadata
}

// verifySignature 验证签名
func (t *MDNSTransport) verifySignature(msg *dns.Msg) bool {
	// 提取签名
	var signature []byte
	var timestamp int64

	for _, rr := range msg.Extra {
		if txt, ok := rr.(*dns.TXT); ok {
			if txt.Hdr.Name == "_sig.local." {
				for _, field := range txt.Txt {
					if strings.HasPrefix(field, "sig=") {
						sig, _ := base64.StdEncoding.DecodeString(field[4:])
						signature = sig
					} else if strings.HasPrefix(field, "ts=") {
						fmt.Sscanf(field, "ts=%d", &timestamp)
					}
				}
			}
		}
	}

	// 验证时间戳
	if time.Now().Unix()-timestamp > 300 {
		return false // 超过5分钟
	}

	// 计算哈希（与签名时使用相同方法）
	var data []byte
	for _, rr := range msg.Answer {
		rrStr := rr.String()
		data = append(data, []byte(rrStr)...)
	}
	hash := sha256.Sum256(data)

	// 验证签名
	return ed25519.Verify(ed25519.PublicKey(t.serverPubKey), hash[:], signature)
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

// chunkData 将数据分块
func chunkData(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

// generateMessageID 生成消息ID
func generateMessageID() string {
	data := []byte(fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63()))
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:4]) // 8字符
}

// generateShortID 生成短ID
func generateShortID() string {
	data := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:3]) // 6字符
}

// generateSessionID 生成会话ID
func generateSessionID() string {
	data := []byte(fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63()))
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:6]) // 12字符
}

// splitIntoChunks 字符串分块（保留用于兼容）
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
	if file == nil {
		return fmt.Errorf("file is nil")
	}
	// 复用消息通道：封装为文件型负载并调用 SendMessage
	payload, err := buildTransportFilePayload(file)
	if err != nil {
		return err
	}
	msg := &Message{Type: MessageTypeData, Payload: payload}
	return t.SendMessage(msg)
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
