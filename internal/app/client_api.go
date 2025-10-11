package app

import (
	"fmt"
	"time"

	"crosswire/internal/client"
	"crosswire/internal/models"
	"crosswire/internal/transport"
)

// ==================== 客户端模式 API ====================

// StartClientMode 启动客户端模式并加入频道
func (a *App) StartClientMode(config ClientConfig) Response {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 检查是否已在运行
	if a.isRunning {
		return NewErrorResponse("already_running", "客户端已在运行", fmt.Sprintf("当前模式: %s", a.mode))
	}

	// 验证配置
	if err := a.validateClientConfig(&config); err != nil {
		return NewErrorResponse("invalid_config", "配置无效", err.Error())
	}

	a.logger.Info("Starting client mode: transport=%s, https=%s:%d, channel_id=%s", config.TransportMode, config.ServerAddress, config.Port, config.ChannelID)

	// 创建客户端配置
	clientConfig := &client.Config{
		ChannelID:       config.ChannelID,
		ChannelPassword: config.Password,
		Nickname:        config.Nickname,
		Avatar:          config.Avatar,
		TransportMode:   config.TransportMode,
		TransportConfig: &transport.Config{
			Mode:          config.TransportMode,
			Interface:     config.NetworkInterface,
			Port:          config.Port,
			Logger:        a.logger,
			ServerAddress: config.ServerAddress,
			SkipTLSVerify: true,
		},
		SyncInterval:    5 * time.Second,
		MaxSyncMessages: 1000,
		CacheSize:       5000,
		CacheDuration:   24 * time.Hour,
		JoinTimeout:     30 * time.Second,
		SyncTimeout:     10 * time.Second,
		DataDir:         "./data",
	}

	// 创建客户端实例
	cli, err := client.NewClient(clientConfig, a.db, a.eventBus)
	if err != nil {
		return NewErrorResponse("client_error", "客户端创建失败", err.Error())
	}

	// 启动客户端
	if err := cli.Start(); err != nil {
		a.logger.Error("Client start failed: %v", err)
		return NewErrorResponse("start_error", "客户端启动失败", err.Error())
	}

	// 更新状态
	a.client = cli
	a.mode = ModeClient
	a.isRunning = true

	a.logger.Info("Client started successfully")

	// 发送事件到前端
	a.emitEvent(EventConnected, map[string]interface{}{
		"mode":      "client",
		"nickname":  config.Nickname,
		"member_id": cli.GetMemberID(),
	})

	// 返回客户端状态
	status := a.getClientStatus()
	return NewSuccessResponse(status)
}

// StopClientMode 停止客户端模式
func (a *App) StopClientMode() Response {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mode != ModeClient || a.client == nil {
		return NewErrorResponse("not_running", "客户端未运行", "")
	}

	a.logger.Info("Stopping client mode")

	// 停止客户端
	if err := a.client.Stop(); err != nil {
		return NewErrorResponse("stop_error", "客户端停止失败", err.Error())
	}

	// 清理状态
	a.client = nil
	a.mode = ModeIdle
	a.isRunning = false

	a.logger.Info("Client stopped")

	// 发送事件到前端
	a.emitEvent(EventDisconnected, map[string]interface{}{
		"mode": "client",
	})

	return NewSuccessResponse(map[string]interface{}{
		"message": "客户端已停止",
	})
}

// GetClientStatus 获取客户端状态
func (a *App) GetClientStatus() Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.mode != ModeClient || a.client == nil {
		return NewErrorResponse("not_running", "客户端未运行", "")
	}

	status := a.getClientStatus()
	return NewSuccessResponse(status)
}

// getClientStatus 内部方法：获取客户端状态（需持有锁）
func (a *App) getClientStatus() *ClientStatus {
	if a.client == nil {
		return nil
	}

	channelInfo := a.client.GetChannelInfo()

	return &ClientStatus{
		Running:       true,
		Connected:     a.client.IsConnected(),
		ChannelID:     channelInfo.ID,
		ChannelName:   channelInfo.Name,
		MemberID:      a.client.GetMemberID(),
		TransportMode: string(channelInfo.TransportMode),
		ConnectTime:   a.client.GetConnectTime().Unix(),
	}
}

// ==================== 服务发现 API ====================

// DiscoverServers 发现本地网络中的服务器
func (a *App) DiscoverServers(timeout int) Response {
	a.mu.RLock()
	mode := a.mode
	cli := a.client
	a.mu.RUnlock()

	// 仅客户端模式支持服务发现
	if mode != ModeClient || cli == nil {
		return NewErrorResponse("invalid_mode", "仅客户端模式支持服务发现", "")
	}

	a.logger.Info("Discovering servers...")

	// 调用客户端的服务发现功能
	servers, err := cli.Discover(time.Duration(timeout) * time.Second)
	if err != nil {
		return NewErrorResponse("discovery_error", "服务发现失败", err.Error())
	}

	return NewSuccessResponse(servers)
}

// GetDiscoveredServers 获取已发现的服务器列表
func (a *App) GetDiscoveredServers() Response {
	a.mu.RLock()
	mode := a.mode
	cli := a.client
	a.mu.RUnlock()

	if mode != ModeClient || cli == nil {
		return NewErrorResponse("invalid_mode", "仅客户端模式可用", "")
	}

	servers := cli.GetDiscoveredServers()
	return NewSuccessResponse(servers)
}

// ==================== 辅助方法 ====================

// validateClientConfig 验证客户端配置
func (a *App) validateClientConfig(config *ClientConfig) error {
	if config.Password == "" {
		return fmt.Errorf("频道密码不能为空")
	}

	if len(config.Password) < 6 {
		return fmt.Errorf("密码长度至少为6个字符")
	}

	if config.Nickname == "" {
		config.Nickname = "User"
	}

	// 验证传输模式特定配置
	switch config.TransportMode {
	case models.TransportARP:
		if config.NetworkInterface == "" {
			return fmt.Errorf("ARP模式需要指定网络接口")
		}
	case models.TransportHTTPS:
		if config.ServerAddress == "" {
			return fmt.Errorf("HTTPS模式需要指定服务器地址")
		}
		if config.Port == 0 {
			config.Port = 8443
		}
	case models.TransportMDNS:
		if config.NetworkInterface == "" {
			return fmt.Errorf("mDNS模式需要指定网络接口")
		}
	default:
		return fmt.Errorf("不支持的传输模式: %s", config.TransportMode)
	}

	return nil
}
