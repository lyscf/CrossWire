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
export async function sendMessage(content, type = 'text', replyToId = null) {
  const payload = { content, type }
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

export async function getMessage(messageId) {
  const res = await App.GetMessage(messageId)
  return unwrap(res)
}

export async function searchMessages(keyword, limit = 50, offset = 0) {
  const res = await App.SearchMessages({ keyword, limit, offset })
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
  const res = await App.GetMembers()
  return unwrap(res)
}

export async function getMember(memberId) {
  const res = await App.GetMember(memberId)
  return unwrap(res)
}

export async function getMyInfo() {
  const res = await App.GetMyInfo()
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
  const res = await App.UpdateUserProfile(profile)
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

export async function assignChallenge(challengeId, memberId) {
  const res = await App.AssignChallenge({ challenge_id: challengeId, member_id: memberId })
  return unwrap(res)
}

export async function submitFlag(challengeId, flag) {
  const res = await App.SubmitFlag({ challenge_id: challengeId, flag })
  return unwrap(res)
}

export async function getChallengeProgress(challengeId) {
  const res = await App.GetChallengeProgress(challengeId)
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

export async function addHint(challengeId, content, cost) {
  const res = await App.AddHint({ challenge_id: challengeId, content, cost })
  return unwrap(res)
}

export async function unlockHint(challengeId, hintId) {
  const res = await App.UnlockHint(challengeId, hintId)
  return unwrap(res)
}

export async function updateChallengeProgress(challengeId, progress) {
  const res = await App.UpdateChallengeProgress({ challenge_id: challengeId, progress })
  return unwrap(res)
}

// 文件
export async function uploadFile(file) {
  const res = await App.UploadFile(file)
  return unwrap(res)
}

export async function getFiles(limit = 50, offset = 0) {
  const res = await App.GetFiles(limit, offset)
  return unwrap(res)
}

export async function downloadFile(fileId) {
  const res = await App.DownloadFile(fileId)
  return unwrap(res)
}

export async function deleteFile(fileId) {
  const res = await App.DeleteFile(fileId)
  return unwrap(res)
}

export async function getFile(fileId) {
  const res = await App.GetFile(fileId)
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
export async function pinMessage(messageId, reason) {
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

