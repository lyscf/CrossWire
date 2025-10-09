package app

import (
	"crosswire/internal/models"
)

// ==================== 消息操作 API ====================

// SendMessage 发送文本消息
func (a *App) SendMessage(req SendMessageRequest) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 验证请求
	if req.Content == "" {
		return NewErrorResponse("invalid_request", "消息内容不能为空", "")
	}

	// 确定目标频道ID（如果传入则优先使用）
	targetChannelID := ""
	if req.ChannelID != nil && *req.ChannelID != "" {
		targetChannelID = *req.ChannelID
	} else {
		if mode == ModeServer && srv != nil {
			ch, _ := srv.GetChannel()
			if ch != nil {
				targetChannelID = ch.ID
			}
		} else if mode == ModeClient && cli != nil {
			targetChannelID = cli.GetChannelID()
		}
	}

	// 客户端发送（服务端接收广播）
	var err error
	if mode == ModeClient && cli != nil {
		if targetChannelID == "" {
			a.logger.Warn("[App] targetChannelID empty in client mode, defaulting to client channel")
			err = cli.SendMessage(req.Content, req.Type)
		} else {
			err = cli.SendMessageToChannel(req.Content, req.Type, targetChannelID)
		}
	} else if mode == ModeServer && srv != nil {
		// 允许服务端直接发送
		var replyTo *string
		if req.ReplyToID != nil && *req.ReplyToID != "" {
			replyTo = req.ReplyToID
		}
		// 始终使用服务端当前频道，避免外键失败
		if ch, _ := srv.GetChannel(); ch != nil {
			targetChannelID = ch.ID
		}
		a.logger.Debug("[App] Server SendMessage type=%s channel=%s replyTo=%v", string(req.Type), targetChannelID, replyTo)
		_, err = srv.SendUserMessage(req.Content, req.Type, targetChannelID, replyTo)
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		a.logger.Error("[App] SendMessage failed: %v (mode=%s, channel=%s)", err, mode, targetChannelID)
		return NewErrorResponse("send_error", "消息发送失败", err.Error())
	}

	a.logger.Info("[App] Message sent (mode=%s, channel=%s)", mode, targetChannelID)
	return NewSuccessResponse(map[string]interface{}{
		"message": "消息发送成功",
	})
}

// SendCodeMessage 发送代码消息
func (a *App) SendCodeMessage(req SendCodeRequest) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 验证请求
	if req.Code == "" {
		return NewErrorResponse("invalid_request", "代码内容不能为空", "")
	}

	var err error
	if mode == ModeClient && cli != nil {
		err = cli.SendMessage(req.Code, models.MessageTypeCode)
	} else if mode == ModeServer && srv != nil {
		// 服务端也可发送代码消息，作为文本内容
		a.logger.Debug("[App] Server SendCodeMessage")
		_, err = srv.SendUserMessage(req.Code, models.MessageTypeCode, "", nil)
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		a.logger.Error("[App] SendCodeMessage failed: %v (mode=%s)", err, mode)
		return NewErrorResponse("send_error", "代码消息发送失败", err.Error())
	}

	a.logger.Info("[App] Code message sent (mode=%s)", mode)
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

// GetMessagesByChannel 获取指定频道的消息列表
func (a *App) GetMessagesByChannel(channelID string, limit, offset int) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	if channelID == "" {
		return NewErrorResponse("invalid_request", "channel_id 不能为空", "")
	}

	messages, err := a.db.MessageRepo().GetByChannelID(channelID, limit, offset)
	if err != nil {
		return NewErrorResponse("db_error", "获取消息失败", err.Error())
	}

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
func (a *App) PinMessage(req PinMessageRequest) Response {
	a.mu.RLock()
	mode := a.mode
	a.mu.RUnlock()

	if mode != ModeServer || a.server == nil {
		return NewErrorResponse("permission_denied", "仅服务端可置顶消息", "")
	}

	if req.MessageID == "" {
		return NewErrorResponse("invalid_request", "message_id 不能为空", "")
	}

	ch, _ := a.server.GetChannel()
	if ch == nil {
		return NewErrorResponse("no_channel", "未初始化频道", "")
	}
	if err := a.db.ChannelRepo().PinMessage(ch.ID, req.MessageID, "server", req.Reason); err != nil {
		return NewErrorResponse("pin_error", "置顶消息失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message":    "消息已置顶",
		"message_id": req.MessageID,
		"reason":     req.Reason,
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
		"message":    "已取消置顶",
		"message_id": messageID,
	})
}

// GetPinnedMessages 获取置顶消息列表（当前频道）
func (a *App) GetPinnedMessages() Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	var channelID string
	if mode == ModeServer && srv != nil {
		ch, _ := srv.GetChannel()
		if ch == nil {
			return NewErrorResponse("no_channel", "未初始化频道", "")
		}
		channelID = ch.ID
	} else if mode == ModeClient && cli != nil {
		channelID = cli.GetChannelID()
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	// 获取带内容的置顶消息
	rows, err := a.db.ChannelRepo().GetPinnedMessagesWithContent(channelID)
	if err != nil {
		return NewErrorResponse("query_error", "查询置顶消息失败", err.Error())
	}

	// 转换为 DTO
	dtos := make([]*PinnedMessageDTO, 0, len(rows))
	for _, item := range rows {
		dto := &PinnedMessageDTO{
			ID:             item.ID,
			ChannelID:      item.ChannelID,
			MessageID:      item.MessageID,
			PinnedBy:       item.PinnedBy,
			Reason:         item.Reason,
			PinnedAt:       item.PinnedAt.Unix(),
			DisplayOrder:   item.DisplayOrder,
			ContentText:    item.ContentText,
			SenderID:       item.SenderID,
			SenderNickname: item.SenderNickname,
		}
		dtos = append(dtos, dto)
	}

	return NewSuccessResponse(dtos)
}

// SetTypingStatus 设置当前用户的正在输入状态（5秒窗口）
func (a *App) SetTypingStatus() Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	var channelID string
	var userID string

	if mode == ModeServer && srv != nil {
		ch, _ := srv.GetChannel()
		if ch == nil {
			return NewErrorResponse("no_channel", "未初始化频道", "")
		}
		channelID = ch.ID
		userID = "server"
	} else if mode == ModeClient && cli != nil {
		channelID = cli.GetChannelID()
		userID = cli.GetMemberID()
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if channelID == "" || userID == "" {
		return NewErrorResponse("invalid_request", "缺少必要的身份信息", "")
	}

	if err := a.db.MessageRepo().SetTypingStatus(channelID, userID); err != nil {
		return NewErrorResponse("db_error", "设置输入状态失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "输入状态已更新",
	})
}

// GetTypingUsers 获取最近5秒内正在输入的用户
func (a *App) GetTypingUsers() Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	var channelID string
	if mode == ModeServer && srv != nil {
		ch, _ := srv.GetChannel()
		if ch == nil {
			return NewErrorResponse("no_channel", "未初始化频道", "")
		}
		channelID = ch.ID
	} else if mode == ModeClient && cli != nil {
		channelID = cli.GetChannelID()
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	list, err := a.db.MessageRepo().GetTypingUsers(channelID)
	if err != nil {
		return NewErrorResponse("db_error", "获取输入用户失败", err.Error())
	}

	return NewSuccessResponse(list)
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

	if mode != ModeClient || cli == nil {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err := cli.SendReaction(messageID, emoji); err != nil {
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

	if mode != ModeClient || cli == nil {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err := cli.RemoveReactionNetwork(messageID, emoji); err != nil {
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

	// 加载并聚合 reactions
	reactions := make([]MessageReaction, 0)
	if a.db != nil {
		dbReactions, err := a.db.MessageRepo().GetReactions(msg.ID)
		if err == nil && len(dbReactions) > 0 {
			agg := make(map[string]*MessageReaction)
			for _, r := range dbReactions {
				if r == nil {
					continue
				}
				if entry, ok := agg[r.Emoji]; ok {
					entry.Count++
					entry.UserIDs = append(entry.UserIDs, r.UserID)
				} else {
					agg[r.Emoji] = &MessageReaction{
						Emoji:   r.Emoji,
						UserIDs: []string{r.UserID},
						Count:   1,
					}
				}
			}
			for _, v := range agg {
				reactions = append(reactions, *v)
			}
		}
	}

	return &MessageDTO{
		ID:         msg.ID,
		ChannelID:  msg.ChannelID,
		SenderID:   msg.SenderID,
		SenderName: senderName,
		Type:       msg.Type,
		Content:    msg.Content,
		Timestamp:  msg.Timestamp.Unix(),
		EditedAt:   msg.EditedAt.Unix(),
		IsDeleted:  msg.IsDeleted,
		IsPinned:   msg.IsPinned,
		ReplyToID:  msg.ReplyToID,
		Reactions:  reactions,
	}
}
