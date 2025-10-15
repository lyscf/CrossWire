package transport

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"crosswire/internal/utils"

	"github.com/gorilla/websocket"
)

// HTTPSTransport HTTPS/WebSocket传输实现
// 参考: docs/PROTOCOL.md - 3. HTTPS传输协议
type HTTPSTransport struct {
	config *Config
	mode   string // "server" or "client"

	// WebSocket
	conn   *websocket.Conn
	connMu sync.RWMutex
	// 串行化写操作，避免与心跳并发写冲突
	writeMu sync.Mutex

	// HTTP服务器（服务端模式）
	server   *http.Server
	upgrader websocket.Upgrader

	// 客户端连接池（服务端模式）
	clients      map[string]*websocket.Conn
	clientsMu    sync.RWMutex
	clientWriteM map[string]*sync.Mutex // 每个连接的写锁，避免并发写冲突

	// 消息处理
	handler     MessageHandler
	fileHandler FileHandler

	// 统计
	stats   TransportStats
	statsMu sync.RWMutex

	// 控制
	ctx       context.Context
	cancel    context.CancelFunc
	started   bool
	connected bool

	// 重连/心跳
	lastURL           string        // 最近一次成功连接的 ws(s) URL
	pingInterval      time.Duration // 心跳发送间隔
	pongWait          time.Duration // 允许的最大未收到PONG的时长
	reconnect         bool          // 是否启用自动重连
	reconnectDelay    time.Duration // 重连初始等待
	reconnectMaxDelay time.Duration // 重连最大等待

	// 日志
	logger *utils.Logger

	// 服务端信息（仅server模式用于 /info）
	serverChannelID   string
	serverChannelName string
}

// 轻量日志封装，避免nil检查分散在代码中
func (t *HTTPSTransport) logDebug(format string, args ...interface{}) {
	if t.logger != nil {
		t.logger.Debug("[HTTPS] "+format, args...)
	}
}

func (t *HTTPSTransport) logInfo(format string, args ...interface{}) {
	if t.logger != nil {
		t.logger.Info("[HTTPS] "+format, args...)
	} else {
		fmt.Printf("[INFO] "+format+"\n", args...)
	}
}

func (t *HTTPSTransport) logWarn(format string, args ...interface{}) {
	if t.logger != nil {
		t.logger.Warn("[HTTPS] "+format, args...)
	} else {
		fmt.Printf("[WARN] "+format+"\n", args...)
	}
}

func (t *HTTPSTransport) logError(format string, args ...interface{}) {
	if t.logger != nil {
		t.logger.Error("[HTTPS] "+format, args...)
	} else {
		fmt.Printf("[ERROR] "+format+"\n", args...)
	}
}

// NewHTTPSTransport 创建HTTPS传输层
func NewHTTPSTransport() *HTTPSTransport {
	return &HTTPSTransport{
		clients:      make(map[string]*websocket.Conn),
		clientWriteM: make(map[string]*sync.Mutex),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: 添加安全的Origin检查
			},
		},
		// 默认心跳与重连参数
		pingInterval:      20 * time.Second,
		pongWait:          60 * time.Second,
		reconnect:         true,
		reconnectDelay:    1 * time.Second,
		reconnectMaxDelay: 15 * time.Second,
	}
}

// ===== 生命周期管理 =====

// Init 初始化
func (t *HTTPSTransport) Init(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	t.config = config
	t.logger = config.Logger
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.stats.StartTime = time.Now()

	t.logInfo("Initialized (mode=%s, port=%d)", t.mode, t.config.Port)
	return nil
}

// Start 启动传输层
func (t *HTTPSTransport) Start() error {
	if t.started {
		return fmt.Errorf("transport already started")
	}

	if t.config == nil {
		return fmt.Errorf("transport not initialized")
	}

	t.started = true

	// 服务端模式：启动HTTP服务器
	if t.mode == "server" {
		return t.startServer()
	}

	return nil
}

// startServer 启动HTTP服务器（服务端模式）
func (t *HTTPSTransport) startServer() error {
	addr := fmt.Sprintf(":%d", t.config.Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", t.handleWebSocket)
	mux.HandleFunc("/info", t.handleInfo)

	// 自签名证书场景：启动 TLS 服务器，但允许客户端跳过校验（客户端侧已禁用验证）
	t.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  t.config.ReadTimeout,
		WriteTimeout: t.config.WriteTimeout,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	// 启动服务器
	go func() {
		var err error
		if t.config.TLSCert != "" && t.config.TLSKey != "" {
			// TLS模式（外部提供证书）
			err = t.server.ListenAndServeTLS(t.config.TLSCert, t.config.TLSKey)
		} else {
			// 自动生成自签名证书（优先走TLS）
			certPath, keyPath, genErr := utils.EnsureSelfSignedCert("./certs", nil, 365)
			if genErr == nil {
				t.logInfo("Using self-signed TLS cert: %s", certPath)
				err = t.server.ListenAndServeTLS(certPath, keyPath)
			} else {
				// 回退到非TLS（仅当生成失败）
				t.logWarn("Self-signed cert generation failed: %v, falling back to HTTP", genErr)
				err = t.server.ListenAndServe()
			}
		}

		if err != nil && err != http.ErrServerClosed {
			t.logError("HTTP server error: %v", err)
		}
	}()

	t.logInfo("HTTPS transport server started on %s", addr)
	return nil
}

// Stop 停止传输层
func (t *HTTPSTransport) Stop() error {
	if !t.started {
		return nil
	}

	t.started = false
	t.cancel()

	// 关闭服务器
	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		t.server.Shutdown(ctx)
	}

	// 关闭所有客户端连接
	t.clientsMu.Lock()
	for _, conn := range t.clients {
		conn.Close()
	}
	t.clients = make(map[string]*websocket.Conn)
	t.clientsMu.Unlock()

	// 关闭当前连接
	t.connMu.Lock()
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
		t.connected = false
	}
	t.connMu.Unlock()

	return nil
}

// ===== 连接管理 =====

// Connect 连接到服务器（客户端模式）
func (t *HTTPSTransport) Connect(target string) error {
	if t.connected {
		return fmt.Errorf("already connected")
	}

	t.mode = "client"

	// 构造WebSocket URL：
	// - 若调用方已提供完整的 ws:// 或 wss:// URL，直接使用
	// - 否则默认使用 wss:// 并追加 /ws（HTTPS模式默认启用TLS）
	wsURL := target
	if !(len(wsURL) >= 5 && (wsURL[:5] == "ws://" || (len(wsURL) >= 6 && wsURL[:6] == "wss://"))) {
		wsURL = fmt.Sprintf("wss://%s/ws", target)
	}

	// 配置TLS
	dialer := websocket.DefaultDialer
	// HTTPS模式下：仅当 SkipTLSVerify 为 true 时跳过校验
	if strings.HasPrefix(wsURL, "wss://") {
		if t.config != nil && t.config.SkipTLSVerify {
			dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}

	// 连接
	t.logInfo("Connecting to %s (wsURL=%s)", target, wsURL)
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		lowered := strings.ToLower(err.Error())
		if strings.Contains(lowered, "tls:") || strings.Contains(lowered, "handshake") || strings.Contains(lowered, "first record does not look like a tls handshake") {
			// 尝试回退到 ws:// 以兼容未启用TLS的开发环境
			fallbackURL := wsURL
			if strings.HasPrefix(wsURL, "wss://") {
				fallbackURL = "ws://" + strings.TrimPrefix(wsURL, "wss://")
			}
			t.logWarn("TLS handshake failed for %s, trying ws fallback: %s", wsURL, fallbackURL)
			conn2, _, err2 := websocket.DefaultDialer.Dial(fallbackURL, nil)
			if err2 != nil {
				t.logError("Fallback ws connect failed: %v (original: %v)", err2, err)
				return fmt.Errorf("failed to connect: %w", err)
			}
			conn = conn2
			wsURL = fallbackURL
		} else {
			t.logError("Failed to connect to %s: %v", wsURL, err)
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	t.connMu.Lock()
	t.conn = conn
	t.connected = true
	t.connMu.Unlock()

	t.lastURL = wsURL

	// WebSocket心跳：设置Pong处理器以刷新读取期限
	_ = t.setupPingPong(conn)

	// 启动接收协程
	go t.receiveLoop()

	if strings.HasPrefix(wsURL, "ws://") {
		t.logInfo("Connected (insecure) to %s", target)
	} else {
		t.logInfo("Connected to %s", target)
	}
	return nil
}

// Disconnect 断开连接
func (t *HTTPSTransport) Disconnect() error {
	t.connMu.Lock()
	defer t.connMu.Unlock()

	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
		t.connected = false
	}

	return nil
}

// IsConnected 检查连接状态
func (t *HTTPSTransport) IsConnected() bool {
	t.connMu.RLock()
	defer t.connMu.RUnlock()
	return t.connected
}

// ===== 消息收发 =====

// SendMessage 发送消息
func (t *HTTPSTransport) SendMessage(msg *Message) error {
	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if t.mode == "server" {
		// 服务端模式：广播到所有客户端
		return t.broadcast(data)
	} else {
		// 客户端模式：发送到服务器
		return t.send(data)
	}
}

// send 发送数据到当前连接
func (t *HTTPSTransport) send(data []byte) error {
	t.connMu.RLock()
	conn := t.conn
	t.connMu.RUnlock()

	if conn == nil {
		t.logWarn("Send called but not connected")
		return fmt.Errorf("not connected")
	}

	// 设置写超时
	if t.config.WriteTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
	}

	// 发送WebSocket消息（串行写）
	t.writeMu.Lock()
	err := conn.WriteMessage(websocket.BinaryMessage, data)
	t.writeMu.Unlock()
	if err != nil {
		t.logError("Failed to send message: %v", err)
		return fmt.Errorf("failed to send: %w", err)
	}

	// 更新统计
	t.statsMu.Lock()
	t.stats.BytesSent += uint64(len(data))
	t.stats.MessagesSent++
	t.stats.LastActivity = time.Now()
	t.statsMu.Unlock()

	return nil
}

// broadcast 广播到所有客户端（服务端模式）
func (t *HTTPSTransport) broadcast(data []byte) error {
	t.clientsMu.RLock()
	defer t.clientsMu.RUnlock()

	var errors []error
	for clientID, conn := range t.clients {
		// 逐连接写锁，串行化发送
		if m, ok := t.clientWriteM[clientID]; ok {
			m.Lock()
		}
		// 写超时
		if t.config.WriteTimeout > 0 {
			conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			errors = append(errors, fmt.Errorf("failed to send to %s: %w", clientID, err))
		}
		if m, ok := t.clientWriteM[clientID]; ok {
			m.Unlock()
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("broadcast errors: %v", errors)
	}

	// 更新统计
	t.statsMu.Lock()
	t.stats.BytesSent += uint64(len(data)) * uint64(len(t.clients))
	t.stats.MessagesSent += uint64(len(t.clients))
	t.stats.LastActivity = time.Now()
	t.statsMu.Unlock()

	return nil
}

// ReceiveMessage 接收消息（阻塞）
func (t *HTTPSTransport) ReceiveMessage() (*Message, error) {
	t.connMu.RLock()
	conn := t.conn
	t.connMu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// 设置读超时
	if t.config.ReadTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(t.config.ReadTimeout))
	}

	// 读取WebSocket消息
	_, data, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to receive: %w", err)
	}

	// 反序列化消息
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// 更新统计
	t.statsMu.Lock()
	t.stats.BytesReceived += uint64(len(data))
	t.stats.MessagesRecv++
	t.stats.LastActivity = time.Now()
	t.statsMu.Unlock()

	return &msg, nil
}

// setupPingPong 启用心跳（客户端）
func (t *HTTPSTransport) setupPingPong(conn *websocket.Conn) error {
	if conn == nil {
		return nil
	}

	// 收到PONG时刷新读期限
	conn.SetPongHandler(func(string) error {
		if t.pongWait > 0 {
			conn.SetReadDeadline(time.Now().Add(t.pongWait))
		}
		return nil
	})

	// 周期发送PING
	if t.pingInterval > 0 {
		go func() {
			ticker := time.NewTicker(t.pingInterval)
			defer ticker.Stop()
			for {
				select {
				case <-t.ctx.Done():
					return
				case <-ticker.C:
					t.connMu.RLock()
					c := t.conn
					t.connMu.RUnlock()
					if c == nil {
						return
					}
					t.writeMu.Lock()
					_ = c.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
					t.writeMu.Unlock()
				}
			}
		}()
	}

	return nil
}

// reconnectLoop 自动重连（指数退避，不超过上限）
func (t *HTTPSTransport) reconnectLoop() {
	delay := t.reconnectDelay
	for t.reconnect {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		t.connMu.RLock()
		connected := t.connected
		t.connMu.RUnlock()
		if connected {
			return
		}

		url := t.lastURL
		if url == "" {
			return
		}

		t.logInfo("Reconnecting to %s ...", url)
		// 使用默认拨号器重连（保留TLS设置）
		dialer := websocket.DefaultDialer
		if strings.HasPrefix(url, "wss://") {
			if t.config != nil && t.config.SkipTLSVerify {
				dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			}
		}
		conn, _, err := dialer.Dial(url, nil)
		if err != nil {
			t.logWarn("Reconnect failed: %v", err)
			time.Sleep(delay)
			// 指数退避
			delay *= 2
			if delay > t.reconnectMaxDelay {
				delay = t.reconnectMaxDelay
			}
			continue
		}

		t.connMu.Lock()
		t.conn = conn
		t.connected = true
		t.connMu.Unlock()
		_ = t.setupPingPong(conn)

		// 重启接收循环
		go t.receiveLoop()
		t.logInfo("Reconnected")
		return
	}
}

// Subscribe 订阅消息（异步回调）
func (t *HTTPSTransport) Subscribe(handler MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}
	t.handler = handler
	return nil
}

// Unsubscribe 取消订阅
func (t *HTTPSTransport) Unsubscribe() {
	t.handler = nil
}

// receiveLoop 接收循环（客户端模式）
func (t *HTTPSTransport) receiveLoop() {
	// 设置初始读期限
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		// 读期限：每次读取前刷新
		t.connMu.RLock()
		conn := t.conn
		t.connMu.RUnlock()
		if conn == nil {
			return
		}
		if t.pongWait > 0 {
			conn.SetReadDeadline(time.Now().Add(t.pongWait))
		}

		msg, err := t.ReceiveMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			t.logWarn("Receive loop error: %v", err)
			// 触发重连
			if t.reconnect {
				go t.reconnectLoop()
			}
			return
		}

		// 调用处理函数
		if t.handler != nil {
			go t.handler(msg)
		}
	}
}

// handleWebSocket 处理WebSocket连接（服务端模式）
func (t *HTTPSTransport) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级到WebSocket
	conn, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.logError("WebSocket upgrade error: %v", err)
		return
	}

	// 生成客户端ID
	clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())

	// 添加到客户端列表
	t.clientsMu.Lock()
	t.clients[clientID] = conn
	t.clientWriteM[clientID] = &sync.Mutex{}
	t.clientsMu.Unlock()

	t.logInfo("Client connected: %s", clientID)

	// 处理客户端消息
	defer func() {
		t.clientsMu.Lock()
		delete(t.clients, clientID)
		delete(t.clientWriteM, clientID)
		t.clientsMu.Unlock()
		conn.Close()
		t.logInfo("Client disconnected: %s", clientID)
	}()

	// Server端也设置心跳：收Pong刷新读期限
	conn.SetPongHandler(func(string) error {
		if t.pongWait > 0 {
			conn.SetReadDeadline(time.Now().Add(t.pongWait))
		}
		return nil
	})

	for {
		if t.pongWait > 0 {
			conn.SetReadDeadline(time.Now().Add(t.pongWait))
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			break
		}

		// 反序列化消息
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.logWarn("Invalid message from %s: %v", clientID, err)
			continue
		}

		// 设置来源信息
		msg.SenderAddr = r.RemoteAddr

		// 更新统计
		t.statsMu.Lock()
		t.stats.BytesReceived += uint64(len(data))
		t.stats.MessagesRecv++
		t.stats.LastActivity = time.Now()
		t.statsMu.Unlock()

		// 调用处理函数
		if t.handler != nil {
			go t.handler(&msg)
		}
	}
}

// ===== 文件传输 =====

// SendFile 发送文件
func (t *HTTPSTransport) SendFile(file *FileTransfer) error {
	// TODO: 实现文件分块传输
	return fmt.Errorf("not implemented")
}

// OnFileReceived 文件接收回调
func (t *HTTPSTransport) OnFileReceived(handler FileHandler) error {
	t.fileHandler = handler
	return nil
}

// ===== 服务发现 =====

// Discover 发现可用的服务端
func (t *HTTPSTransport) Discover(timeout time.Duration) ([]*PeerInfo, error) {
	// TODO: 实现服务发现（可以使用mDNS或广播）
	return nil, fmt.Errorf("not implemented")
}

// Announce 宣告服务
func (t *HTTPSTransport) Announce(info *ServiceInfo) error {
	// HTTPS 模式不参与局域网发现，宣告为无操作以避免上层警告
	return nil
}

// ===== 元数据 =====

// GetMode 获取传输模式
func (t *HTTPSTransport) GetMode() TransportMode {
	return TransportModeHTTPS
}

// GetStats 获取传输统计
func (t *HTTPSTransport) GetStats() *TransportStats {
	t.statsMu.RLock()
	defer t.statsMu.RUnlock()

	// 返回副本
	stats := t.stats
	return &stats
}

// GetClientCount 获取客户端数量（服务端模式）
func (t *HTTPSTransport) GetClientCount() int {
	t.clientsMu.RLock()
	defer t.clientsMu.RUnlock()
	return len(t.clients)
}

// GetLocalAddr 获取本地监听地址
func (t *HTTPSTransport) GetLocalAddr() string {
	if t.server != nil {
		return t.server.Addr
	}
	return ""
}

// GetRemoteAddr 获取远程连接地址（客户端模式）
func (t *HTTPSTransport) GetRemoteAddr() string {
	t.connMu.RLock()
	defer t.connMu.RUnlock()

	if t.conn != nil {
		return t.conn.RemoteAddr().String()
	}
	return ""
}

// SetMode 设置模式（"server" or "client"）
func (t *HTTPSTransport) SetMode(mode string) {
	t.mode = mode
}

// SetChannelInfo 设置频道信息（供 /info 使用）
func (t *HTTPSTransport) SetChannelInfo(channelID, channelName string) {
	t.serverChannelID = channelID
	t.serverChannelName = channelName
}

// handleInfo 返回服务端频道基础信息，供客户端在加入前获取 ChannelID
func (t *HTTPSTransport) handleInfo(w http.ResponseWriter, _ *http.Request) {
	type infoResp struct {
		ChannelID   string `json:"channel_id"`
		ChannelName string `json:"channel_name"`
		Mode        string `json:"mode"`
		Version     int    `json:"version"`
	}
	resp := infoResp{
		ChannelID:   t.serverChannelID,
		ChannelName: t.serverChannelName,
		Mode:        string(TransportModeHTTPS),
		Version:     1,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
