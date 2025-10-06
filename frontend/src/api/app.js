// 前端 API 封装：统一处理 Wails 返回的 Response 结构
import * as App from '../wailsjs/go/app/App'

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

export async function getMessages(limit = 50, offset = 0) {
  const res = await App.GetMessages(limit, offset)
  return unwrap(res)
}


