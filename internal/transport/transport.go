package transport

import (
	"time"

	"crosswire/internal/models"
)

// TransportMode 使用models包的定义
type TransportMode = models.TransportMode

// TransportMode常量（复用models包的定义）
const (
	TransportModeARP   = models.TransportARP
	TransportModeHTTPS = models.TransportHTTPS
	TransportModeMDNS  = models.TransportMDNS
)

// Transport 传输层统一接口
// 参考: docs/ARCHITECTURE.md - 3.1.4 传输模块
type Transport interface {
	// ===== 生命周期管理 =====

	// Init 初始化传输层
	Init(config *Config) error

	// Start 启动传输层
	Start() error

	// Stop 停止传输层
	Stop() error

	// ===== 连接管理 =====

	// Connect 连接到目标（客户端使用）
	Connect(target string) error

	// Disconnect 断开连接
	Disconnect() error

	// IsConnected 检查连接状态
	IsConnected() bool

	// ===== 消息收发 =====

	// SendMessage 发送消息
	SendMessage(msg *Message) error

	// ReceiveMessage 接收消息（阻塞）
	ReceiveMessage() (*Message, error)

	// Subscribe 订阅消息（异步回调）
	Subscribe(handler MessageHandler) error

	// Unsubscribe 取消订阅
	Unsubscribe()

	// ===== 文件传输 =====

	// SendFile 发送文件
	SendFile(file *FileTransfer) error

	// OnFileReceived 文件接收回调
	OnFileReceived(handler FileHandler) error

	// ===== 服务发现 =====

	// Discover 发现可用的服务端
	Discover(timeout time.Duration) ([]*PeerInfo, error)

	// Announce 宣告服务（服务端使用）
	Announce(info *ServiceInfo) error

	// ===== 元数据 =====

	// GetMode 获取传输模式
	GetMode() TransportMode

	// GetStats 获取传输统计
	GetStats() *TransportStats
}

// Config 传输层配置
type Config struct {
	Mode      TransportMode // 传输模式
	Interface string        // 网卡接口名称（ARP模式）
	Port      int           // 监听端口（HTTPS模式）
	TLSCert   string        // TLS证书路径（HTTPS模式）
	TLSKey    string        // TLS私钥路径（HTTPS模式）

	// 超时配置
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration

	// 缓冲区配置
	RecvBufferSize int
	SendBufferSize int

	// 重传配置
	MaxRetries int
	RetryDelay time.Duration
}

// Message 传输层消息
type Message struct {
	// 元数据
	ID        string    // 消息ID
	Timestamp time.Time // 时间戳
	Sequence  uint32    // 序列号

	// 来源
	SenderID   string // 发送者ID
	SenderMAC  string // 发送者MAC地址（ARP模式）
	SenderAddr string // 发送者IP:Port（HTTPS模式）

	// 内容
	Type    MessageType // 消息类型
	Payload []byte      // 加密后的负载

	// 分块传输
	TotalChunks uint16 // 总分块数
	ChunkIndex  uint16 // 当前分块索引

	// 校验
	Checksum uint32 // CRC32校验和

	// 加密
	Encrypted  bool // 是否加密
	KeyVersion int  // 密钥版本

	// 签名（ARP广播模式）
	Signature []byte // Ed25519签名（服务器签名）
}

// MessageType 消息类型
// 参考: docs/PROTOCOL.md - 2.1.3 帧类型定义
type MessageType byte

const (
	MessageTypeData     MessageType = 0x01 // 数据帧
	MessageTypeACK      MessageType = 0x02 // 确认帧
	MessageTypeNACK     MessageType = 0x03 // 否定确认（请求重传）
	MessageTypeControl  MessageType = 0x04 // 控制帧
	MessageTypeDiscover MessageType = 0x05 // 服务发现
	MessageTypeAuth     MessageType = 0x06 // 认证握手
)

// MessageHandler 消息处理回调函数
type MessageHandler func(msg *Message)

// FileTransfer 文件传输
type FileTransfer struct {
	FileID        string
	Filename      string
	Size          int64
	ChunkSize     int
	TotalChunks   int
	Data          []byte // 完整数据或当前分块数据
	ChunkIndex    int    // 当前分块索引
	Checksum      string // 完整文件校验和
	ChunkChecksum string // 当前分块校验和
}

// FileHandler 文件接收回调函数
type FileHandler func(file *FileTransfer)

// PeerInfo 对等节点信息
type PeerInfo struct {
	ID            string        // 节点ID
	Address       string        // 地址（IP:Port或MAC）
	Mode          TransportMode // 传输模式
	LastSeen      time.Time     // 最后发现时间
	ChannelIDHash string        // 频道ID哈希（前8字符）
	Version       int           // 协议版本
}

// ServiceInfo 服务信息（用于宣告）
type ServiceInfo struct {
	ChannelID      string        // 频道ID
	ChannelName    string        // 频道名称（可选）
	Mode           TransportMode // 传输模式
	Port           int           // 端口（HTTPS模式）
	Interface      string        // 网卡接口（ARP模式）
	Version        int           // 协议版本
	MaxMembers     int           // 最大成员数
	CurrentMembers int           // 当前成员数
}

// TransportStats 传输统计
type TransportStats struct {
	BytesSent     uint64    // 发送字节数
	BytesReceived uint64    // 接收字节数
	MessagesSent  uint64    // 发送消息数
	MessagesRecv  uint64    // 接收消息数
	Errors        uint64    // 错误计数
	Retries       uint64    // 重传次数
	StartTime     time.Time // 启动时间
	LastActivity  time.Time // 最后活动时间
}

// ProtocolVersion 协议版本
const ProtocolVersion = 1

// 以太网帧配置（ARP模式）
const (
	EtherTypeCustom = 0x88B5              // CrossWire自定义EtherType
	MaxFramePayload = 1470                // 最大负载大小（字节）
	FrameHeaderSize = 34                  // 帧头大小
	BroadcastMAC    = "FF:FF:FF:FF:FF:FF" // 广播MAC地址
)

// TODO: 实现以下功能
// - 消息分块和重组
// - ACK确认机制
// - 重传队列
// - 流量控制
// - QoS优先级
