package events

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventType 事件类型
// 参考: docs/ARCHITECTURE.md - 3.2.2 事件总线
type EventType string

const (
	// ===== 消息相关事件 =====
	EventMessageReceived EventType = "message:received" // 收到新消息
	EventMessageSent     EventType = "message:sent"     // 消息已发送
	EventMessageDeleted  EventType = "message:deleted"  // 消息被删除
	EventMessagePinned   EventType = "message:pinned"   // 消息被置顶
	EventMessageEdited   EventType = "message:edited"   // 消息被编辑

	// ===== 成员相关事件 =====
	EventMemberJoined      EventType = "member:joined"       // 成员加入
	EventMemberLeft        EventType = "member:left"         // 成员离开
	EventMemberKicked      EventType = "member:kicked"       // 成员被踢出
	EventMemberMuted       EventType = "member:muted"        // 成员被禁言
	EventMemberUnmuted     EventType = "member:unmuted"      // 成员解除禁言
	EventMemberBanned      EventType = "member:banned"       // 成员被封禁
	EventMemberUnbanned    EventType = "member:unbanned"     // 成员解除封禁
	EventMemberRoleChanged EventType = "member:role_changed" // 成员角色变更

	// ===== 状态相关事件 =====
	EventStatusChanged EventType = "status:changed" // 状态变化
	EventTypingStart   EventType = "typing:start"   // 开始输入
	EventTypingStop    EventType = "typing:stop"    // 停止输入

	// ===== 文件相关事件 =====
	EventFileUploaded   EventType = "file:uploaded"   // 文件上传完成
	EventFileDownloaded EventType = "file:downloaded" // 文件下载完成
	EventFileProgress   EventType = "file:progress"   // 文件传输进度

	// ===== 频道相关事件 =====
	EventChannelCreated EventType = "channel:created" // 频道创建
	EventChannelJoined  EventType = "channel:joined"  // 加入频道
	EventChannelLeft    EventType = "channel:left"    // 离开频道
	EventChannelUpdated EventType = "channel:updated" // 频道信息更新

	// ===== 系统相关事件 =====
	EventSystemError      EventType = "system:error"      // 系统错误
	EventSystemConnected  EventType = "system:connected"  // 连接成功
	EventSystemDisconnect EventType = "system:disconnect" // 连接断开
	EventSystemReconnect  EventType = "system:reconnect"  // 重新连接

	// ===== CTF挑战相关事件 =====
	EventChallengeCreated    EventType = "challenge:created"   // 题目创建
	EventChallengeAssigned   EventType = "challenge:assigned"  // 题目分配
	EventChallengeSubmitted  EventType = "challenge:submitted" // Flag提交
	EventChallengeSolved     EventType = "challenge:solved"    // 题目完成
	EventChallengeHintUnlock EventType = "challenge:hint"      // 提示解锁
	EventChallengeProgress   EventType = "challenge:progress"  // 题目进度更新
)

// Event 事件
type Event struct {
	Type      EventType   // 事件类型
	Data      interface{} // 事件数据
	Timestamp time.Time   // 时间戳
	Source    string      // 事件源
}

// EventHandler 事件处理函数
type EventHandler func(event *Event)

// Subscription 订阅信息
type Subscription struct {
	ID        string       // 订阅ID
	EventType EventType    // 事件类型
	Handler   EventHandler // 处理函数
	Filter    EventFilter  // 过滤器（可选）
	CreatedAt time.Time    // 创建时间
}

// EventFilter 事件过滤器
type EventFilter func(event *Event) bool

// EventBus 事件总线
// 参考: docs/ARCHITECTURE.md - 3.2.2 事件总线
type EventBus struct {
	// 订阅者映射: EventType -> []Subscription
	subscribers map[EventType][]*Subscription
	mutex       sync.RWMutex

	// 全局订阅者（监听所有事件）
	globalSubscribers []*Subscription
	globalMutex       sync.RWMutex

	// 事件队列（异步处理）
	eventQueue chan *Event
	queueSize  int

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 统计
	stats EventBusStats

	// 配置
	config *EventBusConfig
}

// EventBusConfig 事件总线配置
type EventBusConfig struct {
	QueueSize      int           // 事件队列大小
	WorkerCount    int           // 工作协程数
	EnableAsync    bool          // 是否启用异步处理
	EnableStats    bool          // 是否启用统计
	MaxRetries     int           // 最大重试次数
	RetryDelay     time.Duration // 重试延迟
	HandlerTimeout time.Duration // 处理函数超时时间
	EnableWildcard bool          // 是否启用通配符订阅
}

// EventBusStats 事件总线统计
type EventBusStats struct {
	EventsPublished uint64        // 发布的事件数
	EventsProcessed uint64        // 处理的事件数
	EventsDropped   uint64        // 丢弃的事件数
	Subscriptions   int           // 订阅数
	HandlerErrors   uint64        // 处理错误数
	AverageLatency  time.Duration // 平均延迟
	StartTime       time.Time     // 启动时间
	LastEventTime   time.Time     // 最后事件时间
	mutex           sync.RWMutex
}

// DefaultConfig 默认配置
var DefaultConfig = &EventBusConfig{
	QueueSize:      1000,
	WorkerCount:    4,
	EnableAsync:    true,
	EnableStats:    true,
	MaxRetries:     3,
	RetryDelay:     100 * time.Millisecond,
	HandlerTimeout: 5 * time.Second,
	EnableWildcard: false,
}

// NewEventBus 创建事件总线
func NewEventBus(config *EventBusConfig) *EventBus {
	if config == nil {
		config = DefaultConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	eb := &EventBus{
		subscribers:       make(map[EventType][]*Subscription),
		globalSubscribers: make([]*Subscription, 0),
		eventQueue:        make(chan *Event, config.QueueSize),
		queueSize:         config.QueueSize,
		ctx:               ctx,
		cancel:            cancel,
		config:            config,
	}

	eb.stats.StartTime = time.Now()

	// 启动异步处理工作协程
	if config.EnableAsync {
		for i := 0; i < config.WorkerCount; i++ {
			eb.wg.Add(1)
			go eb.worker()
		}
	}

	return eb
}

// Subscribe 订阅事件
// 参考: docs/ARCHITECTURE.md - 3.2.2 事件总线
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) string {
	return eb.SubscribeWithFilter(eventType, handler, nil)
}

// SubscribeWithFilter 订阅事件（带过滤器）
func (eb *EventBus) SubscribeWithFilter(eventType EventType, handler EventHandler, filter EventFilter) string {
	if handler == nil {
		return ""
	}

	subscription := &Subscription{
		ID:        generateSubscriptionID(),
		EventType: eventType,
		Handler:   handler,
		Filter:    filter,
		CreatedAt: time.Now(),
	}

	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	eb.subscribers[eventType] = append(eb.subscribers[eventType], subscription)

	// 更新统计
	if eb.config.EnableStats {
		eb.stats.mutex.Lock()
		eb.stats.Subscriptions = eb.countSubscriptions()
		eb.stats.mutex.Unlock()
	}

	return subscription.ID
}

// SubscribeAll 订阅所有事件
func (eb *EventBus) SubscribeAll(handler EventHandler) string {
	if handler == nil {
		return ""
	}

	subscription := &Subscription{
		ID:        generateSubscriptionID(),
		EventType: "*",
		Handler:   handler,
		CreatedAt: time.Now(),
	}

	eb.globalMutex.Lock()
	defer eb.globalMutex.Unlock()

	eb.globalSubscribers = append(eb.globalSubscribers, subscription)

	return subscription.ID
}

// Unsubscribe 取消订阅
func (eb *EventBus) Unsubscribe(subscriptionID string) bool {
	// 从普通订阅者中移除
	eb.mutex.Lock()
	for eventType, subs := range eb.subscribers {
		for i, sub := range subs {
			if sub.ID == subscriptionID {
				eb.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
				eb.mutex.Unlock()

				// 更新统计
				if eb.config.EnableStats {
					eb.stats.mutex.Lock()
					eb.stats.Subscriptions = eb.countSubscriptions()
					eb.stats.mutex.Unlock()
				}

				return true
			}
		}
	}
	eb.mutex.Unlock()

	// 从全局订阅者中移除
	eb.globalMutex.Lock()
	defer eb.globalMutex.Unlock()

	for i, sub := range eb.globalSubscribers {
		if sub.ID == subscriptionID {
			eb.globalSubscribers = append(eb.globalSubscribers[:i], eb.globalSubscribers[i+1:]...)
			return true
		}
	}

	return false
}

// UnsubscribeAll 取消所有订阅
func (eb *EventBus) UnsubscribeAll(eventType EventType) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	delete(eb.subscribers, eventType)

	// 更新统计
	if eb.config.EnableStats {
		eb.stats.mutex.Lock()
		eb.stats.Subscriptions = eb.countSubscriptions()
		eb.stats.mutex.Unlock()
	}
}

// Publish 发布事件
// 参考: docs/ARCHITECTURE.md - 3.2.2 事件总线
func (eb *EventBus) Publish(eventType EventType, data interface{}) {
	eb.PublishWithSource(eventType, data, "")
}

// PublishWithSource 发布事件（带来源）
func (eb *EventBus) PublishWithSource(eventType EventType, data interface{}, source string) {
	event := &Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
		Source:    source,
	}

	// 更新统计
	if eb.config.EnableStats {
		eb.stats.mutex.Lock()
		eb.stats.EventsPublished++
		eb.stats.LastEventTime = time.Now()
		eb.stats.mutex.Unlock()
	}

	// 异步处理
	if eb.config.EnableAsync {
		select {
		case eb.eventQueue <- event:
			// 事件已加入队列
		default:
			// 队列已满，丢弃事件
			if eb.config.EnableStats {
				eb.stats.mutex.Lock()
				eb.stats.EventsDropped++
				eb.stats.mutex.Unlock()
			}
		}
	} else {
		// 同步处理
		eb.processEvent(event)
	}
}

// PublishSync 同步发布事件（阻塞直到处理完成）
func (eb *EventBus) PublishSync(eventType EventType, data interface{}) {
	event := &Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	eb.processEvent(event)
}

// worker 工作协程（异步处理事件）
func (eb *EventBus) worker() {
	defer eb.wg.Done()

	for {
		select {
		case <-eb.ctx.Done():
			return
		case event := <-eb.eventQueue:
			eb.processEvent(event)
		}
	}
}

// processEvent 处理事件
func (eb *EventBus) processEvent(event *Event) {
	startTime := time.Now()

	// 调用全局订阅者
	eb.globalMutex.RLock()
	globalSubs := make([]*Subscription, len(eb.globalSubscribers))
	copy(globalSubs, eb.globalSubscribers)
	eb.globalMutex.RUnlock()

	for _, sub := range globalSubs {
		eb.invokeHandler(sub, event)
	}

	// 调用特定事件订阅者
	eb.mutex.RLock()
	subs := eb.subscribers[event.Type]
	subscribers := make([]*Subscription, len(subs))
	copy(subscribers, subs)
	eb.mutex.RUnlock()

	for _, sub := range subscribers {
		eb.invokeHandler(sub, event)
	}

	// 更新统计
	if eb.config.EnableStats {
		eb.stats.mutex.Lock()
		eb.stats.EventsProcessed++

		// 计算平均延迟
		latency := time.Since(startTime)
		if eb.stats.AverageLatency == 0 {
			eb.stats.AverageLatency = latency
		} else {
			eb.stats.AverageLatency = (eb.stats.AverageLatency + latency) / 2
		}

		eb.stats.mutex.Unlock()
	}
}

// invokeHandler 调用处理函数
func (eb *EventBus) invokeHandler(sub *Subscription, event *Event) {
	// 应用过滤器
	if sub.Filter != nil && !sub.Filter(event) {
		return
	}

	// 带超时的处理
	if eb.config.HandlerTimeout > 0 {
		done := make(chan bool, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					// 处理panic
					if eb.config.EnableStats {
						eb.stats.mutex.Lock()
						eb.stats.HandlerErrors++
						eb.stats.mutex.Unlock()
					}
				}
				done <- true
			}()

			sub.Handler(event)
		}()

		select {
		case <-done:
			// 处理完成
		case <-time.After(eb.config.HandlerTimeout):
			// 超时
			if eb.config.EnableStats {
				eb.stats.mutex.Lock()
				eb.stats.HandlerErrors++
				eb.stats.mutex.Unlock()
			}
		}
	} else {
		// 无超时保护
		defer func() {
			if r := recover(); r != nil {
				if eb.config.EnableStats {
					eb.stats.mutex.Lock()
					eb.stats.HandlerErrors++
					eb.stats.mutex.Unlock()
				}
			}
		}()

		sub.Handler(event)
	}
}

// GetStats 获取统计信息
func (eb *EventBus) GetStats() EventBusStats {
	eb.stats.mutex.RLock()
	defer eb.stats.mutex.RUnlock()

	stats := eb.stats
	stats.Subscriptions = eb.countSubscriptions()

	return stats
}

// countSubscriptions 统计订阅数
func (eb *EventBus) countSubscriptions() int {
	count := 0
	for _, subs := range eb.subscribers {
		count += len(subs)
	}
	count += len(eb.globalSubscribers)
	return count
}

// GetSubscriberCount 获取订阅者数量
func (eb *EventBus) GetSubscriberCount(eventType EventType) int {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	return len(eb.subscribers[eventType])
}

// HasSubscribers 检查是否有订阅者
func (eb *EventBus) HasSubscribers(eventType EventType) bool {
	return eb.GetSubscriberCount(eventType) > 0
}

// Close 关闭事件总线
func (eb *EventBus) Close() error {
	eb.cancel()

	// 等待工作协程完成
	eb.wg.Wait()

	// 清空订阅者
	eb.mutex.Lock()
	eb.subscribers = make(map[EventType][]*Subscription)
	eb.mutex.Unlock()

	eb.globalMutex.Lock()
	eb.globalSubscribers = make([]*Subscription, 0)
	eb.globalMutex.Unlock()

	close(eb.eventQueue)

	return nil
}

// generateSubscriptionID 生成订阅ID
func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

// TODO: 实现以下功能
// - 事件持久化（可选）
// - 事件重播
// - 优先级队列
// - 事件过滤器DSL
// - 事件链追踪
// - 性能监控
