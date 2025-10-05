package app

import (
	"context"
	"fmt"

	"crosswire/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App Wails 应用主类
type App struct {
	ctx context.Context
	db  *storage.Database

	// TODO: 添加以下字段
	// mode      Mode                     // 运行模式（服务端/客户端）
	// server    *server.Server           // 服务端实例
	// client    *client.Client           // 客户端实例
	// transport transport.Transport      // 传输层
	// crypto    *crypto.Manager          // 加密管理器
	// eventBus  *events.EventBus         // 事件总线
	// logger    *logger.Logger           // 日志器
}

// NewApp 创建应用实例
func NewApp(db *storage.Database) *App {
	return &App{
		db: db,
	}
}

// Startup 应用启动时调用
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("CrossWire starting up...")

	// TODO: 初始化各个模块
	// 1. 加载用户配置
	// 2. 初始化加密模块
	// 3. 初始化事件总线
	// 4. 初始化日志系统
	// 5. 检查更新
}

// DomReady DOM 准备完成时调用
func (a *App) DomReady(ctx context.Context) {
	fmt.Println("DOM is ready")

	// 发送初始化数据到前端
	runtime.EventsEmit(ctx, "app:ready", map[string]interface{}{
		"version": "1.0.0",
		"status":  "initialized",
	})
}

// Shutdown 应用关闭时调用
func (a *App) Shutdown(ctx context.Context) {
	fmt.Println("CrossWire shutting down...")

	// TODO: 清理资源
	// 1. 保存用户配置
	// 2. 关闭服务端/客户端
	// 3. 关闭传输层
	// 4. 关闭数据库连接
	// 5. 清理临时文件
}

// Greet 示例方法
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, welcome to CrossWire!", name)
}

// GetAppVersion 获取应用版本
func (a *App) GetAppVersion() string {
	return "1.0.0"
}

// TODO: 实现以下导出方法（供前端调用）
//
// 模式切换
// - StartServerMode(config ServerConfig) error
// - StartClientMode(config ClientConfig) error
// - GetCurrentMode() Mode
// - StopCurrentMode() error
//
// 频道管理
// - CreateChannel(name, password string, config ChannelConfig) (*Channel, error)
// - JoinChannel(serverAddress, password string) error
// - LeaveChannel() error
// - GetChannelInfo() (*Channel, error)
//
// 消息操作
// - SendMessage(content string, msgType MessageType) error
// - SendCodeMessage(code, language string) error
// - GetMessages(limit, offset int) ([]*Message, error)
// - SearchMessages(query string) ([]*Message, error)
// - DeleteMessage(messageID string) error
// - PinMessage(messageID string) error
// - ReactToMessage(messageID, emoji string) error
//
// 文件操作
// - UploadFile(filePath string) error
// - DownloadFile(fileID, savePath string) error
// - GetFiles(limit, offset int) ([]*File, error)
//
// 成员管理
// - GetMembers() ([]*Member, error)
// - UpdateMyStatus(status UserStatus) error
// - UpdateMyProfile(profile UserProfile) error
// - KickMember(memberID, reason string) error
// - MuteMember(memberID string, duration int64) error
// - UnmuteMember(memberID string) error
//
// 题目管理
// - CreateChallenge(challenge Challenge) (*Challenge, error)
// - GetChallenges() ([]*Challenge, error)
// - GetChallenge(challengeID string) (*Challenge, error)
// - UpdateChallenge(challengeID string, updates ChallengeUpdate) error
// - DeleteChallenge(challengeID string) error
// - AssignChallenge(challengeID string, memberIDs []string) error
// - SubmitFlag(challengeID, flag string) (*SubmissionResult, error)
// - UpdateProgress(challengeID string, progress int, summary string) error
// - AddHint(challengeID, content string, cost int) error
// - UnlockHint(hintID string) error
//
// 用户配置
// - GetUserProfile() (*UserProfile, error)
// - UpdateUserProfile(profile UserProfile) error
// - GetRecentChannels() ([]*RecentChannel, error)
//
// 系统功能
// - SelectNetworkInterface() ([]NetworkInterface, error)
// - TestConnection(serverAddress string, mode TransportMode) error
// - GetNetworkStats() (*NetworkStats, error)
// - ExportData(exportPath string) error
// - ImportData(importPath string) error
