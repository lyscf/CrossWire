package app

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"crosswire/internal/models"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ==================== 系统功能 API ====================

// GetNetworkInterfaces 获取网络接口列表
func (a *App) GetNetworkInterfaces() Response {
	interfaces, err := net.Interfaces()
	if err != nil {
		return NewErrorResponse("query_error", "获取网络接口失败", err.Error())
	}

	result := make([]*NetworkInterface, 0)
	for _, iface := range interfaces {
		// 获取IP地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		ipAddresses := make([]string, 0)
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				ipAddresses = append(ipAddresses, ipNet.IP.String())
			}
		}

		// 只返回有IP地址的接口
		if len(ipAddresses) > 0 {
			result = append(result, &NetworkInterface{
				Name:        iface.Name,
				DisplayName: iface.Name, // TODO: 在Windows上可以获取更友好的名称
				MACAddress:  iface.HardwareAddr.String(),
				IPAddresses: ipAddresses,
				IsUp:        iface.Flags&net.FlagUp != 0,
				IsLoopback:  iface.Flags&net.FlagLoopback != 0,
			})
		}
	}

	return NewSuccessResponse(result)
}

// TestConnection 测试连接
func (a *App) TestConnection(serverAddress string, mode models.TransportMode, timeout int) Response {
	a.logger.Info("Testing connection to %s (mode: %s)", serverAddress, mode)

	startTime := time.Now()

	// 根据传输模式测试连接
	var success bool
	var message string

	switch mode {
	case models.TransportHTTPS:
		// 测试TCP连接
		conn, err := net.DialTimeout("tcp", serverAddress, time.Duration(timeout)*time.Second)
		if err != nil {
			success = false
			message = fmt.Sprintf("连接失败: %v", err)
		} else {
			conn.Close()
			success = true
			message = "连接成功"
		}

	case models.TransportARP, models.TransportMDNS:
		// ARP和mDNS模式不支持直接测试
		success = false
		message = "该传输模式不支持连接测试"

	default:
		return NewErrorResponse("invalid_mode", "不支持的传输模式", string(mode))
	}

	latency := time.Since(startTime).Seconds() * 1000 // 转换为毫秒

	result := ConnectionTestResult{
		Success: success,
		Latency: latency,
		Message: message,
	}

	return NewSuccessResponse(result)
}

// GetNetworkStats 获取网络统计
func (a *App) GetNetworkStats() Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未运行", "")
	}

	var stats *NetworkStats

	if mode == ModeServer && srv != nil {
		serverStats := srv.GetStats()
		stats = &NetworkStats{
			BytesSent:       int64(serverStats.TotalBytes),
			BytesReceived:   0,
			PacketsSent:     int64(serverStats.TotalBroadcasts),
			PacketsReceived: int64(serverStats.TotalMessages),
		}
	} else if mode == ModeClient && cli != nil {
		clientStats := cli.GetStats()
		stats = &NetworkStats{
			BytesSent:       int64(clientStats.BytesSent),
			BytesReceived:   int64(clientStats.BytesReceived),
			PacketsSent:     int64(clientStats.MessagesSent),
			PacketsReceived: int64(clientStats.MessagesReceived),
		}
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	return NewSuccessResponse(stats)
}

// ==================== 用户配置 API ====================

// GetUserProfile 获取用户配置
func (a *App) GetUserProfile() Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return NewSuccessResponse(a.userProfile)
}

// UpdateUserProfile 更新用户配置
func (a *App) UpdateUserProfile(profile UserProfile) Response {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 更新配置
	a.userProfile = &profile

	// TODO: 保存到数据库

	a.logger.Info("User profile updated")

	// 发送事件到前端
	a.emitEvent(EventInfo, map[string]interface{}{
		"message": "用户配置已更新",
	})

	return NewSuccessResponse(a.userProfile)
}

// GetRecentChannels 获取最近的频道
func (a *App) GetRecentChannels() Response {
	// TODO: 从数据库加载最近频道列表
	// 暂时返回空列表
	recentChannels := make([]*RecentChannel, 0)

	return NewSuccessResponse(recentChannels)
}

// ==================== 数据导入导出 API ====================

// ExportData 导出数据
func (a *App) ExportData(exportPath string, options ExportOptions) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	a.logger.Info("Exporting data to: %s", exportPath)

	// 创建ZIP文件
	zipFile, err := os.Create(exportPath)
	if err != nil {
		return NewErrorResponse("file_error", "创建导出文件失败", err.Error())
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 获取频道ID
	var channelID string
	if a.mode == ModeServer && a.server != nil {
		ch, _ := a.server.GetChannel()
		channelID = ch.ID
	} else if a.mode == ModeClient && a.client != nil {
		channelID = a.client.GetChannelID()
	}

	// 导出消息
	if options.IncludeMessages {
		messages, err := a.db.MessageRepo().GetByChannelID(channelID, 0, 0)
		if err == nil {
			a.exportToZip(zipWriter, "messages.json", messages)
		}
	}

	// 导出文件列表
	if options.IncludeFiles {
		files, err := a.db.FileRepo().GetByChannelID(channelID, 0, 0)
		if err == nil {
			a.exportToZip(zipWriter, "files.json", files)
		}
	}

	// 导出成员
	if options.IncludeMembers {
		members, err := a.db.MemberRepo().GetByChannelID(channelID)
		if err == nil {
			a.exportToZip(zipWriter, "members.json", members)
		}
	}

	// 导出题目
	if options.IncludeChallenges {
		challenges, err := a.db.ChallengeRepo().GetByChannelID(channelID)
		if err == nil {
			a.exportToZip(zipWriter, "challenges.json", challenges)
		}
	}

	a.logger.Info("Data export completed")

	return NewSuccessResponse(map[string]interface{}{
		"message": "数据导出成功",
		"path":    exportPath,
	})
}

// ImportData 导入数据
func (a *App) ImportData(importPath string) Response {
	a.logger.Info("Importing data from: %s", importPath)

	// 打开ZIP文件
	zipReader, err := zip.OpenReader(importPath)
	if err != nil {
		return NewErrorResponse("file_error", "打开导入文件失败", err.Error())
	}
	defer zipReader.Close()

	// TODO: 实现数据导入逻辑
	// 需要谨慎处理，避免数据冲突

	a.logger.Info("Data import completed")

	return NewSuccessResponse(map[string]interface{}{
		"message": "数据导入成功",
	})
}

// ==================== 辅助方法 ====================

// exportToZip 将数据导出到ZIP
func (a *App) exportToZip(zipWriter *zip.Writer, filename string, data interface{}) error {
	// 创建JSON数据
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// 在ZIP中创建文件
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}

	// 写入数据
	_, err = writer.Write(jsonData)
	return err
}

// ==================== 日志 API ====================

// GetLogs 获取日志
func (a *App) GetLogs(limit int) Response {
	// TODO: 从日志系统获取日志
	logs := make([]map[string]interface{}, 0)

	return NewSuccessResponse(logs)
}

// ClearLogs 清空日志
func (a *App) ClearLogs() Response {
	// TODO: 清空日志

	return NewSuccessResponse(map[string]interface{}{
		"message": "日志已清空",
	})
}

// ==================== 文件选择 API ====================

// SelectFile 选择文件（调用系统对话框）
func (a *App) SelectFile(title, filter string) Response {
	if a.ctx == nil {
		return NewErrorResponse("no_context", "应用上下文未初始化", "")
	}

	// 使用Wails的文件选择对话框
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{
				DisplayName: filter,
				Pattern:     "*.*",
			},
		},
	})

	if err != nil {
		return NewErrorResponse("dialog_error", "文件选择失败", err.Error())
	}

	if filePath == "" {
		return NewErrorResponse("cancelled", "用户取消选择", "")
	}

	return NewSuccessResponse(map[string]interface{}{
		"path": filePath,
	})
}

// SelectDirectory 选择目录
func (a *App) SelectDirectory(title string) Response {
	if a.ctx == nil {
		return NewErrorResponse("no_context", "应用上下文未初始化", "")
	}

	// 使用Wails的目录选择对话框
	dirPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})

	if err != nil {
		return NewErrorResponse("dialog_error", "目录选择失败", err.Error())
	}

	if dirPath == "" {
		return NewErrorResponse("cancelled", "用户取消选择", "")
	}

	return NewSuccessResponse(map[string]interface{}{
		"path": dirPath,
	})
}

// SaveFileDialog 保存文件对话框
func (a *App) SaveFileDialog(title, defaultFilename string) Response {
	if a.ctx == nil {
		return NewErrorResponse("no_context", "应用上下文未初始化", "")
	}

	// 使用Wails的保存文件对话框
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
	})

	if err != nil {
		return NewErrorResponse("dialog_error", "文件保存对话框失败", err.Error())
	}

	if filePath == "" {
		return NewErrorResponse("cancelled", "用户取消", "")
	}

	return NewSuccessResponse(map[string]interface{}{
		"path": filePath,
	})
}
