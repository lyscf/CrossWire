package app

import (
	"archive/zip"
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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
		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 只处理启用的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 获取IP地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		ipAddresses := make([]string, 0)
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				// 只返回IPv4地址
				if ipv4 := ipNet.IP.To4(); ipv4 != nil {
					ipAddresses = append(ipAddresses, ipv4.String())
				}
			}
		}

		// 只返回有IPv4地址的接口
		if len(ipAddresses) > 0 {
			a.logger.Debug("Found network interface: %s, IPs: %v", iface.Name, ipAddresses)
			result = append(result, &NetworkInterface{
				Name:        iface.Name, // 系统内部名称（用于后端）
				DisplayName: iface.Name, // 显示名称（用于前端）
				MACAddress:  iface.HardwareAddr.String(),
				IPAddresses: ipAddresses,
				IsUp:        iface.Flags&net.FlagUp != 0,
				IsLoopback:  iface.Flags&net.FlagLoopback != 0,
			})
		}
	}

	a.logger.Info("Found %d available network interfaces", len(result))

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

// FetchHTTPSInfo 直连服务器 /info（支持自签名）并返回频道信息
func (a *App) FetchHTTPSInfo(server string, port int, skipTLSVerify bool, timeoutSec int) Response {
	if server == "" || port <= 0 {
		return NewErrorResponse("invalid_params", "服务器地址或端口无效", "")
	}
	if timeoutSec <= 0 {
		timeoutSec = 5
	}

	url := fmt.Sprintf("https://%s:%d/info", server, port)
	tr := &http.Transport{}
	if skipTLSVerify {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Transport: tr, Timeout: time.Duration(timeoutSec) * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		// 回退尝试 http
		url2 := fmt.Sprintf("http://%s:%d/info", server, port)
		resp, err = client.Get(url2)
		if err != nil {
			return NewErrorResponse("http_error", "获取频道信息失败", err.Error())
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return NewErrorResponse("bad_status", "获取频道信息失败", fmt.Sprintf("status=%d", resp.StatusCode))
	}
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return NewErrorResponse("decode_error", "解析频道信息失败", err.Error())
	}
	return NewSuccessResponse(data)
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
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		a.logger.Warn("[UpdateUserProfile] Not running")
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 获取当前用户ID
	var memberID string
	if mode == ModeServer && srv != nil {
		memberID = "server"
		a.logger.Debug("[UpdateUserProfile] Server mode, using member ID: %s", memberID)
	} else if mode == ModeClient && cli != nil {
		memberID = cli.GetMemberID()
		a.logger.Debug("[UpdateUserProfile] Client mode, member ID: %s", memberID)
	} else {
		a.logger.Error("[UpdateUserProfile] Invalid mode: %s", mode)
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	// 从数据库获取成员信息
	a.logger.Debug("[UpdateUserProfile] Fetching member info for ID: %s", memberID)
	member, err := a.db.MemberRepo().GetByID(memberID)
	if err != nil {
		a.logger.Error("[UpdateUserProfile] Failed to get member info: %v", err)
		return NewErrorResponse("not_found", "获取用户信息失败", err.Error())
	}

	// 更新成员信息
	if profile.Nickname != "" {
		member.Nickname = profile.Nickname
	}
	if profile.Avatar != "" {
		member.Avatar = profile.Avatar
	}

	// 更新技能标签（优先使用 SkillDetails，向后兼容 Skills）
	if len(profile.SkillDetails) > 0 {
		skills := make(models.SkillTags, len(profile.SkillDetails))
		for i, d := range profile.SkillDetails {
			skills[i] = models.SkillTag{
				Category:   d.Category,
				Level:      d.Level,
				Experience: d.Experience,
				LastUsed:   time.Now(),
			}
		}
		member.Skills = skills
	} else if len(profile.Skills) > 0 {
		skills := make(models.SkillTags, len(profile.Skills))
		for i, skillName := range profile.Skills {
			skills[i] = models.SkillTag{
				Category: skillName,
				Level:    2, // 默认中级
			}
		}
		member.Skills = skills
	}

	// 更新元数据（Email和Bio）
	if member.Metadata == nil {
		member.Metadata = make(map[string]interface{})
	}
	if profile.Email != "" {
		member.Metadata["email"] = profile.Email
	}
	if profile.Bio != "" {
		member.Metadata["bio"] = profile.Bio
	}

	// 保存到数据库
	a.logger.Debug("[UpdateUserProfile] Saving member info to database...")
	if err := a.db.MemberRepo().Update(member); err != nil {
		a.logger.Error("[UpdateUserProfile] Failed to update member: %v", err)
		return NewErrorResponse("update_error", "更新用户信息失败", err.Error())
	}

	a.logger.Info("[UpdateUserProfile] Successfully updated profile for member: %s", memberID)

	// 更新内存中的用户配置
	a.mu.Lock()
	a.userProfile = &profile
	a.mu.Unlock()

	// 发送事件到前端
	a.emitEvent(EventInfo, map[string]interface{}{
		"message": "用户配置已更新",
	})

	return NewSuccessResponse(map[string]interface{}{
		"message": "资料已更新",
		"profile": a.memberToDTO(member),
	})
}

// GetRecentChannels 获取最近的频道
func (a *App) GetRecentChannels() Response {
	// 从 user.db 加载最近频道记录，按最近加入时间倒序
	udb := a.db.GetUserDB()
	if udb == nil {
		return NewSuccessResponse([]*RecentChannel{})
	}
	var rows []models.RecentChannel
	if err := udb.Order("last_joined DESC").Limit(20).Find(&rows).Error; err != nil {
		return NewErrorResponse("db_error", "获取最近频道失败", err.Error())
	}
	out := make([]*RecentChannel, 0, len(rows))
	for _, rc := range rows {
		out = append(out, &RecentChannel{
			ChannelID:   rc.ChannelID,
			ChannelName: rc.ChannelName,
			LastJoined:  rc.LastJoined.Unix(),
			Mode:        a.mode,
		})
	}
	return NewSuccessResponse(out)
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

	// 简化导入：识别若干固定文件并入库，忽略未知文件
	// 注意：仅导入到当前打开的频道数据库
	var channelID string
	if a.mode == ModeServer && a.server != nil {
		ch, _ := a.server.GetChannel()
		channelID = ch.ID
	} else if a.mode == ModeClient && a.client != nil {
		channelID = a.client.GetChannelID()
	}

	if channelID == "" {
		return NewErrorResponse("invalid_state", "未连接频道，无法导入", "")
	}

	type counters struct{ Messages, Files, Members, Challenges int }
	count := counters{}

	for _, f := range zipReader.File {
		name := f.Name
		r, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			continue
		}

		switch name {
		case "messages.json":
			var msgs []models.Message
			if err := json.Unmarshal(data, &msgs); err == nil {
				for i := range msgs {
					m := &msgs[i]
					m.ChannelID = channelID
					_ = a.db.SaveMessage(m)
					count.Messages++
				}
			}
		case "files.json":
			var files []models.File
			if err := json.Unmarshal(data, &files); err == nil {
				for i := range files {
					f := &files[i]
					f.ChannelID = channelID
					_ = a.db.SaveFile(f)
					count.Files++
				}
			}
		case "members.json":
			var members []models.Member
			if err := json.Unmarshal(data, &members); err == nil {
				for i := range members {
					mb := &members[i]
					mb.ChannelID = channelID
					_ = a.db.AddMember(mb)
					count.Members++
				}
			}
		case "challenges.json":
			var challenges []models.Challenge
			if err := json.Unmarshal(data, &challenges); err == nil {
				for i := range challenges {
					ch := &challenges[i]
					ch.ChannelID = channelID
					_ = a.db.CreateChallenge(ch)
					count.Challenges++
				}
			}
		default:
			// ignore others
		}
	}

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
	if limit <= 0 {
		limit = 200
	}
	logDir := "./logs"
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return NewSuccessResponse([]map[string]interface{}{})
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ".log" {
			files = append(files, filepath.Join(logDir, e.Name()))
		}
	}
	if len(files) == 0 {
		return NewSuccessResponse([]map[string]interface{}{})
	}
	sort.Strings(files)
	latest := files[len(files)-1]
	f, err := os.Open(latest)
	if err != nil {
		return NewErrorResponse("file_error", "读取日志失败", err.Error())
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lines := make([]string, 0, limit)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > limit {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return NewErrorResponse("scan_error", "解析日志失败", err.Error())
	}
	out := make([]map[string]interface{}, 0, len(lines))
	for _, ln := range lines {
		out = append(out, map[string]interface{}{"line": ln})
	}
	return NewSuccessResponse(out)
}

// ClearLogs 清空日志
func (a *App) ClearLogs() Response {
	logDir := "./logs"
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return NewErrorResponse("fs_error", "读取日志目录失败", err.Error())
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		_ = os.Remove(filepath.Join(logDir, e.Name()))
	}
	return NewSuccessResponse(map[string]interface{}{"message": "日志已清空"})
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
