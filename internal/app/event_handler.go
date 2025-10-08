package app

import (
	"crosswire/internal/events"
)

// ==================== 事件处理器 ====================
// 负责订阅后端事件总线，并将事件转发到前端

// subscribeEvents 订阅后端事件
func (a *App) subscribeEvents() {
	if a.eventBus == nil {
		a.logger.Warn("Event bus is nil, cannot subscribe to events")
		return
	}

	a.logger.Info("Subscribing to backend events...")

	// 订阅消息事件
	a.subscribeMessageEvents()

	// 订阅成员事件
	a.subscribeMemberEvents()

	// 订阅文件事件
	a.subscribeFileEvents()

	// 订阅题目事件
	a.subscribeChallengeEvents()

	// 订阅系统事件
	a.subscribeSystemEvents()

	a.logger.Info("Event subscription completed")
}

// ==================== 消息事件 ====================

func (a *App) subscribeMessageEvents() {
	// 消息接收
	a.eventBus.Subscribe(events.EventMessageReceived, func(ev *events.Event) {
		if ev == nil || ev.Data == nil {
			return
		}
		a.emitEvent(EventMessageReceived, ev.Data)

		// 提及通知：当消息包含 Mentions 时，为每个被提及用户发一条通知
		me, ok := ev.Data.(*events.MessageEvent)
		if !ok || me == nil || me.Message == nil {
			return
		}
		mentions := me.Message.Mentions
		if len(mentions) == 0 {
			return
		}
		for _, uid := range mentions {
			a.emitEvent(EventInfo, map[string]interface{}{
				"type":    "message:mention",
				"user_id": uid,
				"sender":  me.Message.SenderNickname,
				"channel": me.Message.ChannelID,
				"content": me.Message.ContentText,
			})
		}
	})

	// 消息发送
	a.eventBus.Subscribe(events.EventMessageSent, func(ev *events.Event) {
		a.emitEvent(EventMessageSent, ev.Data)
	})

	// 消息更新
	a.eventBus.Subscribe(events.EventMessageUpdated, func(ev *events.Event) {
		a.emitEvent(EventMessageUpdated, ev.Data)
	})

	// 消息删除
	a.eventBus.Subscribe(events.EventMessageDeleted, func(ev *events.Event) {
		a.emitEvent(EventMessageDeleted, ev.Data)
	})

	// 消息反应新增
	a.eventBus.Subscribe(events.EventReactionAdded, func(ev *events.Event) {
		a.emitEvent("message:reaction:added", ev.Data)
	})

	// 消息反应移除
	a.eventBus.Subscribe(events.EventReactionRemoved, func(ev *events.Event) {
		a.emitEvent("message:reaction:removed", ev.Data)
	})
}

// ==================== 成员事件 ====================

func (a *App) subscribeMemberEvents() {
	// 成员加入
	a.eventBus.Subscribe(events.EventMemberJoined, func(ev *events.Event) {
		a.emitEvent(EventMemberJoined, ev.Data)
	})

	// 成员离开
	a.eventBus.Subscribe(events.EventMemberLeft, func(ev *events.Event) {
		a.emitEvent(EventMemberLeft, ev.Data)
	})

	// 成员更新
	a.eventBus.Subscribe(events.EventMemberUpdated, func(ev *events.Event) {
		a.emitEvent(EventMemberUpdated, ev.Data)
	})

	// 成员被踢出
	a.eventBus.Subscribe(events.EventMemberKicked, func(ev *events.Event) {
		a.emitEvent(EventMemberKicked, ev.Data)
	})

	// 成员被封禁
	a.eventBus.Subscribe(events.EventMemberBanned, func(ev *events.Event) {
		a.emitEvent(EventMemberBanned, ev.Data)
	})
}

// ==================== 文件事件 ====================

func (a *App) subscribeFileEvents() {
	// 文件上传开始
	a.eventBus.Subscribe(events.EventFileUploadStarted, func(ev *events.Event) {
		a.emitEvent(EventFileUploadStarted, ev.Data)
	})

	// 文件上传进度
	a.eventBus.Subscribe(events.EventFileUploadProgress, func(ev *events.Event) {
		a.emitEvent(EventFileUploadProgress, ev.Data)
	})

	// 文件上传完成
	a.eventBus.Subscribe(events.EventFileUploaded, func(ev *events.Event) {
		a.emitEvent(EventFileUploadCompleted, ev.Data)
	})

	// 文件上传失败
	a.eventBus.Subscribe(events.EventFileUploadFailed, func(ev *events.Event) {
		a.emitEvent(EventFileUploadFailed, ev.Data)
	})

	// 文件下载开始
	a.eventBus.Subscribe(events.EventFileDownloadStarted, func(ev *events.Event) {
		a.emitEvent(EventFileDownloadStarted, ev.Data)
	})

	// 文件下载进度
	a.eventBus.Subscribe(events.EventFileDownloadProgress, func(ev *events.Event) {
		a.emitEvent(EventFileDownloadProgress, ev.Data)
	})

	// 文件下载完成
	a.eventBus.Subscribe(events.EventFileDownloadCompleted, func(ev *events.Event) {
		a.emitEvent(EventFileDownloadCompleted, ev.Data)
	})

	// 文件下载失败
	a.eventBus.Subscribe(events.EventFileDownloadFailed, func(ev *events.Event) {
		a.emitEvent(EventFileDownloadFailed, ev.Data)
	})

	// 文件删除
	a.eventBus.Subscribe(events.EventFileDeleted, func(ev *events.Event) {
		a.emitEvent(EventFileDeleted, ev.Data)
	})
}

// ==================== 题目事件 ====================

func (a *App) subscribeChallengeEvents() {
	// 题目创建
	a.eventBus.Subscribe(events.EventChallengeCreated, func(ev *events.Event) {
		a.emitEvent(EventChallengeCreated, ev.Data)
	})

	// 题目更新
	a.eventBus.Subscribe(events.EventChallengeUpdated, func(ev *events.Event) {
		a.emitEvent(EventChallengeUpdated, ev.Data)
	})

	// 题目解决
	a.eventBus.Subscribe(events.EventChallengeSolved, func(ev *events.Event) {
		a.emitEvent(EventChallengeSolved, ev.Data)
	})

	// 题目分配
	a.eventBus.Subscribe(events.EventChallengeAssigned, func(ev *events.Event) {
		a.emitEvent(EventChallengeAssigned, ev.Data)
	})

	// 题目进度
	a.eventBus.Subscribe(events.EventChallengeProgress, func(ev *events.Event) {
		a.emitEvent(EventChallengeProgress, ev.Data)
	})
}

// ==================== 系统事件 ====================

func (a *App) subscribeSystemEvents() {
	// 连接事件
	a.eventBus.Subscribe(events.EventSystemConnected, func(ev *events.Event) {
		a.emitEvent(EventConnected, ev.Data)
	})

	// 断开连接
	a.eventBus.Subscribe(events.EventSystemDisconnect, func(ev *events.Event) {
		a.emitEvent(EventDisconnected, ev.Data)
	})

	// 重连中
	a.eventBus.Subscribe(events.EventSystemReconnect, func(ev *events.Event) {
		a.emitEvent(EventReconnecting, ev.Data)
	})

	// 错误事件
	a.eventBus.Subscribe(events.EventSystemError, func(ev *events.Event) {
		a.emitEvent(EventError, ev.Data)
	})

	// 警告/信息事件：沿用更新事件作为占位
	a.eventBus.Subscribe(events.EventMessageUpdated, func(ev *events.Event) {
		a.emitEvent(EventWarning, ev.Data)
	})
	a.eventBus.Subscribe(events.EventMessageUpdated, func(ev *events.Event) {
		a.emitEvent(EventInfo, ev.Data)
	})
}
