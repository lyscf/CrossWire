package server

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/models"
	"crosswire/internal/transport"
)

// AuthManager 认证管理器
// 参考: docs/ARCHITECTURE.md - 3.1.2 服务端模块 - AuthManager
// 参考: docs/PROTOCOL.md - 2.2.2 认证握手
type AuthManager struct {
	server *Server

	// 会话管理
	sessions      map[string]*Session // memberID -> Session
	sessionsMutex sync.RWMutex

	// TODO: 认证挑战功能（高级安全特性，待实现）
	// challenges      map[string]*AuthChallenge // challengeID -> Challenge
	// challengesMutex sync.RWMutex
}

// Session 会话
type Session struct {
	MemberID   string
	PublicKey  []byte
	CreatedAt  time.Time
	LastSeen   time.Time
	ExpiresAt  time.Time
	IsVerified bool
}

// AuthChallenge 认证挑战
type AuthChallenge struct {
	ID        string
	MemberID  string
	Challenge []byte
	CreatedAt time.Time
	ExpiresAt time.Time
}

// JoinRequest 加入请求
// 参考: docs/PROTOCOL.md - 2.2.2 认证握手
type JoinRequest struct {
	Nickname  string `json:"nickname"`
	PublicKey []byte `json:"public_key"` // X25519公钥
	Timestamp int64  `json:"timestamp"`
	Signature []byte `json:"signature,omitempty"` // 可选的签名
}

// JoinResponse 加入响应
type JoinResponse struct {
	Success         bool          `json:"success"`
	Message         string        `json:"message,omitempty"`
	ChannelKey      []byte        `json:"channel_key,omitempty"`
	ChannelID       string        `json:"channel_id,omitempty"`
	MemberID        string        `json:"member_id,omitempty"`
	MemberList      []*MemberInfo `json:"member_list,omitempty"`
	ServerPublicKey []byte        `json:"server_public_key,omitempty"`
	Timestamp       int64         `json:"timestamp"`
}

// MemberInfo 成员信息（精简版）
type MemberInfo struct {
	ID       string            `json:"id"`
	Nickname string            `json:"nickname"`
	Role     models.Role       `json:"role"`
	Status   models.UserStatus `json:"status"`
}

// NewAuthManager 创建认证管理器
func NewAuthManager(server *Server) *AuthManager {
	am := &AuthManager{
		server:   server,
		sessions: make(map[string]*Session),
	}

	// 启动会话清理协程
	go am.cleanupExpiredSessions()

	return am
}

// HandleJoinRequest 处理加入请求
// 参考: docs/PROTOCOL.md - 2.2.2 认证握手
func (am *AuthManager) HandleJoinRequest(transportMsg *transport.Message) {
	am.server.logger.Debug("[AuthManager] Join request from: %s addr=%s len=%d", transportMsg.SenderID, transportMsg.SenderAddr, len(transportMsg.Payload))

	// 1. 解密请求（使用密码派生的密钥）
	decrypted, err := am.server.crypto.DecryptMessage(transportMsg.Payload)
	if err != nil {
		// 打印更多上下文用于排查：密文前缀与当前频道密钥指纹
		previewLen := 16
		if l := len(transportMsg.Payload); l < previewLen {
			previewLen = l
		}
		cipherHead := fmt.Sprintf("%x", transportMsg.Payload[:previewLen])
		ck := am.server.crypto.GetChannelKey()
		ckHash := sha256.Sum256(ck)
		am.server.logger.Error("[AuthManager] Failed to decrypt join request: %v | addr=%s cipher_len=%d cipher_head=%s channel_id=%s key_fp=%x",
			err, transportMsg.SenderAddr, len(transportMsg.Payload), cipherHead, am.server.config.ChannelID, ckHash[:4])
		am.sendJoinResponse(transportMsg.SenderID, false, "Invalid password or encryption", nil)
		return
	}

	// 2. 反序列化请求
	var joinReq JoinRequest
	if err := json.Unmarshal(decrypted, &joinReq); err != nil {
		am.server.logger.Error("[AuthManager] Failed to unmarshal join request: %v", err)
		am.sendJoinResponse(transportMsg.SenderID, false, "Invalid request format", nil)
		return
	}

	// 3. 验证时间戳（防重放攻击）
	now := time.Now().Unix()
	if now-joinReq.Timestamp > 300 || joinReq.Timestamp > now+60 {
		am.server.logger.Warn("[AuthManager] Invalid timestamp in join request: %d", joinReq.Timestamp)
		am.sendJoinResponse(transportMsg.SenderID, false, "Invalid timestamp", nil)
		return
	}

	// 4. 验证昵称
	if joinReq.Nickname == "" || len(joinReq.Nickname) > 50 {
		am.server.logger.Warn("[AuthManager] Invalid nickname: %s", joinReq.Nickname)
		am.sendJoinResponse(transportMsg.SenderID, false, "Invalid nickname", nil)
		return
	}

	// 5. 检查频道是否已满
	if am.server.channelManager.GetTotalCount() >= am.server.config.MaxMembers {
		am.server.logger.Warn("[AuthManager] Channel is full")
		am.sendJoinResponse(transportMsg.SenderID, false, "Channel is full", nil)
		return
	}

	// 6. 创建新成员
	member := &models.Member{
		ID:         generateMemberID(),
		ChannelID:  am.server.config.ChannelID,
		Nickname:   joinReq.Nickname,
		PublicKey:  joinReq.PublicKey,
		Role:       models.RoleMember,
		Status:     models.StatusOnline,
		JoinedAt:   time.Now(),
		LastSeenAt: time.Now(),
	}

	// 7. 添加成员到频道
	if err := am.server.channelManager.AddMember(member); err != nil {
		am.server.logger.Error("[AuthManager] Failed to add member: %v", err)
		am.sendJoinResponse(transportMsg.SenderID, false, fmt.Sprintf("Failed to join: %v", err), nil)
		return
	}

	// 8. 创建会话
	session := &Session{
		MemberID:   member.ID,
		PublicKey:  joinReq.PublicKey,
		CreatedAt:  time.Now(),
		LastSeen:   time.Now(),
		ExpiresAt:  time.Now().Add(am.server.config.SessionTimeout),
		IsVerified: true,
	}

	am.sessionsMutex.Lock()
	am.sessions[member.ID] = session
	am.sessionsMutex.Unlock()

	// 9. 获取成员列表（包含刚加入的成员）
	members, err := am.server.channelManager.GetMembers()
	if err != nil {
		am.server.logger.Error("[AuthManager] Failed to get members: %v", err)
		members = []*models.Member{}
	}

	memberList := make([]*MemberInfo, 0, len(members))
	for _, m := range members {
		memberList = append(memberList, &MemberInfo{
			ID:       m.ID,
			Nickname: m.Nickname,
			Role:     m.Role,
			Status:   m.Status,
		})
	}

	// 10. 构造响应
	response := &JoinResponse{
		Success:         true,
		Message:         "",
		ChannelKey:      am.server.crypto.GetChannelKey(),
		ChannelID:       am.server.config.ChannelID,
		MemberID:        member.ID,
		MemberList:      memberList,
		ServerPublicKey: am.server.config.PublicKey,
		Timestamp:       time.Now().Unix(),
	}

	// 11. 发送响应
	am.sendJoinResponse(transportMsg.SenderID, true, "", response)

	am.server.logger.Info("[AuthManager] Member joined: %s (%s)", member.Nickname, member.ID)

	// 12. 广播成员加入消息
	am.broadcastMemberJoined(member)
}

// sendJoinResponse 发送加入响应
func (am *AuthManager) sendJoinResponse(to string, success bool, errorMsg string, response *JoinResponse) {
	// 为了兼容客户端，响应格式统一为：
	// {
	//   "type": "auth.join_response",
	//   "success": true/false,
	//   "error": "...",           // 失败时
	//   "member": {"id":"...","nickname":"..."},
	//   "channel_id": "...",
	//   "timestamp": 169...
	// }

	resp := map[string]interface{}{
		"type":       "auth.join_response",
		"success":    success,
		"channel_id": am.server.config.ChannelID,
		"timestamp":  time.Now().Unix(),
	}
	if !success {
		resp["error"] = errorMsg
	}

	if response != nil && success {
		// 使用传入的成员信息
		memberID := response.MemberID
		nickname := ""
		if len(response.MemberList) > 0 {
			nickname = response.MemberList[0].Nickname
		}
		resp["member"] = map[string]interface{}{
			"id":       memberID,
			"nickname": nickname,
		}
		// 附带服务器公钥（用于 ARP 模式验签）。JSON 将 []byte 编码为 base64 字符串
		if len(response.ServerPublicKey) > 0 {
			resp["server_public_key"] = response.ServerPublicKey
		}
		// 附带成员列表，便于客户端初始本地化
		list := make([]map[string]interface{}, 0, len(response.MemberList))
		for _, mi := range response.MemberList {
			list = append(list, map[string]interface{}{
				"id":       mi.ID,
				"nickname": mi.Nickname,
				"role":     string(mi.Role),
				"status":   string(mi.Status),
			})
		}
		resp["member_list"] = list
	}

	// 序列化并加密
	bytes, err := json.Marshal(resp)
	if err != nil {
		am.server.logger.Error("[AuthManager] Failed to marshal join response: %v", err)
		return
	}
	encrypted, err := am.server.crypto.EncryptMessage(bytes)
	if err != nil {
		am.server.logger.Error("[AuthManager] Failed to encrypt join response: %v", err)
		return
	}

	// 发送响应（单播给新成员）：设置 SenderID 为该成员ID，便于客户端识别
	transportMsg := &transport.Message{
		Type:      transport.MessageTypeAuth,
		SenderID:  response.MemberID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}
	if err := am.server.transport.SendMessage(transportMsg); err != nil {
		am.server.logger.Error("[AuthManager] Failed to send join response: %v", err)
	}
}

// broadcastMemberJoined 广播成员加入
func (am *AuthManager) broadcastMemberJoined(member *models.Member) {
	// 创建系统消息（使用结构化 SystemContent）
	content := models.MessageContent{
		"event":    "member_joined",
		"actor_id": member.ID,
		"extra": map[string]interface{}{
			"nickname":  member.Nickname,
			"joined_at": time.Now().Unix(),
		},
	}
	systemMsg := &models.Message{
		ID:        generateMessageID(),
		ChannelID: am.server.config.ChannelID,
		SenderID:  "system",
		Type:      models.MessageTypeSystem,
		Content:   content,
		Timestamp: time.Now(),
	}

	// 广播
	if err := am.server.broadcastManager.Broadcast(systemMsg); err != nil {
		am.server.logger.Error("[AuthManager] Failed to broadcast member joined: %v", err)
	}
}

// VerifySession 验证会话
func (am *AuthManager) VerifySession(memberID string) bool {
	am.sessionsMutex.RLock()
	session, exists := am.sessions[memberID]
	am.sessionsMutex.RUnlock()

	if !exists {
		return false
	}

	// 检查是否过期
	if time.Now().After(session.ExpiresAt) {
		am.sessionsMutex.Lock()
		delete(am.sessions, memberID)
		am.sessionsMutex.Unlock()
		return false
	}

	// 更新最后访问时间
	am.sessionsMutex.Lock()
	session.LastSeen = time.Now()
	am.sessionsMutex.Unlock()

	return session.IsVerified
}

// GetSession 获取会话
func (am *AuthManager) GetSession(memberID string) (*Session, error) {
	am.sessionsMutex.RLock()
	defer am.sessionsMutex.RUnlock()

	session, exists := am.sessions[memberID]
	if !exists {
		return nil, errors.New("session not found")
	}

	return session, nil
}

// RemoveSession 移除会话
func (am *AuthManager) RemoveSession(memberID string) {
	am.sessionsMutex.Lock()
	defer am.sessionsMutex.Unlock()

	delete(am.sessions, memberID)
}

// cleanupExpiredSessions 清理过期会话
func (am *AuthManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-am.server.ctx.Done():
			return
		case <-ticker.C:
			am.sessionsMutex.Lock()
			now := time.Now()
			for memberID, session := range am.sessions {
				if now.After(session.ExpiresAt) {
					delete(am.sessions, memberID)
					am.server.logger.Debug("[AuthManager] Session expired: %s", memberID)
				}
			}
			am.sessionsMutex.Unlock()
		}
	}
}

// CheckPermission 检查权限
func (am *AuthManager) CheckPermission(memberID string, requiredRole models.Role) bool {
	member := am.server.channelManager.GetMemberByID(memberID)
	if member == nil {
		return false
	}

	// 管理员拥有所有权限
	if member.Role == models.RoleAdmin || member.Role == models.RoleOwner {
		return true
	}

	// 检查角色
	return member.Role == requiredRole
}

// generateMemberID 生成成员ID
func generateMemberID() string {
	return fmt.Sprintf("member_%d", time.Now().UnixNano())
}

// generateMessageID 生成消息ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
