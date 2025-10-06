package app

import (
	"fmt"

	"crosswire/internal/models"
	"crosswire/internal/server"
	"crosswire/internal/transport"
)

// ==================== 服务端模式 API ====================

// StartServerMode 启动服务端模式
func (a *App) StartServerMode(config ServerConfig) Response {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 检查是否已在运行
	if a.isRunning {
		return NewErrorResponse("already_running", "服务已在运行", fmt.Sprintf("当前模式: %s", a.mode))
	}

	// 验证配置
	if err := a.validateServerConfig(&config); err != nil {
		return NewErrorResponse("invalid_config", "配置无效", err.Error())
	}

	a.logger.Info("Starting server mode: %s", config.ChannelName)

	// 构造服务端配置
	srvCfg := &server.ServerConfig{
		ChannelID:       config.ChannelName, // 简化：使用名称作为ID，实际应为UUID
		ChannelPassword: config.Password,
		ChannelName:     config.ChannelName,
		MaxMembers:      config.MaxMembers,
		TransportMode:   config.TransportMode,
		TransportConfig: &transport.Config{
			Mode:      config.TransportMode,
			Interface: config.NetworkInterface,
			Port:      config.Port,
		},
	}

	// 创建服务端实例
	srv, err := server.NewServer(srvCfg, a.db, a.eventBus, a.logger)
	if err != nil {
		return NewErrorResponse("server_error", "服务端创建失败", err.Error())
	}

	// 启动服务端
	if err := srv.Start(); err != nil {
		return NewErrorResponse("start_error", "服务端启动失败", err.Error())
	}

	// 更新状态
	a.server = srv
	a.mode = ModeServer
	a.isRunning = true

	a.logger.Info("Server started successfully")

	// 发送事件到前端
	a.emitEvent(EventConnected, map[string]interface{}{
		"mode":         "server",
		"channel_name": config.ChannelName,
		"channel_id":   srv.GetConfig().ChannelID,
	})

	// 返回服务端状态
	status := a.getServerStatus()
	return NewSuccessResponse(status)
}

// StopServerMode 停止服务端模式
func (a *App) StopServerMode() Response {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mode != ModeServer || a.server == nil {
		return NewErrorResponse("not_running", "服务端未运行", "")
	}

	a.logger.Info("Stopping server mode")

	// 停止服务端
	if err := a.server.Stop(); err != nil {
		return NewErrorResponse("stop_error", "服务端停止失败", err.Error())
	}

	// 清理状态
	a.server = nil
	a.mode = ModeIdle
	a.isRunning = false

	a.logger.Info("Server stopped")

	// 发送事件到前端
	a.emitEvent(EventDisconnected, map[string]interface{}{
		"mode": "server",
	})

	return NewSuccessResponse(map[string]interface{}{
		"message": "服务端已停止",
	})
}

// GetServerStatus 获取服务端状态
func (a *App) GetServerStatus() Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.mode != ModeServer || a.server == nil {
		return NewErrorResponse("not_running", "服务端未运行", "")
	}

	status := a.getServerStatus()
	return NewSuccessResponse(status)
}

// getServerStatus 内部方法：获取服务端状态（需持有锁）
func (a *App) getServerStatus() *ServerStatus {
	if a.server == nil {
		return nil
	}

	channel, _ := a.server.GetChannel()
	stats := a.server.GetStats()
	members, _ := a.server.GetMembers()

	return &ServerStatus{
		Running:       true,
		ChannelID:     channel.ID,
		ChannelName:   channel.Name,
		TransportMode: string(channel.TransportMode),
		MemberCount:   len(members),
		StartTime:     stats.StartTime,
		NetworkStats:  &NetworkStats{},
	}
}

// ==================== 辅助方法 ====================

// validateServerConfig 验证服务端配置
func (a *App) validateServerConfig(config *ServerConfig) error {
	if config.ChannelName == "" {
		return fmt.Errorf("频道名称不能为空")
	}

	if config.Password == "" {
		return fmt.Errorf("频道密码不能为空")
	}

	if len(config.Password) < 6 {
		return fmt.Errorf("密码长度至少为6个字符")
	}

	// 验证传输模式特定配置
	switch config.TransportMode {
	case models.TransportARP:
		if config.NetworkInterface == "" {
			return fmt.Errorf("ARP模式需要指定网络接口")
		}
	case models.TransportHTTPS:
		if config.ListenAddress == "" {
			config.ListenAddress = "0.0.0.0"
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

	// 设置默认值
	if config.MaxMembers == 0 {
		config.MaxMembers = 100
	}

	if config.MaxFileSize == 0 {
		config.MaxFileSize = 100 * 1024 * 1024 // 100MB
	}

	return nil
}

// createTransport 旧接口（已不使用）
func (a *App) createTransport(mode models.TransportMode, netInterface, address string, port int) (transport.Transport, error) {
	factory := transport.NewFactory()
	cfg := &transport.Config{Mode: mode, Interface: netInterface, Port: port}
	return factory.CreateWithConfig(mode, cfg)
}
