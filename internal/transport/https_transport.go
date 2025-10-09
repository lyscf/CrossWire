package transport

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

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

	// HTTP服务器（服务端模式）
	server   *http.Server
	upgrader websocket.Upgrader

	// 客户端连接池（服务端模式）
	clients   map[string]*websocket.Conn
	clientsMu sync.RWMutex

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

	// 日志
	// TODO: 集成logger
}

// NewHTTPSTransport 创建HTTPS传输层
func NewHTTPSTransport() *HTTPSTransport {
	return &HTTPSTransport{
		clients: make(map[string]*websocket.Conn),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: 添加安全的Origin检查
			},
		},
	}
}

// ===== 生命周期管理 =====

// Init 初始化
func (t *HTTPSTransport) Init(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	t.config = config
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.stats.StartTime = time.Now()

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
			// TLS模式
			err = t.server.ListenAndServeTLS(t.config.TLSCert, t.config.TLSKey)
		} else {
			// 非TLS模式（开发用）
			err = t.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			// TODO: 记录日志
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	fmt.Printf("HTTPS transport server started on %s\n", addr)
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
	// HTTPS模式下使用自签名证书：禁用证书校验（按需可在将来改为固定指纹校验）
	if len(wsURL) >= 6 && wsURL[:6] == "wss://" {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// 连接
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	t.connMu.Lock()
	t.conn = conn
	t.connected = true
	t.connMu.Unlock()

	// 启动接收协程
	go t.receiveLoop()

	fmt.Printf("Connected to %s\n", target)
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
		return fmt.Errorf("not connected")
	}

	// 设置写超时
	if t.config.WriteTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
	}

	// 发送WebSocket消息
	err := conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
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
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			errors = append(errors, fmt.Errorf("failed to send to %s: %w", clientID, err))
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
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		msg, err := t.ReceiveMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			// TODO: 记录错误日志
			continue
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
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}

	// 生成客户端ID
	clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())

	// 添加到客户端列表
	t.clientsMu.Lock()
	t.clients[clientID] = conn
	t.clientsMu.Unlock()

	fmt.Printf("Client connected: %s\n", clientID)

	// 处理客户端消息
	defer func() {
		t.clientsMu.Lock()
		delete(t.clients, clientID)
		t.clientsMu.Unlock()
		conn.Close()
		fmt.Printf("Client disconnected: %s\n", clientID)
	}()

	for {
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

// TODO: 实现以下功能
// - 心跳检测
// - 自动重连
// - 消息压缩
// - 流量控制
// - TLS证书验证
// - 客户端认证
