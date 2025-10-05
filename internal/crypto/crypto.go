package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/curve25519"
)

// Manager 加密管理器
type Manager struct {
	channelKey []byte // 频道密钥（用于消息加密）
}

// NewManager 创建加密管理器
func NewManager() (*Manager, error) {
	return &Manager{}, nil
}

// ===== AES 加密 =====

// AESEncrypt 使用AES-256-GCM加密数据
func (m *Manager) AESEncrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// nonce + ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// AESDecrypt 使用AES-256-GCM解密数据
func (m *Manager) AESDecrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// ===== X25519 密钥交换 =====

// GenerateX25519KeyPair 生成X25519密钥对
func (m *Manager) GenerateX25519KeyPair() (privateKey, publicKey []byte, err error) {
	privateKey = make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, nil, err
	}

	publicKey, err = curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, publicKey, nil
}

// X25519SharedSecret 计算共享密钥
func (m *Manager) X25519SharedSecret(privateKey, peerPublicKey []byte) ([]byte, error) {
	if len(privateKey) != curve25519.ScalarSize {
		return nil, fmt.Errorf("invalid private key size")
	}
	if len(peerPublicKey) != curve25519.PointSize {
		return nil, fmt.Errorf("invalid public key size")
	}

	sharedSecret, err := curve25519.X25519(privateKey, peerPublicKey)
	if err != nil {
		return nil, err
	}

	// 使用SHA256派生最终密钥
	hash := sha256.Sum256(sharedSecret)
	return hash[:], nil
}

// ===== Ed25519 签名 =====

// GenerateEd25519KeyPair 生成Ed25519密钥对（用于签名）
func (m *Manager) GenerateEd25519KeyPair() (privateKey, publicKey []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

// Ed25519Sign 使用Ed25519签名
func (m *Manager) Ed25519Sign(privateKey, message []byte) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}

	signature := ed25519.Sign(privateKey, message)
	return signature, nil
}

// Ed25519Verify 验证Ed25519签名
func (m *Manager) Ed25519Verify(publicKey, message, signature []byte) bool {
	if len(publicKey) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(publicKey, message, signature)
}

// ===== 密钥派生 =====

// DeriveKey 从密码派生密钥（带错误返回）
func (m *Manager) DeriveKey(password string, salt []byte) ([]byte, error) {
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}
	key := m.DeriveKeyFromPassword(password, salt)
	return key, nil
}

// DeriveKeyFromPassword 从密码派生密钥（使用Argon2id）
func (m *Manager) DeriveKeyFromPassword(password string, salt []byte) []byte {
	// Argon2id 参数
	// time=1, memory=64MB, threads=4, keyLen=32
	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return key
}

// GenerateSalt 生成随机盐值
func (m *Manager) GenerateSalt() ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// ===== 哈希函数 =====

// SHA256Hash 计算SHA256哈希
func (m *Manager) SHA256Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// SHA256HashHex 计算SHA256哈希并返回hex字符串
func (m *Manager) SHA256HashHex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ===== 密码哈希 =====

// HashPassword 哈希密码（使用Argon2id）
func (m *Manager) HashPassword(password string, salt []byte) string {
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return hex.EncodeToString(hash)
}

// VerifyPassword 验证密码
func (m *Manager) VerifyPassword(password, hashHex string, salt []byte) bool {
	expectedHash, err := hex.DecodeString(hashHex)
	if err != nil {
		return false
	}

	actualHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// 使用恒定时间比较防止时序攻击
	if len(actualHash) != len(expectedHash) {
		return false
	}

	var v byte
	for i := 0; i < len(actualHash); i++ {
		v |= actualHash[i] ^ expectedHash[i]
	}

	return v == 0
}

// ===== 随机数生成 =====

// GenerateRandomBytes 生成随机字节
func (m *Manager) GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateRandomHex 生成随机hex字符串
func (m *Manager) GenerateRandomHex(n int) (string, error) {
	b, err := m.GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ===== 频道密钥管理 =====

// SetChannelKey 设置频道密钥
func (m *Manager) SetChannelKey(key []byte) {
	m.channelKey = key
}

// GetChannelKey 获取频道密钥
func (m *Manager) GetChannelKey() []byte {
	return m.channelKey
}

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

// TODO: 实现以下功能
// - 密钥轮换
// - 密钥缓存管理
// - 文件加密/解密（分块）
// - HMAC签名
// - 更多密钥派生函数
