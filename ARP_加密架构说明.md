# ARP传输层加密架构说明

> CrossWire项目的分层加密设计

---

## 🔐 加密架构概览

CrossWire采用**分层加密**设计：

```
┌─────────────────────────────────────────────┐
│         应用层 (Client/Server)              │
│  - 构造业务消息 (JSON)                       │
│  - 使用 crypto.EncryptMessage() 加密         │
└─────────────────────────────────────────────┘
                    ↓
         [AES-256-GCM加密后的数据]
                    ↓
┌─────────────────────────────────────────────┐
│          传输层 (ARP/mDNS/HTTPS)            │
│  - 接收已加密的 Payload                      │
│  - 添加传输层签名 (Ed25519)                  │
│  - 分块、发送、接收、重组                     │
└─────────────────────────────────────────────┘
                    ↓
         [网络传输 - 以太网帧]
                    ↓
┌─────────────────────────────────────────────┐
│          传输层 (接收端)                     │
│  - 接收、重组、验证签名                       │
│  - 返回已加密的 Payload                      │
└─────────────────────────────────────────────┘
                    ↓
         [AES-256-GCM加密后的数据]
                    ↓
┌─────────────────────────────────────────────┐
│         应用层 (Client/Server)              │
│  - 使用 crypto.DecryptMessage() 解密         │
│  - 解析业务消息 (JSON)                       │
└─────────────────────────────────────────────┘
```

---

## 📍 加密实现位置

### 1. **AES-256-GCM加密** (`internal/crypto/crypto.go`)

**实现位置:** Lines 30-83

#### 加密方法

```go
func (m *Manager) AESEncrypt(plaintext, key []byte) ([]byte, error) {
    // 1. 使用32字节密钥创建AES cipher
    block, err := aes.NewCipher(key)  // AES-256
    
    // 2. 创建GCM模式（认证加密）
    gcm, err := cipher.NewGCM(block)
    
    // 3. 生成随机nonce
    nonce := make([]byte, gcm.NonceSize())  // 12字节
    io.ReadFull(rand.Reader, nonce)
    
    // 4. 加密并附加认证标签
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    
    return ciphertext, nil  // [nonce][密文+认证标签]
}
```

**格式:**
```
[12 bytes Nonce][N bytes Ciphertext + 16 bytes Auth Tag]
```

#### 解密方法

```go
func (m *Manager) AESDecrypt(ciphertext, key []byte) ([]byte, error) {
    // 1. 创建AES cipher
    block, err := aes.NewCipher(key)
    gcm, err := cipher.NewGCM(block)
    
    // 2. 提取nonce
    nonceSize := gcm.NonceSize()  // 12
    nonce := ciphertext[:nonceSize]
    ciphertext := ciphertext[nonceSize:]
    
    // 3. 解密并验证认证标签
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    
    return plaintext, nil
}
```

---

### 2. **频道密钥管理** (`internal/crypto/crypto.go`)

**包装方法:**

```go
// EncryptMessage 使用频道密钥加密消息
func (m *Manager) EncryptMessage(plaintext []byte) ([]byte, error) {
    if m.channelKey == nil {
        return nil, fmt.Errorf("channel key not set")
    }
    return m.AESEncrypt(plaintext, m.channelKey)
}

// DecryptMessage 使用频道密钥解密消息
func (m *Manager) DecryptMessage(ciphertext []byte) ([]byte, error) {
    if m.channelKey == nil {
        return nil, fmt.Errorf("channel key not set")
    }
    return m.AESDecrypt(ciphertext, m.channelKey)
}
```

**频道密钥派生:**

```go
// DeriveKey 从密码派生密钥（PBKDF2）
func (m *Manager) DeriveKey(password string, salt []byte) ([]byte, error) {
    return pbkdf2.Key(
        []byte(password),
        salt,
        100000,  // 迭代次数
        32,      // 32字节 = AES-256
        sha256.New,
    ), nil
}
```

---

### 3. **应用层加密调用**

#### 客户端发送 (`internal/client/client.go`)

**示例: 发送消息**

```go
// Line 586-610
func (c *Client) SendMessageToChannel(...) error {
    // 1. 构造消息
    msg := &models.Message{...}
    msgJSON, _ := json.Marshal(msg)
    
    // 2. 签名消息
    signature := ed25519.Sign(c.privateKey, msgJSON)
    signedMsg := &SignedMessage{
        Message:   msgJSON,
        Signature: signature,
        SenderID:  c.memberID,
    }
    signedJSON, _ := json.Marshal(signedMsg)
    
    // 3. AES加密 ✅
    encrypted, err := c.crypto.EncryptMessage(signedJSON)
    
    // 4. 发送到传输层（已加密）
    transportMsg := &transport.Message{
        Type:    transport.MessageTypeData,
        Payload: encrypted,  // 加密后的数据
    }
    c.transport.SendMessage(transportMsg)
}
```

#### 服务器接收 (`internal/server/message_router.go`)

**示例: 处理客户端消息**

```go
// Line 111-140
func (mr *MessageRouter) processMessageTask(task *MessageTask) {
    // 1. AES解密 ✅
    decrypted, err := mr.server.crypto.DecryptMessage(
        task.TransportMessage.Payload,
    )
    
    // 2. 解析签名消息
    var signedMsg SignedMessage
    json.Unmarshal(decrypted, &signedMsg)
    
    // 3. 验证签名
    // ...
    
    // 4. 解析业务消息
    var msg models.Message
    json.Unmarshal(signedMsg.Message, &msg)
    
    // 5. 处理业务逻辑
    // ...
}
```

#### 服务器广播 (`internal/server/broadcast_manager.go`)

**示例: 广播消息**

```go
// Line 127-145
func (bm *BroadcastManager) Broadcast(msg *models.Message) error {
    // 1. 序列化消息
    messageData, _ := json.Marshal(msg)
    
    // 2. AES加密 ✅
    encryptedData, err := bm.server.crypto.EncryptMessage(messageData)
    
    // 3. 服务器签名
    signature := ed25519.Sign(
        bm.server.config.PrivateKey,
        encryptedData,
    )
    signedPayload := &transport.SignedPayload{
        Message:   encryptedData,  // 加密后的数据
        Signature: signature,
        Timestamp: time.Now().UnixNano(),
    }
    
    // 4. 发送到传输层（已加密+签名）
    bm.server.transport.SendMessage(&transport.Message{
        Payload: signedPayload,
    })
}
```

---

### 4. **传输层处理** (`internal/transport/arp_transport.go`)

**ARP传输层的角色:**

```go
// ✅ ARP传输层只负责：
// 1. 接收已加密的 Payload
// 2. 添加Ed25519签名（服务器模式）
// 3. 分块传输
// 4. 接收和重组
// 5. 验证Ed25519签名（客户端模式）
// 6. 返回已加密的 Payload 给上层

// ❌ ARP传输层不负责：
// - AES加密/解密（由上层crypto.Manager处理）
// - 业务逻辑处理
```

**服务器发送流程:**

```go
// Line 461-508
func (t *ARPTransport) signAndBroadcast(msg *Message) error {
    // msg.Payload 已经是加密后的数据！
    
    // 1. 使用Ed25519签名（传输层签名）
    signature := ed25519.Sign(
        ed25519.PrivateKey(t.serverPrivKey),
        msg.Payload,  // 签名的是已加密的数据
    )
    
    // 2. 构造签名载荷
    signedPayload := &SignedPayload{
        Message:   msg.Payload,  // 已加密
        Signature: signature,
        Timestamp: time.Now().UnixNano(),
    }
    
    // 3. 序列化并分块发送
    // ...
}
```

**客户端接收流程:**

```go
// Line 600-663
func (t *ARPTransport) handleFrame(frame *ARPFrame) {
    // 客户端模式
    if t.mode == "client" {
        // 1. 验证Ed25519签名（传输层签名）
        if !ed25519.Verify(
            ed25519.PublicKey(t.serverPubKey),
            signedPayload.Message,
            signedPayload.Signature,
        ) {
            return  // 签名无效
        }
        
        // 2. 构造消息（Payload仍是加密的）
        msg := &Message{
            Payload: signedPayload.Message,  // 仍是加密状态
        }
        
        // 3. 回调上层处理（上层会解密）
        t.handler(msg)
    }
}
```

---

## 🔑 密钥体系

### 密钥类型

| 密钥类型 | 算法 | 用途 | 作用域 |
|---------|------|------|-------|
| **频道密钥** | AES-256 | 消息加密 | 所有频道成员共享 |
| **服务器签名密钥** | Ed25519 | 广播消息签名 | 服务器私钥 |
| **客户端签名密钥** | Ed25519 | 用户消息签名 | 每个客户端私钥 |

### 密钥派生流程

```
频道密码 (用户输入)
     ↓
 PBKDF2-SHA256 (100,000 iterations)
     ↓
频道密钥 (32 bytes = AES-256)
     ↓
crypto.SetChannelKey()
     ↓
用于所有消息的加密/解密
```

**代码示例:**

```go
// internal/server/server.go: Lines 194-201
cryptoManager, _ := crypto.NewManager()

// 从密码派生频道密钥
channelKey, _ := cryptoManager.DeriveKey(
    config.ChannelPassword,  // 用户密码
    []byte(config.ChannelID), // 盐值（频道ID）
)

cryptoManager.SetChannelKey(channelKey)
```

---

## 🔒 双重保护机制

CrossWire实现了**双重保护**：

### 1️⃣ AES-256-GCM加密（应用层）

**保护目标:** 消息内容机密性和完整性

**特点:**
- ✅ 对称加密，性能高
- ✅ GCM模式提供认证，防篡改
- ✅ 所有频道成员共享密钥
- ✅ 即使传输层被监听，消息内容也无法读取

### 2️⃣ Ed25519签名（传输层）

**保护目标:** 消息来源认证

**特点:**
- ✅ 非对称签名，防冒充
- ✅ 服务器签名所有广播消息
- ✅ 客户端验证消息来自真实服务器
- ✅ 防止中间人攻击

---

## 🎯 为什么分层？

### 优势

1. **职责分离**
   - 应用层：处理业务逻辑和加密
   - 传输层：只负责可靠传输和来源认证

2. **灵活性**
   - 可以更换传输层（ARP/mDNS/HTTPS）而不影响加密
   - 可以调整加密算法而不影响传输

3. **安全性**
   - 双重保护：内容加密 + 来源签名
   - 即使传输层签名被破解，内容仍加密

4. **性能优化**
   - 传输层不需要加密，减少开销
   - 签名验证比加密快，可快速过滤非法消息

---

## 📊 数据流示例

### 客户端发送消息

```
┌─────────────────────────────────────────────┐
│ 1. 原始消息                                  │
│    {"type": "text", "content": "Hello"}     │
└─────────────────────────────────────────────┘
                ↓ client.SendMessage()
┌─────────────────────────────────────────────┐
│ 2. 签名消息 (应用层签名)                     │
│    {                                        │
│      "message": "{...}",                    │
│      "signature": "0x...",                  │
│      "sender_id": "alice"                   │
│    }                                        │
└─────────────────────────────────────────────┘
                ↓ crypto.EncryptMessage()
┌─────────────────────────────────────────────┐
│ 3. AES-256-GCM加密                          │
│    [nonce][encrypted_data][auth_tag]        │
│    0x4a7b2c... (密文)                       │
└─────────────────────────────────────────────┘
                ↓ transport.SendMessage()
┌─────────────────────────────────────────────┐
│ 4. 传输层处理（ARP）                         │
│    - 广播给服务器（可能是广播）               │
│    - 不做加密，直接传输密文                   │
└─────────────────────────────────────────────┘
                ↓ 网络传输
┌─────────────────────────────────────────────┐
│ 5. 服务器接收                                │
│    - 接收密文                                │
│    - 回ACK（如果是单播）                     │
└─────────────────────────────────────────────┘
                ↓ crypto.DecryptMessage()
┌─────────────────────────────────────────────┐
│ 6. AES-256-GCM解密                          │
│    {                                        │
│      "message": "{...}",                    │
│      "signature": "0x...",                  │
│      "sender_id": "alice"                   │
│    }                                        │
└─────────────────────────────────────────────┘
                ↓ 验证应用层签名
┌─────────────────────────────────────────────┐
│ 7. 解析业务消息                              │
│    {"type": "text", "content": "Hello"}     │
└─────────────────────────────────────────────┘
```

### 服务器广播消息

```
┌─────────────────────────────────────────────┐
│ 1. 业务消息                                  │
│    {"type": "text", "content": "World"}     │
└─────────────────────────────────────────────┘
                ↓ crypto.EncryptMessage()
┌─────────────────────────────────────────────┐
│ 2. AES-256-GCM加密                          │
│    0x8f3a1c... (密文)                       │
└─────────────────────────────────────────────┘
                ↓ 传输层签名
┌─────────────────────────────────────────────┐
│ 3. Ed25519签名 (传输层)                      │
│    {                                        │
│      "message": "0x8f3a1c...",  // 密文     │
│      "signature": "0xab12...",   // 传输签名│
│      "timestamp": 1234567890                │
│    }                                        │
└─────────────────────────────────────────────┘
                ↓ ARP广播
┌─────────────────────────────────────────────┐
│ 4. 以太网帧                                  │
│    DstMAC: FF:FF:FF:FF:FF:FF (广播)         │
│    Payload: SignedPayload                   │
└─────────────────────────────────────────────┘
                ↓ 所有客户端接收
┌─────────────────────────────────────────────┐
│ 5. 客户端验证传输层签名                       │
│    - 验证Ed25519签名                         │
│    - 验证时间戳（防重放）                     │
│    - 检查去重                                │
└─────────────────────────────────────────────┘
                ↓ 提取密文
┌─────────────────────────────────────────────┐
│ 6. AES-256-GCM解密                          │
│    {"type": "text", "content": "World"}     │
└─────────────────────────────────────────────┘
```

---

## 🔧 channelKey字段的作用

### 在ARP传输层中

```go
// Line 42
channelKey []byte // AES-256密钥

// Line 1046
func (t *ARPTransport) SetChannelKey(key []byte) {
    t.channelKey = key
}
```

**当前状态:** ⚠️ **已定义但未使用**

### 为什么定义？

这是为了**未来扩展**预留的接口：

1. **可选的传输层加密**
   - 如果需要双重加密（应用层+传输层）
   - 可以在传输层再次加密Payload

2. **接口统一**
   - 所有传输层都有相同的接口
   - 便于在不同传输层之间切换

3. **灵活配置**
   - 可以选择是否启用传输层加密
   - 高安全场景可以启用双重加密

### 当前实现

**实际使用的是上层加密：**

```go
// ✅ 当前架构（推荐）
Client → crypto.EncryptMessage() → Transport → Network

// ❌ 如果使用传输层加密（双重加密，开销大）
Client → crypto.EncryptMessage() → Transport.Encrypt() → Network
```

---

## ✅ 总结

### AES加密实现位置

| 层级 | 位置 | 方法 | 作用 |
|-----|------|------|------|
| **加密核心** | `internal/crypto/crypto.go` | `AESEncrypt/AESDecrypt` | 实现AES-256-GCM |
| **包装层** | `internal/crypto/crypto.go` | `EncryptMessage/DecryptMessage` | 使用频道密钥 |
| **应用层** | `internal/client/client.go`<br>`internal/server/server.go` | 调用crypto方法 | 发送前加密<br>接收后解密 |
| **传输层** | `internal/transport/arp_transport.go` | 传输Payload | **不做加密**<br>只传输已加密数据 |

### 核心设计原则

✅ **职责分离:** 应用层加密，传输层传输  
✅ **双重保护:** AES加密 + Ed25519签名  
✅ **统一接口:** 所有传输层接口一致  
✅ **高性能:** GCM认证加密，避免双重加密  

---

**文档版本:** 1.0  
**最后更新:** 2025-10-14  
**状态:** ✅ 当前实现正确且安全

