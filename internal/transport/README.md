# Transport 传输层

> CrossWire传输层实现

---

## 📚 概述

Transport传输层提供了统一的网络通信接口，支持多种传输模式的无缝切换。

**参考文档**:
- `docs/ARCHITECTURE.md` - 3.1.4 传输模块
- `docs/PROTOCOL.md` - 传输协议规范

---

## 🏗️ 架构设计

### 接口定义

```go
type Transport interface {
    // 生命周期
    Init(config *Config) error
    Start() error
    Stop() error
    
    // 连接管理
    Connect(target string) error
    Disconnect() error
    IsConnected() bool
    
    // 消息收发
    SendMessage(msg *Message) error
    ReceiveMessage() (*Message, error)
    Subscribe(handler MessageHandler) error
    
    // 文件传输
    SendFile(file *FileTransfer) error
    OnFileReceived(handler FileHandler) error
    
    // 服务发现
    Discover(timeout time.Duration) ([]*PeerInfo, error)
    Announce(info *ServiceInfo) error
    
    // 元数据
    GetMode() TransportMode
    GetStats() *TransportStats
}
```

---

## 📦 实现状态

### ✅ HTTPS Transport (已实现)

**文件**: `https_transport.go`

**功能**:
- ✅ WebSocket通信
- ✅ 服务端模式（监听连接）
- ✅ 客户端模式（主动连接）
- ✅ 消息广播（服务端）
- ✅ 异步消息处理
- ✅ 连接管理
- ✅ 统计信息

**使用示例**:

```go
// 服务端
transport := NewHTTPSTransport()
transport.SetMode("server")
transport.Init(&Config{
    Mode: TransportModeHTTPS,
    Port: 8443,
})
transport.Subscribe(func(msg *Message) {
    // 处理消息
})
transport.Start()

// 客户端
transport := NewHTTPSTransport()
transport.SetMode("client")
transport.Init(&Config{
    Mode: TransportModeHTTPS,
})
transport.Subscribe(func(msg *Message) {
    // 处理消息
})
transport.Connect("localhost:8443")
```

**待完成**:
- ⏳ 心跳检测
- ⏳ 自动重连
- ⏳ TLS证书验证
- ⏳ 消息压缩

---

### ✅ ARP Transport (已实现)

**文件**: `arp_transport.go`

**功能**:
- ✅ 原始以太网帧构造（gopacket）
- ✅ 服务器签名广播模式
- ✅ 客户端过滤接收
- ✅ CRC32校验
- ✅ 防重放攻击
- ✅ 签名验证（待集成crypto）
- ✅ 统计信息

**使用示例**:

```go
// 服务端
transport := NewARPTransport()
transport.SetMode("server")
transport.Init(&Config{
    Mode: TransportModeARP,
    Interface: "eth0",
})
transport.SetServerKeys(privKey, pubKey)
transport.Subscribe(func(msg *Message) {
    // 处理消息
})
transport.Start()

// 客户端
transport := NewARPTransport()
transport.SetMode("client")
transport.Init(&Config{
    Mode: TransportModeARP,
    Interface: "eth0",
})
transport.SetServerPublicKey(pubKey)
transport.Subscribe(func(msg *Message) {
    // 处理消息
})
transport.Connect("aa:bb:cc:dd:ee:ff")
```

**待完成**:
- ⏳ 消息分块和重组
- ⏳ ACK确认机制
- ⏳ 重传队列
- ⏳ 与crypto.Manager集成

**参考文档**: `docs/PROTOCOL.md` - 2. ARP传输协议, `docs/ARP_BROADCAST_MODE.md`

---

### ✅ mDNS Transport (已实现)

**文件**: `mdns_transport.go`

**功能**:
- ✅ mDNS服务发现
- ✅ 服务注册和宣告
- ✅ 服务名编码传输（Base64URL）
- ✅ 消息分块和重组
- ✅ UDP单播通信
- ✅ 服务器签名模式（待集成crypto）
- ✅ 防重放攻击
- ✅ 统计信息

**使用示例**:

```go
// 服务端
transport := NewMDNSTransport()
transport.SetMode("server")
transport.SetChannelInfo(channelID, "CTF-Team")
transport.SetServerKeys(privKey, pubKey)
transport.Init(&Config{
    Mode: TransportModeMDNS,
})
transport.Subscribe(func(msg *Message) {
    // 处理消息
})
transport.Start()

// 客户端
transport := NewMDNSTransport()
transport.SetMode("client")
transport.Init(&Config{
    Mode: TransportModeMDNS,
})
transport.SetServerPublicKey(pubKey)
transport.Subscribe(func(msg *Message) {
    // 处理消息
})

// 发现服务
peers, _ := transport.Discover(5 * time.Second)
transport.Connect(peers[0].Address)
```

**待完成**:
- ⏳ 与crypto.Manager集成
- ⏳ 文件传输
- ⏳ 更高效的消息编码

**参考文档**: `docs/PROTOCOL.md` - 4. mDNS传输协议

---

## 🔧 配置说明

### Config结构

```go
type Config struct {
    Mode           TransportMode // 传输模式
    Interface      string        // 网卡接口（ARP）
    Port           int          // 端口（HTTPS）
    TLSCert        string       // TLS证书
    TLSKey         string       // TLS密钥
    ConnectTimeout time.Duration
    ReadTimeout    time.Duration
    WriteTimeout   time.Duration
    RecvBufferSize int
    SendBufferSize int
    MaxRetries     int
    RetryDelay     time.Duration
}
```

### 默认配置

```go
DefaultConfig = &Config{
    Port:           8443,
    ConnectTimeout: 10 * time.Second,
    ReadTimeout:    30 * time.Second,
    WriteTimeout:   10 * time.Second,
    RecvBufferSize: 4096,
    SendBufferSize: 4096,
    MaxRetries:     3,
    RetryDelay:     1 * time.Second,
}
```

---

## 🎯 消息格式

### Message结构

```go
type Message struct {
    ID          string      // 消息ID
    Timestamp   time.Time   // 时间戳
    Sequence    uint32      // 序列号
    SenderID    string      // 发送者ID
    Type        MessageType // 消息类型
    Payload     []byte      // 加密负载
    TotalChunks uint16      // 总分块数
    ChunkIndex  uint16      // 分块索引
    Checksum    uint32      // CRC32校验
    Encrypted   bool        // 是否加密
    KeyVersion  int         // 密钥版本
    Signature   []byte      // Ed25519签名
}
```

### MessageType枚举

```go
const (
    MessageTypeData     = 0x01 // 数据帧
    MessageTypeACK      = 0x02 // 确认帧
    MessageTypeNACK     = 0x03 // 否定确认
    MessageTypeControl  = 0x04 // 控制帧
    MessageTypeDiscover = 0x05 // 服务发现
    MessageTypeAuth     = 0x06 // 认证握手
)
```

---

## 🏭 工厂模式

使用Factory创建Transport实例:

```go
factory := NewFactory()

// 创建HTTPS传输层
transport, err := factory.Create(TransportModeHTTPS)

// 创建并初始化
transport, err := factory.CreateWithConfig(
    TransportModeHTTPS,
    &Config{Port: 8443},
)

// 检查支持的模式
modes := factory.GetSupportedModes()
isSupported := factory.IsModeSupported(TransportModeHTTPS)
```

---

## 📊 统计信息

### TransportStats结构

```go
type TransportStats struct {
    BytesSent     uint64
    BytesReceived uint64
    MessagesSent  uint64
    MessagesRecv  uint64
    Errors        uint64
    Retries       uint64
    StartTime     time.Time
    LastActivity  time.Time
}
```

### 获取统计

```go
stats := transport.GetStats()
fmt.Printf("Sent: %d bytes, %d messages\n", 
    stats.BytesSent, stats.MessagesSent)
fmt.Printf("Received: %d bytes, %d messages\n",
    stats.BytesReceived, stats.MessagesRecv)
```

---

## 🔐 安全考虑

### 加密

- 所有消息负载在传输层之上加密（应用层加密）
- HTTPS模式额外提供TLS传输层加密
- 支持密钥版本管理

### 签名

- ARP广播模式：服务器Ed25519签名
- 客户端验证签名防止伪造
- 签名包含时间戳防重放

### 认证

- 使用频道密码派生的密钥进行握手
- 解密成功即表示认证通过
- 无需额外的Challenge-Response

---

## 🧪 测试

### 单元测试

```bash
go test ./internal/transport/...
```

### 集成测试

```bash
go test -tags=integration ./internal/transport/...
```

---

## 📝 TODO列表

### HTTPS Transport
- [ ] 实现心跳检测
- [ ] 实现自动重连
- [ ] TLS证书验证
- [ ] 客户端认证
- [ ] 消息压缩
- [ ] 流量控制
- [ ] QoS优先级

### ARP Transport
- [x] 实现完整的ARP传输层
- [x] 原始套接字操作（gopacket）
- [x] 以太网帧构造
- [x] 服务器签名广播模式
- [x] 客户端签名验证
- [ ] 消息分块和重组
- [ ] ACK机制
- [ ] 重传队列
- [ ] 与crypto.Manager集成

### mDNS Transport
- [x] 实现mDNS传输层
- [x] 服务发现
- [x] 服务注册和宣告
- [x] 服务名编码（Base64URL）
- [x] UDP通信
- [x] 消息重组器
- [x] 服务器签名模式
- [ ] 与crypto.Manager集成
- [ ] 文件传输

### 通用功能
- [x] 统一接口定义
- [x] 工厂模式
- [x] 统计信息
- [x] 防重放攻击
- [ ] 消息分块和重组（通用）
- [ ] ACK确认机制
- [ ] 重传队列
- [ ] 流量控制
- [ ] 日志集成
- [ ] 性能优化

---

## 📖 相关文档

- [ARCHITECTURE.md](../../docs/ARCHITECTURE.md) - 系统架构
- [PROTOCOL.md](../../docs/PROTOCOL.md) - 通信协议
- [ARP_BROADCAST_MODE.md](../../docs/ARP_BROADCAST_MODE.md) - ARP广播模式

---

**最后更新**: 2025-10-05  
**状态**: ✅ 三种传输层全部实现完成（HTTPS/ARP/mDNS）


