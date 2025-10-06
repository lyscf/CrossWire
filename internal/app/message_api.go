package app

import (
	"crosswire/internal/models"
)

// ==================== 消息操作 API ====================

// SendMessage 发送文本消息
func (a *App) SendMessage(req SendMessageRequest) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 验证请求
	if req.Content == "" {
		return NewErrorResponse("invalid_request", "消息内容不能为空", "")
	}

	// 客户端发送（服务端接收广播），当前仅支持客户端直接发送
	var err error
	if mode == ModeClient && cli != nil {
		err = cli.SendMessage(req.Content, req.Type)
	} else if mode == ModeServer && a.server != nil {
		// 服务端暂不直接发送用户消息
		return NewErrorResponse("invalid_mode", "服务端不支持直接发送", "")
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("send_error", "消息发送失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "消息发送成功",
	})
}

// SendCodeMessage 发送代码消息
func (a *App) SendCodeMessage(req SendCodeRequest) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 验证请求
	if req.Code == "" {
		return NewErrorResponse("invalid_request", "代码内容不能为空", "")
	}

	// 简化：客户端以文本发送代码片段（后续可扩展消息格式）
	var err error
	if mode == ModeClient && cli != nil {
		err = cli.SendMessage(req.Code, models.MessageTypeCode)
	} else {
		return NewErrorResponse("invalid_mode", "仅客户端可发送消息", "")
	}

	if err != nil {
		return NewErrorResponse("send_error", "代码消息发送失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "代码消息发送成功",
	})
}

// GetMessages 获取消息列表
func (a *App) GetMessages(limit, offset int) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 获取当前频道ID
	var channelID string
	if a.mode == ModeServer && a.server != nil {
		ch, _ := a.server.GetChannel()
		channelID = ch.ID
	} else if a.mode == ModeClient && a.client != nil {
		channelID = a.client.GetChannelID()
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	// 从数据库获取消息
	messages, err := a.db.MessageRepo().GetByChannelID(channelID, limit, offset)
	if err != nil {
		return NewErrorResponse("db_error", "获取消息失败", err.Error())
	}

	// 转换为DTO
	messageDTOs := make([]*MessageDTO, 0, len(messages))
	for _, msg := range messages {
		dto := a.messageToDTO(msg)
		messageDTOs = append(messageDTOs, dto)
	}

	return NewSuccessResponse(messageDTOs)
}

// GetMessage 获取单条消息
func (a *App) GetMessage(messageID string) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 从数据库获取消息
	msg, err := a.db.MessageRepo().GetByID(messageID)
	if err != nil {
		return NewErrorResponse("not_found", "消息不存在", err.Error())
	}

	dto := a.messageToDTO(msg)
	return NewSuccessResponse(dto)
}

// SearchMessages 搜索消息
func (a *App) SearchMessages(req SearchMessagesRequest) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 获取当前频道ID
	var channelID string
	if a.mode == ModeServer && a.server != nil {
		ch, _ := a.server.GetChannel()
		channelID = ch.ID
	} else if a.mode == ModeClient && a.client != nil {
		channelID = a.client.GetChannelID()
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	// 搜索消息
	messages, err := a.db.MessageRepo().Search(channelID, req.Query, req.Limit, req.Offset)
	if err != nil {
		return NewErrorResponse("search_error", "搜索失败", err.Error())
	}

	// 转换为DTO
	messageDTOs := make([]*MessageDTO, 0, len(messages))
	for _, msg := range messages {
		dto := a.messageToDTO(msg)
		messageDTOs = append(messageDTOs, dto)
	}

	return NewSuccessResponse(messageDTOs)
}

// DeleteMessage 删除消息（仅服务端）
func (a *App) DeleteMessage(messageID string) Response {
	a.mu.RLock()
	mode := a.mode
	a.mu.RUnlock()

	if mode != ModeServer || a.server == nil {
		return NewErrorResponse("permission_denied", "仅服务端可删除消息", "")
	}

	// 使用仓库执行软删除
	if err := a.db.MessageRepo().Delete(messageID, "server"); err != nil {
		return NewErrorResponse("delete_error", "删除消息失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "消息已删除",
	})
}

// PinMessage 置顶消息（仅服务端）
func (a *App) PinMessage(messageID string) Response {
	a.mu.RLock()
	mode := a.mode
	a.mu.RUnlock()

	if mode != ModeServer || a.server == nil {
		return NewErrorResponse("permission_denied", "仅服务端可置顶消息", "")
	}

	ch, _ := a.server.GetChannel()
	if err := a.db.ChannelRepo().PinMessage(ch.ID, messageID, "server", ""); err != nil {
		return NewErrorResponse("pin_error", "置顶消息失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "消息已置顶",
	})
}

// UnpinMessage 取消置顶消息（仅服务端）
func (a *App) UnpinMessage(messageID string) Response {
	a.mu.RLock()
	mode := a.mode
	a.mu.RUnlock()

	if mode != ModeServer || a.server == nil {
		return NewErrorResponse("permission_denied", "仅服务端可取消置顶", "")
	}

	ch, _ := a.server.GetChannel()
	if err := a.db.ChannelRepo().UnpinMessage(ch.ID, messageID); err != nil {
		return NewErrorResponse("unpin_error", "取消置顶失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "已取消置顶",
	})
}

// ReactToMessage 对消息添加反应
func (a *App) ReactToMessage(messageID, emoji string) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 验证emoji
	if emoji == "" {
		return NewErrorResponse("invalid_request", "emoji不能为空", "")
	}

	// 添加反应
	var err error
	if mode == ModeClient && cli != nil {
		// TODO: 客户端本地记录/发送反应
		err = nil
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("reaction_error", "添加反应失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "反应已添加",
	})
}

// RemoveReaction 移除消息反应
func (a *App) RemoveReaction(messageID, emoji string) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 移除反应
	var err error
	if mode == ModeClient && cli != nil {
		// TODO: 客户端本地记录/发送反应移除
		err = nil
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("reaction_error", "移除反应失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "反应已移除",
	})
}

// ==================== 辅助方法 ====================

// messageToDTO 转换消息模型为DTO
func (a *App) messageToDTO(msg *models.Message) *MessageDTO {
	// 获取发送者信息
	senderName := "Unknown"
	if msg.SenderID != "" {
		member, err := a.db.MemberRepo().GetByID(msg.SenderID)
		if err == nil && member != nil {
			senderName = member.Nickname
		}
	}

	// 转换reactions（简化版）
	reactions := make([]MessageReaction, 0)
	// TODO: 从数据库加载reactions

	return &MessageDTO{
		ID:         msg.ID,
		ChannelID:  msg.ChannelID,
		SenderID:   msg.SenderID,
		SenderName: senderName,
		Type:       msg.Type,
		Content:    msg.Content,
		Timestamp:  msg.Timestamp,
		IsEdited:   msg.IsEdited,
		IsDeleted:  msg.IsDeleted,
		IsPinned:   msg.IsPinned,
		ReplyToID:  msg.ReplyToID,
		Reactions:  reactions,
	}
}
