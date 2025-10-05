# CrossWire 通信协议文档

> CTF 线下赛通讯系统 - 通信协议规范
> 
> Version: 1.0.0  
> Date: 2025-10-05

---

## 📑 目录

- [1. 协议概述](#1-协议概述)
- [2. ARP 传输协议](#2-arp-传输协议)
- [3. HTTPS 传输协议](#3-https-传输协议)
- [4. mDNS 传输协议](#4-mdns-传输协议)
- [5. 应用层协议](#5-应用层协议)
- [6. 加密与安全](#6-加密与安全)
- [7. 协议扩展](#7-协议扩展)

---

## 1. 协议概述

### 1.1 协议栈架构

```
┌────────────────────────────────────┐
│     应用层 (Application Layer)     │
│  - 消息格式                        │
│  - 文件传输协议                    │
│  - 控制命令                        │
├────────────────────────────────────┤
│     加密层 (Encryption Layer)      │
│  - X25519 对称加密            │
│  - RSA 密钥交换                    │
│  - 消息签名验证                    │
├────────────────────────────────────┤
│     传输层 (Transport Layer)       │
│  - ARP (原始以太网)                │
│  - HTTPS/WebSocket                 │
│  - mDNS (服务发现)                 │
└────────────────────────────────────┘
```

### 1.2 协议特点

| 特性 | ARP 协议 | HTTPS 协议 | mDNS 协议 |
|------|----------|------------|-----------|
| **层级** | OSI 第2层 | OSI 第7层 | OSI 第7层 |
| **基础协议** | Ethernet | TCP/TLS | UDP/DNS |
| **数据单位** | Frame (帧) | Message (消息) | Service (服务) |
| **最大负载** | 1470 字节 | 无限制 | 50 字节 |
| **可靠性** | 手动 ACK | TCP 保证 | 手动重传 |
| **加密** | 应用层 | TLS + 应用层 | 应用层 |

---

## 2. ARP 传输协议

### 2.1 以太网帧结构

#### 2.1.1 完整帧格式

```
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
├─────────────────────────────────────────────────────────────┤
│                    目标 MAC 地址 (6 字节)                    │
├─────────────────────────────────────────────────────────────┤
│                    源 MAC 地址 (6 字节)                      │
├─────────────────────────────────────────────────────────────┤
│        EtherType (0x88B5)     │   Version   │  Frame Type   │
├─────────────────────────────────────────────────────────────┤
│                        Sequence Number                        │
├─────────────────────────────────────────────────────────────┤
│     Total Chunks    │   Chunk Index   │    Payload Length   │
├─────────────────────────────────────────────────────────────┤
│                          Checksum (CRC32)                     │
├─────────────────────────────────────────────────────────────┤
│                          Reserved (4 字节)                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│                  Payload (最大 1470 字节)                     │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

#### 2.1.2 字段说明

| 字段 | 偏移 | 长度 | 说明 |
|------|------|------|------|
| **目标 MAC** | 0 | 6 | 目标设备 MAC 地址（广播为 FF:FF:FF:FF:FF:FF）|
| **源 MAC** | 6 | 6 | 发送设备 MAC 地址 |
| **EtherType** | 12 | 2 | 固定为 `0x88B5`（CrossWire 自定义协议）|
| **Version** | 14 | 1 | 协议版本号（当前为 `0x01`）|
| **Frame Type** | 15 | 1 | 帧类型（见下表）|
| **Sequence** | 16 | 4 | 消息序列号（唯一标识一条消息）|
| **Total Chunks** | 20 | 2 | 总分块数 |
| **Chunk Index** | 22 | 2 | 当前块索引（从 0 开始）|
| **Payload Length** | 24 | 2 | 实际负载长度 |
| **Checksum** | 26 | 4 | CRC32 校验和（仅负载部分）|
| **Reserved** | 30 | 4 | 预留字段（用于未来扩展）|
| **Payload** | 34 | 变长 | 加密后的数据 |

#### 2.1.3 帧类型定义

```go
const (
    FrameTypeData     = 0x01  // 数据帧
    FrameTypeACK      = 0x02  // 确认帧
    FrameTypeNACK     = 0x03  // 否定确认（请求重传）
    FrameTypeControl  = 0x04  // 控制帧（心跳、断开等）
    FrameTypeDiscover = 0x05  // 服务发现
    FrameTypeAuth     = 0x06  // 认证握手
)
```

---

### 2.2 通信流程

#### 2.2.1 服务发现（可选）

**方案 1：无需服务发现（推荐）**

客户端直接监听广播，根据密钥自动过滤：
- 尝试用频道密码解密每个广播帧
- 解密成功 = 找到正确的频道
- 无需额外的发现流程

**方案 2：简单发现（可选）**

```
Client: 广播 DISCOVER 帧
  DstMAC: FF:FF:FF:FF:FF:FF
  Payload: {
    "type": "discover",
    "protocol_version": 1
  }

Server: 广播 ANNOUNCE 帧
  DstMAC: FF:FF:FF:FF:FF:FF
  Payload: {
    "type": "announce",
    "channel_id_hash": "sha256(channel_id)[:8]",  // 不泄露完整ID
    "protocol_version": 1
  }
```

---

#### 2.2.2 认证握手

```
1. Client -> Broadcast: JOIN_REQUEST
   DstMAC: FF:FF:FF:FF:FF:FF (广播)
   Payload: AES_Encrypt(password_derived_key, {
     "nickname": "alice",
     "public_key": "X25519_PUBLIC_KEY",
     "timestamp": 1696512000
   })

2. Server -> Broadcast: JOIN_RESPONSE
   DstMAC: FF:FF:FF:FF:FF:FF (广播)
   Payload: AES_Encrypt(password_derived_key, {
     "channel_key": "AES_256_KEY",
     "channel_id": "uuid",
     "member_list": [...],
     "server_public_key": "X25519_PUBLIC_KEY"
   })

3. 完成！客户端获得 channel_key，开始监听所有广播消息
```

**说明：**

- 使用频道密码派生的密钥加密握手消息
- 无需 Challenge-Response（广播环境下简化认证）
- 客户端验证解密成功即表示密码正确
- 所有消息广播，无需维护单播连接

---

#### 2.2.3 消息广播（服务器签名模式）

**发送消息（客户端 → 服务器 → 广播）：**

```
客户端：
1. 构造消息
   msg = {
     "sender_id": "user-uuid",
     "type": "text",
     "content": "...",
     "timestamp": now()
   }

2. 加密消息（使用频道密钥）
   encrypted = X25519(msg, channel_key)

3. 单播发送给服务器（而非广播）
   Frame {
     DstMAC: <server_mac>  // 单播给服务器
     SrcMAC: <client_mac>
     Type: FrameTypeClientMessage
     Payload: encrypted
   }

服务器：
4. 接收客户端消息
   decrypted = AES_Decrypt(payload, channel_key)

5. 验证权限
   - 检查 sender_id 是否是合法成员
   - 检查是否被禁言
   - 检查频率限制

6. 服务器签名
   signature = Ed25519_Sign(decrypted, server_private_key)

7. 广播消息
   Frame {
     DstMAC: FF:FF:FF:FF:FF:FF  // 广播
     SrcMAC: <server_mac>
     Type: FrameTypeServerBroadcast
     Payload: {
       "message": encrypted,
       "signature": signature,
       "timestamp": now()
     }
   }
```

**接收消息（所有客户端）：**

```
1. 监听广播帧
   Filter: 
     - DstMAC == FF:FF:FF:FF:FF:FF
     - Type == FrameTypeServerBroadcast
   
2. 验证服务器签名（关键！）
   if !Ed25519_Verify(frame.Payload.message, frame.Payload.signature, server_public_key):
     return  // 拒绝非服务器签名的消息

3. 解密消息
   decrypted = AES_Decrypt(frame.Payload.message, channel_key)
   if decrypted == nil:
     return  // 解密失败，忽略

4. 验证消息完整性
   - 检查 channel_id
   - 检查 timestamp（防重放）
   - 检查 sender_id 是否在成员列表

5. 处理消息
   msg = Unmarshal(decrypted)
   handleMessage(msg)
```

**架构优势：**
- ✅ **安全性高**：只信任服务器签名的消息
- ✅ **防伪造**：客户端无法伪造服务器签名
- ✅ **权限控制**：服务器可以拦截违规消息
- ✅ **审计能力**：服务器记录所有消息
- ✅ **简单实现**：客户端逻辑简化

---

### 2.3 可靠性保证

#### 2.3.1 ACK 机制

```go
type ACKFrame struct {
    Sequence       uint32
    ReceivedChunks []uint16  // 已接收的块索引
    MissingChunks  []uint16  // 丢失的块索引（请求重传）
}

// 发送方维护未确认队列
type UnackedFrame struct {
    Frame       *ARPFrame
    SendTime    time.Time
    RetryCount  int
}

var unackedFrames = make(map[uint32][]*UnackedFrame)

// 超时重传
func retransmitLoop() {
    ticker := time.NewTicker(500 * time.Millisecond)
    for range ticker.C {
        now := time.Now()
        for seq, frames := range unackedFrames {
            for _, uf := range frames {
                if now.Sub(uf.SendTime) > 2*time.Second {
                    if uf.RetryCount < 3 {
                        resendFrame(uf.Frame)
                        uf.SendTime = now
                        uf.RetryCount++
                    } else {
                        // 放弃重传，标记为失败
                        reportTransmitFailure(seq)
                    }
                }
            }
        }
    }
}
```

---

#### 2.3.2 流量控制

**滑动窗口：**

```go
type SlidingWindow struct {
    Size        int           // 窗口大小
    Sent        int           // 已发送帧数
    Acked       int           // 已确认帧数
    InFlight    int           // 在途帧数
}

func (w *SlidingWindow) CanSend() bool {
    return w.InFlight < w.Size
}

func (w *SlidingWindow) SendFrame(frame *ARPFrame) {
    for !w.CanSend() {
        time.Sleep(10 * time.Millisecond)
    }
    sendFrameToNetwork(frame)
    w.Sent++
    w.InFlight++
}

func (w *SlidingWindow) OnACK(sequence uint32) {
    w.Acked++
    w.InFlight--
}
```

**动态调整窗口大小：**

```go
func adjustWindowSize(lossRate float64) int {
    if lossRate < 0.01 {
        return 16  // 低丢包率，增大窗口
    } else if lossRate < 0.05 {
        return 8
    } else {
        return 4   // 高丢包率，减小窗口
    }
}
```

---

### 2.4 服务器签名模式实现

**架构：星型拓扑**

```
Client A ──┐
           ├──→ Server (签名) ──→ Broadcast ──→ All Clients
Client B ──┘
```

**服务端实现：**

```go
type ARPServer struct {
    iface         *net.Interface
    handle        *pcap.Handle
    channelKey    []byte
    privateKey    ed25519.PrivateKey  // 服务器私钥
    publicKey     ed25519.PublicKey   // 服务器公钥
    members       map[string]*Member
}

// 接收客户端消息（单播）
func (s *ARPServer) ReceiveClientMessage() {
    for {
        frame, _ := s.receiveRawFrame()
        
        // 只接收发给服务器的单播消息
        if !bytes.Equal(frame.DstMAC, s.iface.HardwareAddr) {
            continue
        }
        
        // 解密
        decrypted, err := s.decrypt(frame.Payload)
        if err != nil {
            continue
        }
        
        // 反序列化
        msg := &Message{}
        json.Unmarshal(decrypted, msg)
        
        // 验证权限
        if !s.validateMember(msg.SenderID) {
            continue  // 非法成员，丢弃
        }
        
        if s.isMuted(msg.SenderID) {
            continue  // 被禁言，丢弃
        }
        
        // 签名后广播
        s.signAndBroadcast(msg, decrypted)
    }
}

// 签名并广播
func (s *ARPServer) signAndBroadcast(msg *Message, encrypted []byte) {
    // 1. 使用服务器私钥签名
    signature := ed25519.Sign(s.privateKey, encrypted)
    
    // 2. 构造广播帧
    payload := &SignedPayload{
        Message:   encrypted,
        Signature: signature,
        Timestamp: time.Now().UnixNano(),
    }
    
    payloadBytes, _ := json.Marshal(payload)
    
    // 3. 广播
    frame := &ARPFrame{
        DstMAC:  BROADCAST_MAC,
        SrcMAC:  s.iface.HardwareAddr,
        Type:    FrameTypeServerBroadcast,
        Payload: payloadBytes,
    }
    
    s.sendRawFrame(frame)
    
    // 4. 持久化
    s.db.SaveMessage(msg)
}
```

**客户端实现：**

```go
type ARPClient struct {
    iface         *net.Interface
    handle        *pcap.Handle
    channelKey    []byte
    serverMAC     net.HardwareAddr
    serverPubKey  ed25519.PublicKey  // 服务器公钥
}

// 发送消息（单播给服务器）
func (c *ARPClient) SendMessage(content string) error {
    // 1. 构造消息
    msg := &Message{
        ID:        uuid.New().String(),
        SenderID:  c.userID,
        Type:      "text",
        Content:   content,
        Timestamp: time.Now().UnixNano(),
    }
    
    // 2. 序列化并加密
    data, _ := json.Marshal(msg)
    encrypted, _ := c.encrypt(data)
    
    // 3. 单播给服务器
    frame := &ARPFrame{
        DstMAC:  c.serverMAC,  // 单播给服务器
        SrcMAC:  c.iface.HardwareAddr,
        Type:    FrameTypeClientMessage,
        Payload: encrypted,
    }
    
    return c.sendRawFrame(frame)
}

// 接收广播消息
func (c *ARPClient) ReceiveLoop() {
    for {
        frame, _ := c.receiveRawFrame()
        
        // 1. 只接收来自服务器的广播
        if !frame.DstMAC.IsBroadcast() {
            continue
        }
        
        if !bytes.Equal(frame.SrcMAC, c.serverMAC) {
            continue  // 不是服务器发的，忽略
        }
        
        // 2. 解析签名载荷
        payload := &SignedPayload{}
        json.Unmarshal(frame.Payload, payload)
        
        // 3. 验证服务器签名（关键！）
        if !ed25519.Verify(c.serverPubKey, payload.Message, payload.Signature) {
            continue  // 签名无效，拒绝
        }
        
        // 4. 解密消息
        decrypted, err := c.decrypt(payload.Message)
        if err != nil {
            continue
        }
        
        // 5. 处理消息
        msg := &Message{}
        json.Unmarshal(decrypted, msg)
        c.handleMessage(msg)
    }
}
```

**安全优势：**
- ✅ **防消息伪造**：客户端无法伪造服务器签名
- ✅ **权限控制**：服务器过滤非法消息
- ✅ **防中间人攻击**：客户端验证服务器公钥
- ✅ **审计完整**：服务器记录所有消息
- ✅ **禁言生效**：服务器拦截被禁言用户的消息

---

## 3. HTTPS 传输协议

### 3.1 WebSocket 子协议

#### 3.1.1 握手

**客户端请求：**

```http
GET /ws HTTP/1.1
Host: server.local:8443
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Protocol: crosswire-v1
Sec-WebSocket-Version: 13
```

**服务端响应：**

```http
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
Sec-WebSocket-Protocol: crosswire-v1
```

---

#### 3.1.2 消息格式

**WebSocket 帧结构：**

```
FIN: 1 (最后一帧)
Opcode: 0x2 (Binary)
Payload:
  ┌─────────────────────────────────────┐
  │ CrossWire Message Header (8 bytes)  │
  ├─────────────────────────────────────┤
  │ Encrypted Payload (变长)            │
  └─────────────────────────────────────┘
```

**CrossWire 消息头：**

```
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
├─────────────────────────────────────────────────────────────┤
│   Version   │  Msg Type   │            Reserved             │
├─────────────────────────────────────────────────────────────┤
│                       Payload Length                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│                    Encrypted Payload                          │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

| 字段 | 长度 | 说明 |
|------|------|------|
| Version | 1 | 协议版本 |
| Msg Type | 1 | 消息类型（见下表）|
| Reserved | 2 | 预留 |
| Payload Length | 4 | 负载长度 |
| Encrypted Payload | 变长 | 加密数据 |

**消息类型：**

```go
const (
    MsgTypeData       = 0x01  // 数据消息
    MsgTypeFileChunk  = 0x02  // 文件分块
    MsgTypeControl    = 0x03  // 控制消息
    MsgTypeHeartbeat  = 0x04  // 心跳
    MsgTypeSync       = 0x05  // 同步请求
)
```

---

#### 3.1.3 心跳机制

**客户端发送 Ping：**

```
每 30 秒发送一次:
  WebSocket Ping Frame
  Payload: { "timestamp": 1696512000 }
```

**服务端响应 Pong：**

```
WebSocket Pong Frame
Payload: { "timestamp": 1696512000 }
```

**超时检测：**

```go
func (c *WSClient) HeartbeatLoop() {
    ticker := time.NewTicker(30 * time.Second)
    timeout := time.NewTimer(90 * time.Second)
    
    for {
        select {
        case <-ticker.C:
            c.SendPing()
            timeout.Reset(90 * time.Second)
        
        case <-timeout.C:
            c.Disconnect(ErrHeartbeatTimeout)
            return
        
        case <-c.pongReceived:
            timeout.Reset(90 * time.Second)
        }
    }
}
```

---

### 3.2 HTTP API

#### 3.2.1 文件上传

**请求：**

```http
POST /api/file/upload HTTP/1.1
Host: server.local:8443
Authorization: Bearer <JWT_TOKEN>
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary

------WebKitFormBoundary
Content-Disposition: form-data; name="file"; filename="exploit.py"
Content-Type: application/x-python

#!/usr/bin/env python3
import pwn
...

------WebKitFormBoundary--
```

**响应：**

```json
{
  "success": true,
  "file_id": "uuid-v4",
  "filename": "exploit.py",
  "size": 2048,
  "sha256": "hash...",
  "url": "/api/file/download/uuid-v4"
}
```

---

#### 3.2.2 文件下载

**请求：**

```http
GET /api/file/download/<file_id> HTTP/1.1
Host: server.local:8443
Authorization: Bearer <JWT_TOKEN>
Range: bytes=0-1023
```

**响应：**

```http
HTTP/1.1 206 Partial Content
Content-Type: application/octet-stream
Content-Length: 1024
Content-Range: bytes 0-1023/2048
Content-Disposition: attachment; filename="exploit.py"

<binary data>
```

---

#### 3.2.3 消息同步

**请求：**

```http
POST /api/sync HTTP/1.1
Host: server.local:8443
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "channel_id": "uuid",
  "last_message_id": "msg-uuid",
  "last_timestamp": 1696512000,
  "limit": 100
}
```

**响应：**

```json
{
  "messages": [
    {
      "id": "msg-uuid-1",
      "sender_id": "user-uuid",
      "sender_nickname": "alice",
      "type": "text",
      "content": "Hello",
      "timestamp": 1696512100
    },
    ...
  ],
  "has_more": false,
  "next_cursor": null
}
```

---

### 3.3 TLS 配置

#### 3.3.1 证书生成

**自签名证书（开发/局域网）：**

```bash
openssl req -x509 -newkey rsa:4096 \
  -keyout key.pem -out cert.pem \
  -days 365 -nodes \
  -subj "/CN=crosswire.local"
```

**Go 代码生成：**

```go
import "crypto/tls"
import "crypto/x509"

func GenerateSelfSignedCert() (tls.Certificate, error) {
    priv, _ := rsa.GenerateKey(rand.Reader, 4096)
    
    template := x509.Certificate{
        SerialNumber: big.NewInt(1),
        Subject: pkix.Name{
            CommonName: "CrossWire Server",
        },
        NotBefore: time.Now(),
        NotAfter:  time.Now().Add(365 * 24 * time.Hour),
        KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
        IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
        DNSNames:    []string{"localhost", "*.local"},
    }
    
    certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
    
    return tls.Certificate{
        Certificate: [][]byte{certDER},
        PrivateKey:  priv,
    }, nil
}
```

---

#### 3.3.2 TLS 配置

```go
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS13,
    MaxVersion: tls.VersionTLS13,
    
    CipherSuites: []uint16{
        tls.TLS_AES_256_GCM_SHA384,
        tls.TLS_CHACHA20_POLY1305_SHA256,
    },
    
    CurvePreferences: []tls.CurveID{
        tls.X25519,
        tls.CurveP256,
    },
    
    Certificates: []tls.Certificate{cert},
    
    // 客户端配置（跳过验证仅用于局域网）
    InsecureSkipVerify: true,
}
```

---

## 4. mDNS 传输协议

> **设计原则：服务器签名模式**  
> 与 ARP 模式类似，mDNS 也采用星型拓扑架构：客户端 → 服务器 → mDNS 广播

---

### 4.1 架构设计

#### 4.1.1 星型拓扑

```
Client A (UDP) ─┐
                ├─→ Server (签名验证) ─→ mDNS Multicast ─→ All Clients
Client B (UDP) ─┘
```

**流程：**
1. 客户端通过 UDP 单播消息给服务器
2. 服务器验证权限并签名
3. 服务器通过 mDNS 服务实例名组播消息
4. 所有客户端监听 mDNS，验证签名后处理

---

### 4.2 服务发现协议

#### 4.2.1 服务注册

**CrossWire 服务类型：**

```
_crosswire._udp.local        # 频道发现
_crosswire-msg._udp.local    # 消息广播（服务器签名）
_crosswire-ctl._udp.local    # 控制消息
```

**频道发现示例：**

```go
import "github.com/hashicorp/mdns"

// 服务器注册频道信息
info := &mdns.ServiceInfo{
    ServiceName: "CTF-Team-Alpha",
    ServiceType: "_crosswire._udp",
    Domain:      "local",
    HostName:    "server.local",
    Port:        5353,  // 服务器监听端口
    IPs:         []net.IP{net.ParseIP("192.168.1.100")},
    TXT: []string{
        "version=1",
        "protocol=server-signed",  // 标识使用签名模式
        "pubkey=<base64_server_public_key>",  // 服务器公钥
        "members=5",
    },
}

server, _ := mdns.NewServer(&mdns.Config{Zone: info})
defer server.Shutdown()
```

---

#### 4.2.2 频道查询

**客户端查询服务器：**

```go
entriesCh := make(chan *mdns.ServiceEntry, 10)

mdns.Lookup("_crosswire._udp", entriesCh)

for entry := range entriesCh {
    fmt.Printf("发现频道: %s\n", entry.Name)
    fmt.Printf("  服务器: %s:%d\n", entry.AddrV4, entry.Port)
    
    // 提取服务器公钥
    for _, txt := range entry.InfoFields {
        if strings.HasPrefix(txt, "pubkey=") {
            serverPubKey := base64.StdEncoding.DecodeString(txt[7:])
            // 保存公钥用于验证签名
        }
    }
}
```

---

### 4.3 消息传输流程（服务器签名模式）

#### 4.3.1 客户端发送消息

**客户端 → 服务器（UDP 单播）：**

```go
type mDNSClient struct {
    serverAddr   *net.UDPAddr
    serverPubKey ed25519.PublicKey
    channelKey   []byte
    conn         *net.UDPConn
}

// 发送消息到服务器
func (c *mDNSClient) SendMessage(content string) error {
    // 1. 构造消息
    msg := &Message{
        ID:        uuid.New().String(),
        SenderID:  c.userID,
        Type:      "text",
        Content:   content,
        Timestamp: time.Now().UnixNano(),
    }
    
    // 2. 序列化并加密
    data, _ := json.Marshal(msg)
    encrypted, _ := encryptAES256GCM(data, c.channelKey)
    
    // 3. UDP 单播给服务器
    _, err := c.conn.WriteToUDP(encrypted, c.serverAddr)
    return err
}
```

---

#### 4.3.2 服务器处理与签名

**服务器接收、验证、签名、广播：**

```go
type mDNSServer struct {
    conn        *net.UDPConn
    privateKey  ed25519.PrivateKey
    publicKey   ed25519.PublicKey
    channelKey  []byte
    members     map[string]*Member
}

// 监听客户端消息
func (s *mDNSServer) ReceiveLoop() {
    buf := make([]byte, 4096)
    
    for {
        n, addr, _ := s.conn.ReadFromUDP(buf)
        
        // 解密
        decrypted, err := decryptAES256GCM(buf[:n], s.channelKey)
        if err != nil {
            continue
        }
        
        // 反序列化
        msg := &Message{}
        json.Unmarshal(decrypted, msg)
        
        // 验证权限
        if !s.validateMember(msg.SenderID) {
            log.Warn("Invalid member:", msg.SenderID)
            continue
        }
        
        if s.isMuted(msg.SenderID) {
            log.Info("Muted user blocked:", msg.SenderID)
            continue
        }
        
        // 签名后通过 mDNS 广播
        s.signAndBroadcastViaMDNS(msg, decrypted)
    }
}

// 通过 mDNS 广播已签名消息
func (s *mDNSServer) signAndBroadcastViaMDNS(msg *Message, encrypted []byte) {
    // 1. 签名
    signature := ed25519.Sign(s.privateKey, encrypted)
    
    // 2. 构造载荷
    payload := &SignedPayload{
        Message:   encrypted,
        Signature: signature,
        Timestamp: time.Now().UnixNano(),
    }
    
    payloadBytes, _ := json.Marshal(payload)
    
    // 3. 编码为 mDNS 服务实例名
    s.broadcastViaMDNSService(msg.ID, payloadBytes)
    
    // 4. 持久化
    s.db.SaveMessage(msg)
}
```

---

#### 4.3.3 服务实例名编码（包含签名）

**编码格式：**

```
<msgid>-<seq>-<data>._crosswire-msg._udp.local

msgid: 消息ID前6字符
seq:   序列号（3位数字，000-999）
data:  Base64URL编码的已签名载荷（签名+加密消息）
```

**广播实现：**

```go
func (s *mDNSServer) broadcastViaMDNSService(msgID string, signedPayload []byte) {
    // 1. Base64URL 编码（URL 安全）
    encoded := base64.URLEncoding.EncodeToString(signedPayload)
    
    // 2. 分块（DNS 标签最大 63 字符）
    const chunkSize = 50
    chunks := splitIntoChunks(encoded, chunkSize)
    
    // 3. 为每个块注册 mDNS 服务
    msgIDShort := msgID[:6]
    for i, chunk := range chunks {
        instanceName := fmt.Sprintf("%s-%03d-%s", msgIDShort, i, chunk)
        
        info := &mdns.ServiceInfo{
            ServiceName: instanceName,
            ServiceType: "_crosswire-msg._udp",
            Domain:      "local",
            Port:        0,
            TXT: []string{
                fmt.Sprintf("total=%d", len(chunks)),
            },
        }
        
        // 注册服务（5秒后自动注销）
        server, _ := mdns.NewServer(&mdns.Config{Zone: info})
        time.AfterFunc(5*time.Second, server.Shutdown)
    }
}
```

**示例：**

```
原始消息: "Hello CTF"

1. 序列化: {"type":"text","content":"Hello CTF"}
2. 加密: X25519(message, channel_key)
3. 服务器签名: Ed25519(encrypted, server_private_key)
4. 构造载荷: {
     "message": encrypted,
     "signature": signature,
     "timestamp": 1696512000
   }
5. Base64URL编码
6. 分块并注册:
   - 8f3c9a-000-eyJtZXNzYWdlIjoi...._crosswire-msg._udp.local
   - 8f3c9a-001-InNpZ25hdHVyZSI6...._crosswire-msg._udp.local
```

---

#### 4.3.4 客户端接收与验证签名

**监听 mDNS 服务：**

```go
type mDNSClient struct {
    serverPubKey ed25519.PublicKey
    channelKey   []byte
    assembler    *MessageAssembler
}

// 监听消息服务
func (c *mDNSClient) ReceiveLoop() {
    entriesCh := make(chan *mdns.ServiceEntry, 100)
    
    // 监听消息服务
    go mdns.Lookup("_crosswire-msg._udp", entriesCh)
    
    for entry := range entriesCh {
        // 解析服务实例名
        parts := strings.Split(entry.Name, "-")
        if len(parts) < 3 {
            continue
        }
        
        msgID := parts[0]
        seq, _ := strconv.Atoi(parts[1])
        data := parts[2]
        
        // 添加到分块重组器
        if assembled := c.assembler.AddChunk(msgID, seq, data); assembled != "" {
            // 重组完成，验证签名
            c.verifyAndProcess(assembled)
        }
    }
}

// 验证签名并处理消息
func (c *mDNSClient) verifyAndProcess(encodedPayload string) {
    // 1. Base64URL 解码
    payloadBytes, _ := base64.URLEncoding.DecodeString(encodedPayload)
    
    // 2. 反序列化签名载荷
    payload := &SignedPayload{}
    json.Unmarshal(payloadBytes, payload)
    
    // 3. 验证服务器签名（关键！）
    if !ed25519.Verify(c.serverPubKey, payload.Message, payload.Signature) {
        log.Warn("Invalid signature, possible attack!")
        return  // 拒绝伪造消息
    }
    
    // 4. 验证时间戳（防重放）
    if abs(time.Now().UnixNano()-payload.Timestamp) > 5*60*1e9 {
        log.Warn("Message too old, possible replay attack")
        return
    }
    
    // 5. 解密消息
    decrypted, err := decryptAES256GCM(payload.Message, c.channelKey)
    if err != nil {
        log.Warn("Decryption failed")
        return
    }
    
    // 6. 反序列化并处理
    msg := &Message{}
    json.Unmarshal(decrypted, msg)
    c.handleMessage(msg)
}
```

**消息重组器：**

```go
type MessageAssembler struct {
    chunks map[string]*ChunkSet
    mutex  sync.Mutex
}

type ChunkSet struct {
    data      map[int]string
    total     int
    lastSeen  time.Time
}

func (a *MessageAssembler) AddChunk(msgID string, seq int, data string) string {
    a.mutex.Lock()
    defer a.mutex.Unlock()
    
    if _, exists := a.chunks[msgID]; !exists {
        a.chunks[msgID] = &ChunkSet{
            data:     make(map[int]string),
            lastSeen: time.Now(),
        }
    }
    
    set := a.chunks[msgID]
    set.data[seq] = data
    set.lastSeen = time.Now()
    
    // 检查是否完整
    if a.isComplete(set) {
        assembled := a.assemble(set)
        delete(a.chunks, msgID)
        return assembled
    }
    
    return ""
}

func (a *MessageAssembler) isComplete(set *ChunkSet) bool {
    // 检查序号是否连续
    for i := 0; i < len(set.data); i++ {
        if _, exists := set.data[i]; !exists {
            return false
        }
    }
    return true
}

func (a *MessageAssembler) assemble(set *ChunkSet) string {
    // 按顺序拼接
    var result strings.Builder
    for i := 0; i < len(set.data); i++ {
        result.WriteString(set.data[i])
    }
    return result.String()
}
```

---

### 4.4 安全优势

#### 4.4.1 多层防护

与 ARP 模式类似，mDNS 也采用多层安全防护：

1. **服务器签名验证**
   - 客户端只接受服务器签名的消息
   - Ed25519 签名无法伪造
   
2. **服务器权限控制**
   - 服务器验证成员身份
   - 服务器拦截被禁言用户
   - 频率限制防刷屏
   
3. **X25519 加密**
   - 不同频道使用不同密钥
   - 无法解密其他频道消息
   
4. **防重放攻击**
   - 时间戳验证
   - Nonce 去重

**对比传统 mDNS 数据传输：**

| 方面 | 传统 mDNS | 服务器签名模式 |
|------|----------|----------------|
| 消息伪造 | ❌ 无防护 | ✅ Ed25519 签名防护 |
| 权限控制 | ❌ 无 | ✅ 服务器验证 |
| 禁言功能 | ❌ 无效 | ✅ 服务器拦截 |
| 消息审计 | ❌ 不完整 | ✅ 服务器完整记录 |

---

### 4.5 性能优化

#### 4.5.1 自适应速率

```go
type RateLimiter struct {
    interval    time.Duration
    lossRate    float64
    successCount int
    failCount   int
}

func (r *RateLimiter) AdjustRate() {
    r.lossRate = float64(r.failCount) / float64(r.successCount + r.failCount)
    
    if r.lossRate < 0.05 {
        // 低丢包率，加速
        r.interval = time.Duration(float64(r.interval) * 0.9)
        if r.interval < 50*time.Millisecond {
            r.interval = 50 * time.Millisecond
        }
    } else if r.lossRate > 0.20 {
        // 高丢包率，减速
        r.interval = time.Duration(float64(r.interval) * 1.5)
        if r.interval > 2*time.Second {
            r.interval = 2 * time.Second
        }
    }
    
    // 重置计数
    r.successCount = 0
    r.failCount = 0
}
```

---

## 5. 应用层协议

### 5.1 消息格式

#### 5.1.1 通用消息结构

```protobuf
message Message {
    string id = 1;              // UUID
    string channel_id = 2;      // 频道 ID
    string sender_id = 3;       // 发送者 ID
    MessageType type = 4;       // 消息类型
    bytes content = 5;          // 消息内容（加密）
    int64 timestamp = 6;        // 时间戳（Unix 纳秒）
    map<string, string> metadata = 7;  // 元数据
    bytes signature = 8;        // 消息签名
}

enum MessageType {
    TEXT = 0;
    CODE = 1;
    FILE = 2;
    SYSTEM = 3;
    CONTROL = 4;
}
```

**JSON 格式（加密前）：**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "channel_id": "channel-uuid",
  "sender_id": "user-uuid",
  "type": "text",
  "content": {
    "text": "发现一个 SQL 注入点",
    "format": "markdown",
    "mentions": ["user-uuid-2"],
    "tags": ["web", "sqli"]
  },
  "timestamp": 1696512000000000000,
  "metadata": {
    "client_version": "1.0.0",
    "platform": "windows"
  }
}
```

---

#### 5.1.2 文本消息

```json
{
  "type": "text",
  "content": {
    "text": "消息内容",
    "format": "plain|markdown|html",
    "mentions": ["user-id-1", "user-id-2"],
    "tags": ["tag1", "tag2"],
    "reply_to": "message-id"  // 可选，回复某条消息
  }
}
```

---

#### 5.1.3 代码消息

```json
{
  "type": "code",
  "content": {
    "language": "python",
    "code": "import pwn\n...",
    "filename": "exploit.py",
    "description": "Pwn-300 exp 脚本",
    "highlighted": true
  }
}
```

---

#### 5.1.4 文件消息

```json
{
  "type": "file",
  "content": {
    "file_id": "uuid",
    "filename": "exploit.py",
    "size": 2048,
    "mime_type": "application/x-python",
    "sha256": "hash...",
    "thumbnail": "base64...",  // 可选
    "expires_at": 1696598400
  }
}
```

---

#### 5.1.5 系统消息

```json
{
  "type": "system",
  "content": {
    "event": "member_join|member_leave|member_kicked|message_pinned",
    "actor_id": "user-uuid",
    "target_id": "user-uuid",
    "reason": "可选原因",
    "extra": {}
  }
}
```

---

#### 5.1.6 控制消息

```json
{
  "type": "control",
  "content": {
    "command": "heartbeat|sync|disconnect|update_status",
    "params": {}
  }
}
```

---

### 5.2 文件传输协议

#### 5.2.1 文件元数据

```json
{
  "id": "file-uuid",
  "filename": "payload.bin",
  "size": 1048576,
  "mime_type": "application/octet-stream",
  "sha256": "hash...",
  "chunk_size": 8192,
  "total_chunks": 128,
  "encryption": "X25519",
  "compression": "gzip"
}
```

---

#### 5.2.2 分块传输

**分块格式：**

```json
{
  "file_id": "file-uuid",
  "chunk_index": 0,
  "total_chunks": 128,
  "data": "base64_encoded_encrypted_data",
  "checksum": "sha256_hash"
}
```

**传输流程：**

```
1. 发送方发送文件元数据
   Client -> Server: FILE_META

2. 服务端准备接收
   Server -> Client: FILE_READY

3. 发送所有分块
   For each chunk:
     Client -> Server: FILE_CHUNK

4. 接收方定期发送进度确认
   Server -> Client: FILE_PROGRESS
   {
     "file_id": "uuid",
     "received_chunks": [0, 1, 2, ...],
     "progress": 0.45
   }

5. 传输完成
   Client -> Server: FILE_COMPLETE
   Server -> Client: FILE_ACK
```

---

### 5.3 同步协议

#### 5.3.1 增量同步

**同步请求：**

```json
{
  "type": "sync_request",
  "channel_id": "channel-uuid",
  "last_message_id": "msg-uuid",
  "last_timestamp": 1696512000,
  "cursor": "optional_cursor",
  "limit": 100
}
```

**同步响应：**

```json
{
  "type": "sync_response",
  "messages": [
    // Message objects
  ],
  "members": [
    // Member objects (if changed)
  ],
  "has_more": false,
  "next_cursor": null
}
```

---

#### 5.3.2 冲突解决

**时间戳 + UUID 排序：**

```go
func CompareMessages(a, b *Message) int {
    if a.Timestamp != b.Timestamp {
        return int(a.Timestamp - b.Timestamp)
    }
    return strings.Compare(a.ID, b.ID)
}
```

**Last-Write-Wins（最后写入获胜）：**

```go
func ResolveConflict(local, remote *Message) *Message {
    if CompareMessages(remote, local) > 0 {
        return remote  // 使用远程版本
    }
    return local  // 保留本地版本
}
```

---

## 6. 加密与安全

### 6.1 密钥体系

```
Channel Key (AES-256)
    ├─ 由频道密码派生（PBKDF2/Argon2）
    ├─ 用于加密消息内容
    └─ 所有成员共享

User Key Pair (X25519)
    ├─ 公钥：用于接收加密的 Channel Key
    └─ 私钥：保存在本地，不传输

Session Key (AES-256)
    ├─ 每次连接随机生成
    ├─ 用于加密控制消息
    └─ 通过 RSA 加密交换
```

---

### 6.2 加密算法

#### 6.2.1 频道密钥派生

```go
import "golang.org/x/crypto/argon2"

func DeriveChannelKey(password string, salt []byte) []byte {
    return argon2.IDKey(
        []byte(password),
        salt,
        1,          // time cost
        64*1024,    // memory cost (64MB)
        4,          // parallelism
        32,         // key length (256 bits)
    )
}
```

---

#### 6.2.2 消息加密

**X25519：**

```go
import "crypto/aes"
import "crypto/cipher"

func EncryptMessage(plaintext, key []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    
    nonce := make([]byte, gcm.NonceSize())
    rand.Read(nonce)
    
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}

func DecryptMessage(ciphertext, key []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    
    nonceSize := gcm.NonceSize()
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    return plaintext, err
}
```

---

#### 6.2.3 消息签名

**HMAC-SHA256：**

```go
import "crypto/hmac"
import "crypto/sha256"

func SignMessage(message []byte, key []byte) []byte {
    mac := hmac.New(sha256.New, key)
    mac.Write(message)
    return mac.Sum(nil)
}

func VerifySignature(message, signature, key []byte) bool {
    expectedMAC := SignMessage(message, key)
    return hmac.Equal(signature, expectedMAC)
}
```

---

### 6.3 安全特性

#### 6.3.1 防重放攻击

**Nonce + Timestamp：**

```go
type MessageEnvelope struct {
    Nonce     [16]byte
    Timestamp int64
    Message   []byte
}

var usedNonces = make(map[[16]byte]int64)
var nonceMutex sync.Mutex

func ValidateNonce(nonce [16]byte, timestamp int64) bool {
    nonceMutex.Lock()
    defer nonceMutex.Unlock()
    
    // 检查时间戳（容忍 5 分钟偏差）
    if math.Abs(float64(time.Now().Unix()-timestamp)) > 300 {
        return false
    }
    
    // 检查 nonce 是否已使用
    if _, exists := usedNonces[nonce]; exists {
        return false
    }
    
    // 记录 nonce
    usedNonces[nonce] = timestamp
    
    // 清理过期 nonce（超过 10 分钟）
    for n, ts := range usedNonces {
        if time.Now().Unix()-ts > 600 {
            delete(usedNonces, n)
        }
    }
    
    return true
}
```

---

#### 6.3.2 密钥轮换

```go
func RotateChannelKey(channel *Channel) error {
    // 生成新密钥
    newKey := make([]byte, 32)
    rand.Read(newKey)
    
    // 用每个成员的公钥加密新密钥
    for _, member := range channel.Members {
        encryptedKey, _ := rsa.EncryptOAEP(
            sha256.New(),
            rand.Reader,
            member.PublicKey,
            newKey,
            nil,
        )
        
        // 发送加密后的新密钥
        channel.SendControlMessage(member.ID, &ControlMessage{
            Type: "key_rotation",
            Data: encryptedKey,
        })
    }
    
    // 更新频道密钥
    channel.EncryptionKey = newKey
    
    return nil
}
```

---

## 7. 协议扩展

### 7.1 自定义字段

**保留字段：**

```go
type Message struct {
    // ... 标准字段
    
    Extensions map[string][]byte  // 扩展字段
}

// 使用示例
msg.Extensions["custom_field"] = customData
```

---

### 7.2 协议版本协商

```go
type ProtocolVersion struct {
    Major int
    Minor int
    Patch int
}

func NegotiateProtocol(clientVer, serverVer ProtocolVersion) ProtocolVersion {
    // 使用较低的主版本
    if clientVer.Major != serverVer.Major {
        return ProtocolVersion{
            Major: min(clientVer.Major, serverVer.Major),
            Minor: 0,
            Patch: 0,
        }
    }
    
    // 主版本相同，使用较低的次版本
    return ProtocolVersion{
        Major: clientVer.Major,
        Minor: min(clientVer.Minor, serverVer.Minor),
        Patch: 0,
    }
}
```

---

### 7.3 协议升级

**协议切换流程：**

```
1. 客户端检测到服务端支持更好的协议
   Client: "Detected server supports HTTPS, current: ARP"

2. 发送升级请求
   Client -> Server: PROTOCOL_UPGRADE_REQUEST
   {
     "current_protocol": "arp",
     "target_protocol": "https",
     "reason": "better_performance"
   }

3. 服务端准备升级
   Server -> Client: PROTOCOL_UPGRADE_READY
   {
     "https_endpoint": "wss://server.local:8443/ws",
     "migration_token": "temporary_token"
   }

4. 客户端建立新连接
   Client connects to HTTPS endpoint

5. 断开旧连接
   Client disconnects ARP transport
```

---

## 总结

CrossWire 协议设计特点：

✅ **多传输支持**：ARP/HTTPS/mDNS 三种传输方式  
✅ **可靠传输**：ACK、重传、滑动窗口  
✅ **安全加密**：X25519 + RSA 4096  
✅ **灵活扩展**：支持自定义字段和协议升级  
✅ **性能优化**：分块传输、增量同步、自适应速率  

---

**相关文档：**
- [FEATURES.md](FEATURES.md) - 功能详细说明
- [ARCHITECTURE.md](ARCHITECTURE.md) - 系统架构设计
- [DATABASE.md](DATABASE.md) - 数据库设计
