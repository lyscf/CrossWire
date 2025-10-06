package server

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"crosswire/internal/models"
)

// SpamDetector 反垃圾消息检测器
// 参考: docs/FEATURES.md - 反垃圾消息功能
type SpamDetector struct {
	server *Server

	// 消息指纹（用于检测重复消息）
	messageFingerprints map[string]time.Time // hash -> timestamp
	fingerprintMutex    sync.RWMutex

	// 成员消息历史（用于检测连续相同消息）
	memberHistory      map[string][]string // memberID -> message hashes
	memberHistoryMutex sync.RWMutex

	// 黑名单关键词
	blacklistWords []string
	blacklistMutex sync.RWMutex

	// 配置
	config SpamDetectorConfig

	// 统计
	stats SpamDetectorStats
}

// SpamDetectorConfig 反垃圾配置
type SpamDetectorConfig struct {
	EnableDuplicateDetection bool          // 是否检测重复消息
	EnableContentFilter      bool          // 是否启用内容过滤
	EnableRapidPostDetection bool          // 是否检测快速连发
	MaxDuplicateWindow       time.Duration // 重复消息检测时间窗口
	MaxSimilarInHistory      int           // 历史记录中允许的最大相似消息数
	RapidPostThreshold       time.Duration // 快速连发阈值
	MaxHistorySize           int           // 每个成员的历史记录大小
}

// SpamDetectorStats 反垃圾统计
type SpamDetectorStats struct {
	TotalChecked       uint64 // 总检查数
	DuplicateDetected  uint64 // 重复消息数
	BlacklistDetected  uint64 // 黑名单检测数
	RapidPostDetected  uint64 // 快速连发检测数
	SimilarityDetected uint64 // 相似消息检测数
	mutex              sync.RWMutex
}

// DefaultSpamDetectorConfig 默认配置
var DefaultSpamDetectorConfig = SpamDetectorConfig{
	EnableDuplicateDetection: true,
	EnableContentFilter:      true,
	EnableRapidPostDetection: true,
	MaxDuplicateWindow:       5 * time.Minute,
	MaxSimilarInHistory:      3,
	RapidPostThreshold:       2 * time.Second,
	MaxHistorySize:           20,
}

// NewSpamDetector 创建反垃圾检测器
func NewSpamDetector(server *Server) *SpamDetector {
	sd := &SpamDetector{
		server:              server,
		messageFingerprints: make(map[string]time.Time),
		memberHistory:       make(map[string][]string),
		config:              DefaultSpamDetectorConfig,
		blacklistWords: []string{
			// 默认黑名单（可根据需要扩展）
			"spam", "advertisement", "广告",
		},
	}

	// 启动定期清理
	go sd.cleanupWorker()

	return sd
}

// CheckMessage 检查消息是否为垃圾消息
// 返回 (isSpam, reason)
func (sd *SpamDetector) CheckMessage(msg *models.Message, memberID string) (bool, string) {
	sd.stats.mutex.Lock()
	sd.stats.TotalChecked++
	sd.stats.mutex.Unlock()

	// 1. 检测内容过滤（黑名单）
	if sd.config.EnableContentFilter {
		if isBlacklisted, reason := sd.checkBlacklist(msg); isBlacklisted {
			sd.stats.mutex.Lock()
			sd.stats.BlacklistDetected++
			sd.stats.mutex.Unlock()
			return true, reason
		}
	}

	// 2. 计算消息指纹
	fingerprint := sd.calculateFingerprint(msg)

	// 3. 检测重复消息
	if sd.config.EnableDuplicateDetection {
		if isDuplicate := sd.checkDuplicate(fingerprint); isDuplicate {
			sd.stats.mutex.Lock()
			sd.stats.DuplicateDetected++
			sd.stats.mutex.Unlock()
			return true, "Duplicate message detected"
		}
	}

	// 4. 检测成员快速连发相似消息
	if sd.config.EnableRapidPostDetection {
		if isRapid, reason := sd.checkMemberHistory(memberID, fingerprint); isRapid {
			sd.stats.mutex.Lock()
			sd.stats.RapidPostDetected++
			sd.stats.mutex.Unlock()
			return true, reason
		}
	}

	// 5. 记录消息指纹和历史
	sd.recordMessage(memberID, fingerprint)

	return false, ""
}

// checkBlacklist 检查黑名单关键词
func (sd *SpamDetector) checkBlacklist(msg *models.Message) (bool, string) {
	sd.blacklistMutex.RLock()
	defer sd.blacklistMutex.RUnlock()

	// 获取消息文本内容
	content := ""
	if msg.Type == models.MessageTypeText {
		if text, ok := msg.Content["text"].(string); ok {
			content = strings.ToLower(text)
		}
	}

	// 检查黑名单关键词
	for _, word := range sd.blacklistWords {
		if strings.Contains(content, strings.ToLower(word)) {
			return true, "Blacklisted keyword detected: " + word
		}
	}

	return false, ""
}

// checkDuplicate 检查是否为重复消息
func (sd *SpamDetector) checkDuplicate(fingerprint string) bool {
	sd.fingerprintMutex.RLock()
	defer sd.fingerprintMutex.RUnlock()

	if timestamp, exists := sd.messageFingerprints[fingerprint]; exists {
		// 检查是否在时间窗口内
		if time.Since(timestamp) < sd.config.MaxDuplicateWindow {
			return true
		}
	}

	return false
}

// checkMemberHistory 检查成员历史消息
func (sd *SpamDetector) checkMemberHistory(memberID, fingerprint string) (bool, string) {
	sd.memberHistoryMutex.RLock()
	history := sd.memberHistory[memberID]
	sd.memberHistoryMutex.RUnlock()

	if len(history) == 0 {
		return false, ""
	}

	// 检查是否快速连发相同消息
	similarCount := 0
	for _, oldHash := range history {
		if oldHash == fingerprint {
			similarCount++
		}
	}

	if similarCount >= sd.config.MaxSimilarInHistory {
		return true, "Too many similar messages in history"
	}

	return false, ""
}

// recordMessage 记录消息
func (sd *SpamDetector) recordMessage(memberID, fingerprint string) {
	now := time.Now()

	// 记录全局指纹
	sd.fingerprintMutex.Lock()
	sd.messageFingerprints[fingerprint] = now
	sd.fingerprintMutex.Unlock()

	// 记录成员历史
	sd.memberHistoryMutex.Lock()
	history := sd.memberHistory[memberID]
	history = append(history, fingerprint)

	// 限制历史大小
	if len(history) > sd.config.MaxHistorySize {
		history = history[len(history)-sd.config.MaxHistorySize:]
	}

	sd.memberHistory[memberID] = history
	sd.memberHistoryMutex.Unlock()
}

// calculateFingerprint 计算消息指纹
func (sd *SpamDetector) calculateFingerprint(msg *models.Message) string {
	// 使用发送者ID + 内容 计算SHA256
	data := msg.SenderID + ":"

	if msg.Type == models.MessageTypeText {
		if text, ok := msg.Content["text"].(string); ok {
			data += text
		}
	} else if msg.Type == models.MessageTypeCode {
		if code, ok := msg.Content["code"].(string); ok {
			data += code
		}
	}

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// cleanupWorker 定期清理过期数据
func (sd *SpamDetector) cleanupWorker() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sd.server.ctx.Done():
			return
		case <-ticker.C:
			sd.cleanup()
		}
	}
}

// cleanup 清理过期数据
func (sd *SpamDetector) cleanup() {
	// 清理过期的消息指纹
	sd.fingerprintMutex.Lock()
	now := time.Now()
	expiry := now.Add(-sd.config.MaxDuplicateWindow)
	for hash, timestamp := range sd.messageFingerprints {
		if timestamp.Before(expiry) {
			delete(sd.messageFingerprints, hash)
		}
	}
	sd.fingerprintMutex.Unlock()

	sd.server.logger.Debug("[SpamDetector] Cleanup completed, remaining fingerprints: %d",
		len(sd.messageFingerprints))
}

// AddBlacklistWord 添加黑名单关键词
func (sd *SpamDetector) AddBlacklistWord(word string) {
	sd.blacklistMutex.Lock()
	defer sd.blacklistMutex.Unlock()

	sd.blacklistWords = append(sd.blacklistWords, word)
	sd.server.logger.Info("[SpamDetector] Added blacklist word: %s", word)
}

// RemoveBlacklistWord 移除黑名单关键词
func (sd *SpamDetector) RemoveBlacklistWord(word string) {
	sd.blacklistMutex.Lock()
	defer sd.blacklistMutex.Unlock()

	for i, w := range sd.blacklistWords {
		if w == word {
			sd.blacklistWords = append(sd.blacklistWords[:i], sd.blacklistWords[i+1:]...)
			sd.server.logger.Info("[SpamDetector] Removed blacklist word: %s", word)
			break
		}
	}
}

// GetBlacklistWords 获取黑名单关键词列表
func (sd *SpamDetector) GetBlacklistWords() []string {
	sd.blacklistMutex.RLock()
	defer sd.blacklistMutex.RUnlock()

	words := make([]string, len(sd.blacklistWords))
	copy(words, sd.blacklistWords)
	return words
}

// ClearMemberHistory 清除指定成员的历史记录
func (sd *SpamDetector) ClearMemberHistory(memberID string) {
	sd.memberHistoryMutex.Lock()
	defer sd.memberHistoryMutex.Unlock()

	delete(sd.memberHistory, memberID)
	sd.server.logger.Debug("[SpamDetector] Cleared history for member: %s", memberID)
}

// GetStats 获取统计信息
func (sd *SpamDetector) GetStats() SpamDetectorStats {
	sd.stats.mutex.RLock()
	defer sd.stats.mutex.RUnlock()

	return SpamDetectorStats{
		TotalChecked:       sd.stats.TotalChecked,
		DuplicateDetected:  sd.stats.DuplicateDetected,
		BlacklistDetected:  sd.stats.BlacklistDetected,
		RapidPostDetected:  sd.stats.RapidPostDetected,
		SimilarityDetected: sd.stats.SimilarityDetected,
	}
}

// ResetStats 重置统计信息
func (sd *SpamDetector) ResetStats() {
	sd.stats.mutex.Lock()
	defer sd.stats.mutex.Unlock()

	sd.stats.TotalChecked = 0
	sd.stats.DuplicateDetected = 0
	sd.stats.BlacklistDetected = 0
	sd.stats.RapidPostDetected = 0
	sd.stats.SimilarityDetected = 0
}
