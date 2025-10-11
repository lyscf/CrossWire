package app

import (
	"context"
	"sync"
	"time"

	"crosswire/internal/client"
	"crosswire/internal/events"
	"crosswire/internal/models"
	"crosswire/internal/server"
	"crosswire/internal/storage"
	"crosswire/internal/utils"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App Wails 应用主类
type App struct {
	ctx context.Context
	db  *storage.Database

	// 核心组件
	mode     Mode             // 运行模式（服务端/客户端）
	server   *server.Server   // 服务端实例
	client   *client.Client   // 客户端实例
	eventBus *events.EventBus // 事件总线
	logger   *utils.Logger    // 日志器

	// 状态管理
	mu        sync.RWMutex // 读写锁
	isRunning bool         // 是否运行中

	// 用户配置
	userProfile *UserProfile // 用户配置
}

// NewApp 创建应用实例
func NewApp(db *storage.Database) *App {
	// 初始化事件总线（使用默认配置）
	eventBus := events.NewEventBus(nil)

	// 初始化日志器（Info级别，日志目录为./logs）
	logger, err := utils.NewLogger(utils.LogLevelInfo, "./logs")
	if err != nil {
		// 如果日志初始化失败，使用标准输出
		logger = &utils.Logger{}
	}

	// 加载用户配置
	userProfile := loadUserProfile(db)

	return &App{
		db:          db,
		mode:        ModeIdle,
		eventBus:    eventBus,
		logger:      logger,
		userProfile: userProfile,
		isRunning:   false,
	}
}

// loadUserProfile 加载用户配置
func loadUserProfile(db *storage.Database) *UserProfile {
	// 从 user.db 加载用户配置（若不存在，返回默认配置）
	if db == nil || db.GetUserDB() == nil {
		return getDefaultUserProfile()
	}

	var up models.UserProfile
	if err := db.GetUserDB().First(&up).Error; err != nil {
		// 未初始化，返回默认配置
		return getDefaultUserProfile()
	}

	// 映射到 App 层 Profile
	return &UserProfile{
		Nickname: up.Nickname,
		Avatar:   up.Avatar,
		Status:   models.StatusOnline,
		Theme:    up.Theme,
		Language: up.Language,
		Notifications: NotificationSettings{
			Enabled:     true,
			Sound:       true,
			Desktop:     true,
			MentionOnly: false,
		},
	}
}

func getDefaultUserProfile() *UserProfile {
	return &UserProfile{
		Nickname: "User",
		Avatar:   "",
		Status:   models.StatusOnline,
		Theme:    "light",
		Language: "zh-CN",
		Notifications: NotificationSettings{
			Enabled:     true,
			Sound:       true,
			Desktop:     true,
			MentionOnly: false,
		},
	}
}

// Startup 应用启动时调用
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.logger.Info("CrossWire starting up...")

	// 订阅事件总线，将后端事件转发到前端
	a.subscribeEvents()

	a.logger.Info("CrossWire startup completed")
}

// DomReady DOM 准备完成时调用
func (a *App) DomReady(ctx context.Context) {
	a.logger.Info("DOM is ready")

	// 发送初始化数据到前端
	runtime.EventsEmit(ctx, "app:ready", map[string]interface{}{
		"version": a.GetAppVersion(),
		"status":  "initialized",
		"mode":    string(a.mode),
		"profile": a.userProfile,
	})
}

// Shutdown 应用关闭时调用
func (a *App) Shutdown(ctx context.Context) {
	a.logger.Info("CrossWire shutting down...")

	a.mu.Lock()
	defer a.mu.Unlock()

	// 停止当前模式
	if a.mode == ModeServer && a.server != nil {
		if err := a.server.Stop(); err != nil {
			a.logger.Error("Failed to stop server: %v", err)
		}
		a.server = nil
	}

	if a.mode == ModeClient && a.client != nil {
		if err := a.client.Stop(); err != nil {
			a.logger.Error("Failed to stop client: %v", err)
		}
		a.client = nil
	}

	// 保存用户配置（如果存在）
	if a.userProfile != nil && a.db != nil && a.db.GetUserDB() != nil {
		// 读取或创建第一条用户配置记录
		var up models.UserProfile
		if err := a.db.GetUserDB().First(&up).Error; err != nil {
			// 创建默认记录
			up.ID = "default"
			up.Nickname = a.userProfile.Nickname
			up.Avatar = a.userProfile.Avatar
			up.Theme = a.userProfile.Theme
			up.Language = a.userProfile.Language
			_ = a.db.GetUserDB().Create(&up).Error
		} else {
			up.Nickname = a.userProfile.Nickname
			up.Avatar = a.userProfile.Avatar
			up.Theme = a.userProfile.Theme
			up.Language = a.userProfile.Language
			_ = a.db.GetUserDB().Save(&up).Error
		}
	}

	a.mode = ModeIdle
	a.isRunning = false

	a.logger.Info("CrossWire shutdown completed")
}

// ==================== 基础方法 ====================

// GetAppVersion 获取应用版本
func (a *App) GetAppVersion() string {
	return "1.0.0"
}

// GetCurrentMode 获取当前运行模式
func (a *App) GetCurrentMode() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return string(a.mode)
}

// IsRunning 检查是否正在运行
func (a *App) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isRunning
}

// ==================== 事件处理 ====================

// emitEvent 向前端发送事件
func (a *App) emitEvent(eventType string, data interface{}) {
	if a.ctx == nil {
		return
	}

	event := AppEvent{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}

	// 事件转发调试日志：事件名与数据长度（若可计算）
	size := 0
	switch v := data.(type) {
	case string:
		size = len(v)
	case []byte:
		size = len(v)
	}
	a.logger.Debug("[App] emitEvent type=%s size=%d", eventType, size)

	runtime.EventsEmit(a.ctx, "app:event", event)
}
