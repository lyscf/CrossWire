// 前端 API 封装：统一处理 Wails 返回的 Response 结构
import * as App from '../../wailsjs/go/app/App'

function unwrap(response) {
  // 兼容后端 Response 通用结构 { success, data, error }
  if (response && typeof response === 'object' && 'success' in response) {
    if (response.success) return response.data
    const err = response.error || { message: 'Unknown error' }
    const e = new Error(err.message || 'Request failed')
    e.code = err.code
    e.details = err.details
    throw e
  }
  // 原始标量/对象
  return response
}

// 系统
export async function getNetworkInterfaces() {
  const res = await App.GetNetworkInterfaces()
  return unwrap(res)
}

// 服务端
export async function startServer(config) {
  const res = await App.StartServerMode(config)
  return unwrap(res)
}

export async function getServerStatus() {
  const res = await App.GetServerStatus()
  return unwrap(res)
}

// 客户端
export async function startClient(config) {
  const res = await App.StartClientMode(config)
  return unwrap(res)
}

export async function discoverServers(timeoutSec = 5) {
  const res = await App.DiscoverServers(timeoutSec)
  return unwrap(res)
}

// 消息
export async function sendMessage(content, type = 'text', channelId = null, replyToId = null) {
  const payload = { content, type }
  if (channelId) payload.channel_id = channelId
  if (replyToId) payload.reply_to_id = replyToId
  const res = await App.SendMessage(payload)
  return unwrap(res)
}

export async function sendCodeMessage(code, language, description = '') {
  const res = await App.SendCodeMessage({ code, language, description })
  return unwrap(res)
}

export async function getMessages(limit = 50, offset = 0) {
  const res = await App.GetMessages(limit, offset)
  return unwrap(res)
}

export async function getMessagesByChannel(channelId, limit = 50, offset = 0) {
  const res = await App.GetMessagesByChannel(channelId, limit, offset)
  return unwrap(res)
}

export async function getMessage(messageId) {
  const res = await App.GetMessage(messageId)
  return unwrap(res)
}

export async function searchMessages(query, limit = 50, offset = 0) {
  const res = await App.SearchMessages({ query, limit, offset })
  return unwrap(res)
}

// 新增：按时间范围/标签/@我/统计
export async function getMessagesByTimeRange(startSec = 0, endSec = 0, limit = 100, offset = 0) {
  const res = await App.GetMessagesByTimeRange(startSec, endSec, limit, offset)
  return unwrap(res)
}

export async function getMessagesByTag(tag, limit = 50, offset = 0) {
  const res = await App.GetMessagesByTag(tag, limit, offset)
  return unwrap(res)
}

export async function getMentionedMessages(limit = 50, offset = 0) {
  const res = await App.GetMentionedMessages(limit, offset)
  return unwrap(res)
}

export async function getMessageStats(fromSec = 0, toSec = 0) {
  const res = await App.GetMessageStats(fromSec, toSec)
  return unwrap(res)
}

export async function deleteMessage(messageId) {
  const res = await App.DeleteMessage(messageId)
  return unwrap(res)
}

export async function reactToMessage(messageId, emoji) {
  const res = await App.ReactToMessage(messageId, emoji)
  return unwrap(res)
}

export async function removeReaction(messageId, emoji) {
  const res = await App.RemoveReaction(messageId, emoji)
  return unwrap(res)
}

// 子频道
export async function getSubChannels() {
  const res = await App.GetSubChannels()
  return unwrap(res)
}

// 成员
export async function getMembers() {
  console.log('[API] Calling GetMembers...')
  const res = await App.GetMembers()
  console.log('[API] GetMembers response:', res)
  return unwrap(res)
}

export async function getMember(memberId) {
  console.log('[API] Calling GetMember with ID:', memberId)
  const res = await App.GetMember(memberId)
  console.log('[API] GetMember response:', res)
  return unwrap(res)
}

export async function getMyInfo() {
  console.log('[API] Calling GetMyInfo...')
  const res = await App.GetMyInfo()
  console.log('[API] GetMyInfo response:', res)
  return unwrap(res)
}

export async function updateMyStatus(status) {
  const res = await App.UpdateMyStatus(status)
  return unwrap(res)
}

export async function updateMyProfile(nickname, avatar) {
  const res = await App.UpdateMyProfile(nickname, avatar)
  return unwrap(res)
}

export async function updateMemberRole(memberId, role) {
  const res = await App.UpdateMemberRole(memberId, role)
  return unwrap(res)
}

// 用户配置
export async function getUserProfile() {
  const res = await App.GetUserProfile()
  return unwrap(res)
}

export async function updateUserProfile(profile) {
  console.log('[API] Calling UpdateUserProfile with:', profile)
  const res = await App.UpdateUserProfile(profile)
  console.log('[API] UpdateUserProfile response:', res)
  return unwrap(res)
}

// 题目
export async function getChallenges() {
  const res = await App.GetChallenges()
  return unwrap(res)
}

export async function getChallenge(challengeId) {
  const res = await App.GetChallenge(challengeId)
  return unwrap(res)
}

export async function createChallenge(challenge) {
  const res = await App.CreateChallenge(challenge)
  return unwrap(res)
}

export async function updateChallenge(challenge) {
  const res = await App.UpdateChallenge(challenge)
  return unwrap(res)
}

export async function deleteChallenge(challengeId) {
  const res = await App.DeleteChallenge(challengeId)
  return unwrap(res)
}

export async function assignChallenge(challengeId, memberIds) {
  // 兼容单个/数组，Wails端签名为 (string, []string)
  const ids = Array.isArray(memberIds) ? memberIds : [memberIds]
  const res = await App.AssignChallenge(challengeId, ids)
  return unwrap(res)
}

export async function submitFlag(challengeId, flag) {
  const res = await App.SubmitFlag({ challenge_id: challengeId, flag })
  return unwrap(res)
}

export async function getChallengeProgress(challengeId, memberId = null) {
  let effectiveMemberId = memberId
  if (!effectiveMemberId) {
    try {
      const me = await App.GetMyInfo()
      const data = unwrap(me)
      effectiveMemberId = data?.id || data?.ID || 'server'
    } catch (e) {
      effectiveMemberId = 'server'
    }
  }
  const res = await App.GetChallengeProgress(challengeId, effectiveMemberId)
  return unwrap(res)
}

export async function getChallengeSubmissions(challengeId) {
  const res = await App.GetChallengeSubmissions(challengeId)
  return unwrap(res)
}

export async function getChallengeStats() {
  const res = await App.GetChallengeStats()
  return unwrap(res)
}

export async function getLeaderboard() {
  const res = await App.GetLeaderboard()
  return unwrap(res)
}

export async function updateChallengeProgress(challengeId, progress) {
  const res = await App.UpdateChallengeProgress({ challenge_id: challengeId, progress })
  return unwrap(res)
}

// 文件
export async function uploadFile(input) {
  // 兼容字符串/对象两种传参，规范化为 { file_path }
  const payload = typeof input === 'string' ? { file_path: input } : input
  const res = await App.UploadFile(payload)
  return unwrap(res)
}

export async function getFiles(limit = 50, offset = 0) {
  const res = await App.GetFiles(limit, offset)
  return unwrap(res)
}

export async function downloadFile(fileId, savePath = null, defaultFilename = null) {
  if (!savePath) {
    const title = '选择保存位置'
    const def = defaultFilename || `file_${fileId}`
    const sp = await App.SaveFileDialog(title, def)
    savePath = unwrap(sp)
    if (!savePath) throw new Error('用户取消下载')
  }
  const res = await App.DownloadFile({ file_id: fileId, save_path: savePath })
  const data = unwrap(res)
  return { ...data, savePath }
}

export async function deleteFile(fileId) {
  const res = await App.DeleteFile(fileId)
  return unwrap(res)
}

export async function getFile(fileId) {
  const res = await App.GetFile(fileId)
  return unwrap(res)
}

export async function getFileContent(fileId) {
  const res = await App.GetFileContent(fileId)
  return unwrap(res)
}

export async function getFileProgress(fileId) {
  const res = await App.GetFileProgress(fileId)
  return unwrap(res)
}

export async function cancelUpload(fileId) {
  const res = await App.CancelUpload(fileId)
  return unwrap(res)
}

export async function cancelDownload(fileId) {
  const res = await App.CancelDownload(fileId)
  return unwrap(res)
}

export async function getFileTransferStats() {
  const res = await App.GetFileTransferStats()
  return unwrap(res)
}

// 成员管理
export async function kickMember(memberId) {
  const res = await App.KickMember(memberId)
  return unwrap(res)
}

export async function muteMember(memberId, duration) {
  const res = await App.MuteMember({ member_id: memberId, duration })
  return unwrap(res)
}

export async function unmuteMember(memberId) {
  const res = await App.UnmuteMember(memberId)
  return unwrap(res)
}

export async function banMember(memberId, reason) {
  const res = await App.BanMember({ member_id: memberId, reason })
  return unwrap(res)
}

export async function unbanMember(memberId) {
  const res = await App.UnbanMember(memberId)
  return unwrap(res)
}

// 频道管理
export async function pinMessage(messageId, reason = '') {
  const res = await App.PinMessage({ message_id: messageId, reason })
  return unwrap(res)
}

export async function unpinMessage(messageId) {
  const res = await App.UnpinMessage(messageId)
  return unwrap(res)
}

export async function getPinnedMessages() {
  const res = await App.GetPinnedMessages()
  return unwrap(res)
}

// 系统功能
export async function getNetworkStats() {
  const res = await App.GetNetworkStats()
  return unwrap(res)
}

export async function testConnection(serverAddress, mode, timeout = 5) {
  const res = await App.TestConnection(serverAddress, mode, timeout)
  return unwrap(res)
}

export async function getRecentChannels() {
  const res = await App.GetRecentChannels()
  return unwrap(res)
}

// HTTPS: 获取服务器频道信息 (/info)
export async function fetchHttpsInfo(server, port, skipTLSVerify = true, timeoutSec = 5) {
  const res = await App.FetchHTTPSInfo(server, port, skipTLSVerify, timeoutSec)
  return unwrap(res)
}

export async function exportData(exportPath, options) {
  const res = await App.ExportData(exportPath, options)
  return unwrap(res)
}

export async function importData(importPath) {
  const res = await App.ImportData(importPath)
  return unwrap(res)
}

export async function getLogs(limit = 100) {
  const res = await App.GetLogs(limit)
  return unwrap(res)
}

export async function clearLogs() {
  const res = await App.ClearLogs()
  return unwrap(res)
}

// 文件对话框
export async function selectFile(title, filter) {
  const res = await App.SelectFile(title, filter)
  return unwrap(res)
}

export async function selectDirectory(title) {
  const res = await App.SelectDirectory(title)
  return unwrap(res)
}

export async function saveFileDialog(title, defaultFilename) {
  const res = await App.SaveFileDialog(title, defaultFilename)
  return unwrap(res)
}

// 客户端模式
export async function getClientStatus() {
  const res = await App.GetClientStatus()
  return unwrap(res)
}

export async function stopClient() {
  const res = await App.StopClientMode()
  return unwrap(res)
}

export async function stopServer() {
  const res = await App.StopServerMode()
  return unwrap(res)
}

export async function getDiscoveredServers() {
  const res = await App.GetDiscoveredServers()
  return unwrap(res)
}

