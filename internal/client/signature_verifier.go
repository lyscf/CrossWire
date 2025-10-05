package client

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/transport"
)

// SignatureVerifier 消息签名验证器
type SignatureVerifier struct {
	client *Client

	// 服务器公钥
	serverPublicKey ed25519.PublicKey
	keyMutex        sync.RWMutex

	// 已验证的消息缓存（防止重复验证）
	verifiedCache      map[string]time.Time
	verifiedCacheMutex sync.RWMutex
	maxCacheSize       int
	cacheExpiry        time.Duration

	// 统计
	stats      SignatureStats
	statsMutex sync.RWMutex
}

// SignatureStats 签名验证统计
type SignatureStats struct {
	TotalVerified  int64
	SuccessCount   int64
	FailureCount   int64
	CacheHits      int64
	LastVerifyTime time.Time
}

// NewSignatureVerifier 创建签名验证器
func NewSignatureVerifier(client *Client) *SignatureVerifier {
	return &SignatureVerifier{
		client:        client,
		verifiedCache: make(map[string]time.Time),
		maxCacheSize:  10000,
		cacheExpiry:   1 * time.Hour,
	}
}

// SetServerPublicKey 设置服务器公钥
func (sv *SignatureVerifier) SetServerPublicKey(publicKey ed25519.PublicKey) {
	sv.keyMutex.Lock()
	defer sv.keyMutex.Unlock()

	sv.serverPublicKey = publicKey
	sv.client.logger.Info("[SignatureVerifier] Server public key set")
}

// GetServerPublicKey 获取服务器公钥
func (sv *SignatureVerifier) GetServerPublicKey() ed25519.PublicKey {
	sv.keyMutex.RLock()
	defer sv.keyMutex.RUnlock()

	return sv.serverPublicKey
}

// VerifyMessage 验证消息签名
func (sv *SignatureVerifier) VerifyMessage(msg *transport.Message) (bool, error) {
	sv.statsMutex.Lock()
	sv.stats.TotalVerified++
	sv.stats.LastVerifyTime = time.Now()
	sv.statsMutex.Unlock()

	// 检查是否已验证（缓存命中）
	if sv.isVerified(msg.ID) {
		sv.statsMutex.Lock()
		sv.stats.CacheHits++
		sv.statsMutex.Unlock()
		return true, nil
	}

	// 检查是否有服务器公钥
	sv.keyMutex.RLock()
	if len(sv.serverPublicKey) == 0 {
		sv.keyMutex.RUnlock()
		sv.client.logger.Warn("[SignatureVerifier] No server public key, skipping verification")
		return true, nil // 如果没有公钥，跳过验证
	}
	publicKey := sv.serverPublicKey
	sv.keyMutex.RUnlock()

	// 解析签名消息
	var signedMsg SignedMessage
	if err := json.Unmarshal(msg.Payload, &signedMsg); err != nil {
		// 不是签名消息格式，可能是旧格式
		sv.client.logger.Debug("[SignatureVerifier] Message not in signed format: %s", msg.ID)
		return true, nil
	}

	// 验证签名
	valid := ed25519.Verify(publicKey, signedMsg.Message, signedMsg.Signature)

	if valid {
		// 缓存验证结果
		sv.addToCache(msg.ID)

		sv.statsMutex.Lock()
		sv.stats.SuccessCount++
		sv.statsMutex.Unlock()

		sv.client.logger.Debug("[SignatureVerifier] Message signature verified: %s", msg.ID)
	} else {
		sv.statsMutex.Lock()
		sv.stats.FailureCount++
		sv.statsMutex.Unlock()

		sv.client.logger.Warn("[SignatureVerifier] Message signature verification failed: %s", msg.ID)
	}

	return valid, nil
}

// VerifyMessagePayload 验证消息并返回原始负载
func (sv *SignatureVerifier) VerifyMessagePayload(msg *transport.Message) ([]byte, error) {
	// 验证签名
	valid, err := sv.VerifyMessage(msg)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("signature verification failed")
	}

	// 解析并返回原始消息
	var signedMsg SignedMessage
	if err := json.Unmarshal(msg.Payload, &signedMsg); err != nil {
		// 不是签名格式，直接返回原始负载
		return msg.Payload, nil
	}

	return signedMsg.Message, nil
}

// isVerified 检查消息是否已验证
func (sv *SignatureVerifier) isVerified(messageID string) bool {
	sv.verifiedCacheMutex.RLock()
	defer sv.verifiedCacheMutex.RUnlock()

	timestamp, exists := sv.verifiedCache[messageID]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Since(timestamp) > sv.cacheExpiry {
		return false
	}

	return true
}

// addToCache 添加到验证缓存
func (sv *SignatureVerifier) addToCache(messageID string) {
	sv.verifiedCacheMutex.Lock()
	defer sv.verifiedCacheMutex.Unlock()

	// 检查缓存大小
	if len(sv.verifiedCache) >= sv.maxCacheSize {
		// 清理过期条目
		sv.cleanupExpired()

		// 如果还是满的，清理最旧的条目
		if len(sv.verifiedCache) >= sv.maxCacheSize {
			sv.cleanupOldest()
		}
	}

	sv.verifiedCache[messageID] = time.Now()
}

// cleanupExpired 清理过期条目
func (sv *SignatureVerifier) cleanupExpired() {
	now := time.Now()
	for id, timestamp := range sv.verifiedCache {
		if now.Sub(timestamp) > sv.cacheExpiry {
			delete(sv.verifiedCache, id)
		}
	}
}

// cleanupOldest 清理最旧的条目
func (sv *SignatureVerifier) cleanupOldest() {
	if len(sv.verifiedCache) == 0 {
		return
	}

	// 找到最旧的条目
	var oldestID string
	var oldestTime time.Time
	first := true

	for id, timestamp := range sv.verifiedCache {
		if first || timestamp.Before(oldestTime) {
			oldestID = id
			oldestTime = timestamp
			first = false
		}
	}

	delete(sv.verifiedCache, oldestID)
}

// ClearCache 清空验证缓存
func (sv *SignatureVerifier) ClearCache() {
	sv.verifiedCacheMutex.Lock()
	defer sv.verifiedCacheMutex.Unlock()

	sv.verifiedCache = make(map[string]time.Time)
	sv.client.logger.Debug("[SignatureVerifier] Verification cache cleared")
}

// GetStats 获取统计信息
func (sv *SignatureVerifier) GetStats() SignatureStats {
	sv.statsMutex.RLock()
	defer sv.statsMutex.RUnlock()

	return SignatureStats{
		TotalVerified:  sv.stats.TotalVerified,
		SuccessCount:   sv.stats.SuccessCount,
		FailureCount:   sv.stats.FailureCount,
		CacheHits:      sv.stats.CacheHits,
		LastVerifyTime: sv.stats.LastVerifyTime,
	}
}

// GetCacheSize 获取缓存大小
func (sv *SignatureVerifier) GetCacheSize() int {
	sv.verifiedCacheMutex.RLock()
	defer sv.verifiedCacheMutex.RUnlock()

	return len(sv.verifiedCache)
}

// SetMaxCacheSize 设置最大缓存大小
func (sv *SignatureVerifier) SetMaxCacheSize(size int) {
	sv.maxCacheSize = size
	sv.client.logger.Debug("[SignatureVerifier] Max cache size set to: %d", size)
}

// SetCacheExpiry 设置缓存过期时间
func (sv *SignatureVerifier) SetCacheExpiry(expiry time.Duration) {
	sv.cacheExpiry = expiry
	sv.client.logger.Debug("[SignatureVerifier] Cache expiry set to: %v", expiry)
}
