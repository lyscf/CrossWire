<template>
  <a-modal
    v-model:open="visible"
    title="设置"
    width="700px"
    :footer="null"
  >
    <a-tabs v-model:activeKey="activeTab" tab-position="left" class="settings-tabs">
      <!-- 通用设置 -->
      <a-tab-pane key="general" tab="通用">
        <template #tab>
          <SettingOutlined /> 通用
        </template>
        
        <div class="settings-section">
          <h4 class="section-title">界面设置</h4>
          
          <a-form layout="vertical">
            <a-form-item label="主题模式">
              <a-radio-group v-model:value="settings.theme">
                <a-radio value="light">浅色</a-radio>
                <a-radio value="dark" disabled>深色（即将推出）</a-radio>
                <a-radio value="auto" disabled>跟随系统</a-radio>
              </a-radio-group>
            </a-form-item>

            <a-form-item label="语言">
              <a-select v-model:value="settings.language" style="width: 200px">
                <a-select-option value="zh-CN">简体中文</a-select-option>
                <a-select-option value="en-US" disabled>English</a-select-option>
              </a-select>
            </a-form-item>

            <a-form-item label="字体大小">
              <a-slider
                v-model:value="settings.fontSize"
                :min="12"
                :max="18"
                :marks="{ 12: '小', 14: '标准', 16: '大', 18: '超大' }"
              />
            </a-form-item>

            <a-form-item label="消息密度">
              <a-radio-group v-model:value="settings.messageDensity">
                <a-radio value="comfortable">舒适</a-radio>
                <a-radio value="compact">紧凑</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-form>
        </div>
      </a-tab-pane>

      <!-- 通知设置 -->
      <a-tab-pane key="notifications" tab="通知">
        <template #tab>
          <BellOutlined /> 通知
        </template>
        
        <div class="settings-section">
          <h4 class="section-title">通知设置</h4>
          
          <a-form layout="vertical">
            <a-form-item label="桌面通知">
              <a-switch v-model:checked="settings.notifications.desktop" />
              <p class="help-text">允许显示桌面通知</p>
            </a-form-item>

            <a-form-item label="声音提示">
              <a-switch v-model:checked="settings.notifications.sound" />
              <p class="help-text">收到新消息时播放提示音</p>
            </a-form-item>

            <a-divider />

            <a-form-item label="通知类型">
              <a-checkbox-group v-model:value="settings.notifications.types">
                <a-checkbox value="mention">@提及我时</a-checkbox>
                <a-checkbox value="message">所有新消息</a-checkbox>
                <a-checkbox value="challenge">题目分配</a-checkbox>
                <a-checkbox value="flag">Flag提交结果</a-checkbox>
                <a-checkbox value="system">系统通知</a-checkbox>
              </a-checkbox-group>
            </a-form-item>

            <a-form-item label="免打扰时间">
              <a-time-range-picker
                v-model:value="settings.notifications.dndTime"
                format="HH:mm"
                placeholder="['开始时间', '结束时间']"
              />
              <p class="help-text">在此时间段内不显示通知</p>
            </a-form-item>
          </a-form>
        </div>
      </a-tab-pane>

      <!-- 网络设置 -->
      <a-tab-pane key="network" tab="网络">
        <template #tab>
          <GlobalOutlined /> 网络
        </template>
        
        <div class="settings-section">
          <h4 class="section-title">连接设置</h4>
          
          <a-form layout="vertical">
            <a-form-item label="传输模式">
              <a-radio-group v-model:value="settings.network.transport">
                <a-radio value="arp">ARP 广播</a-radio>
                <a-radio value="https">HTTPS</a-radio>
                <a-radio value="mdns">mDNS</a-radio>
              </a-radio-group>
              <p class="help-text">选择网络传输方式</p>
            </a-form-item>

            <a-form-item label="自动重连">
              <a-switch v-model:checked="settings.network.autoReconnect" />
              <p class="help-text">连接断开时自动尝试重连</p>
            </a-form-item>

            <a-form-item label="心跳间隔">
              <a-input-number
                v-model:value="settings.network.heartbeatInterval"
                :min="10"
                :max="300"
                :step="10"
                suffix="秒"
                style="width: 200px"
              />
              <p class="help-text">发送心跳包的时间间隔</p>
            </a-form-item>

            <a-form-item label="网络接口">
              <a-select
                v-model:value="settings.network.interface"
                style="width: 100%"
                placeholder="选择网络接口"
              >
                <a-select-option value="auto">自动选择</a-select-option>
                <a-select-option value="eth0">eth0</a-select-option>
                <a-select-option value="wlan0">wlan0</a-select-option>
              </a-select>
            </a-form-item>
          </a-form>
        </div>
      </a-tab-pane>

      <!-- 快捷键设置 -->
      <a-tab-pane key="shortcuts" tab="快捷键">
        <template #tab>
          <ThunderboltOutlined /> 快捷键
        </template>
        
        <div class="settings-section">
          <h4 class="section-title">键盘快捷键</h4>
          
          <a-list :data-source="shortcuts" size="small">
            <template #renderItem="{ item }">
              <a-list-item>
                <a-list-item-meta>
                  <template #title>
                    {{ item.name }}
                  </template>
                  <template #description>
                    {{ item.description }}
                  </template>
                </a-list-item-meta>
                <template #actions>
                  <a-tag>{{ item.key }}</a-tag>
                </template>
              </a-list-item>
            </template>
          </a-list>
        </div>
      </a-tab-pane>

      <!-- 高级设置 -->
      <a-tab-pane key="advanced" tab="高级">
        <template #tab>
          <ToolOutlined /> 高级
        </template>
        
        <div class="settings-section">
          <h4 class="section-title">高级选项</h4>
          
          <a-form layout="vertical">
            <a-form-item label="开发者模式">
              <a-switch v-model:checked="settings.advanced.devMode" />
              <p class="help-text">启用开发者工具和调试功能</p>
            </a-form-item>

            <a-form-item label="日志级别">
              <a-select v-model:value="settings.advanced.logLevel" style="width: 200px">
                <a-select-option value="debug">Debug</a-select-option>
                <a-select-option value="info">Info</a-select-option>
                <a-select-option value="warn">Warn</a-select-option>
                <a-select-option value="error">Error</a-select-option>
              </a-select>
            </a-form-item>

            <a-form-item label="缓存管理">
              <a-space direction="vertical" style="width: 100%">
                <div class="cache-info">
                  <span>当前缓存大小: 45.6 MB</span>
                </div>
                <a-button danger @click="clearCache">
                  <DeleteOutlined /> 清除缓存
                </a-button>
              </a-space>
            </a-form-item>

            <a-form-item label="数据导出">
              <a-space>
                <a-button @click="exportData">
                  <DownloadOutlined /> 导出聊天记录
                </a-button>
                <a-button @click="exportSettings">
                  <DownloadOutlined /> 导出设置
                </a-button>
              </a-space>
            </a-form-item>

            <a-form-item label="危险操作">
              <a-button danger @click="resetSettings">
                <RedoOutlined /> 恢复默认设置
              </a-button>
            </a-form-item>
          </a-form>
        </div>
      </a-tab-pane>

      <!-- 关于 -->
      <a-tab-pane key="about" tab="关于">
        <template #tab>
          <InfoCircleOutlined /> 关于
        </template>
        
        <div class="settings-section">
          <div class="about-section">
            <h2>CrossWire</h2>
            <p class="version">版本 1.0.0</p>
            <p class="description">
              安全的 CTF 团队协作通讯工具，支持端到端加密、题目管理和实时协作。
            </p>

            <a-divider />

            <div class="info-item">
              <strong>开源协议：</strong>
              <span>MIT License</span>
            </div>

            <div class="info-item">
              <strong>项目地址：</strong>
              <a href="https://github.com/crosswire/crosswire" target="_blank">
                GitHub
              </a>
            </div>

            <div class="info-item">
              <strong>技术栈：</strong>
              <a-space wrap>
                <a-tag>Vue 3</a-tag>
                <a-tag>Ant Design Vue</a-tag>
                <a-tag>Wails 2</a-tag>
                <a-tag>Go</a-tag>
              </a-space>
            </div>

            <a-divider />

            <a-button type="primary" block @click="checkUpdate">
              <UploadOutlined /> 检查更新
            </a-button>
          </div>
        </div>
      </a-tab-pane>
    </a-tabs>

    <div class="settings-footer">
      <a-button type="primary" @click="saveSettings">
        <SaveOutlined /> 保存设置
      </a-button>
      <a-button @click="visible = false">取消</a-button>
    </div>
  </a-modal>
</template>

<script setup>
import { ref, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import {
  SettingOutlined,
  BellOutlined,
  GlobalOutlined,
  ThunderboltOutlined,
  ToolOutlined,
  InfoCircleOutlined,
  SaveOutlined,
  DeleteOutlined,
  DownloadOutlined,
  RedoOutlined,
  UploadOutlined
} from '@ant-design/icons-vue'

const props = defineProps({
  open: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:open', 'save'])

const visible = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

const activeTab = ref('general')

const settings = ref({
  theme: 'light',
  language: 'zh-CN',
  fontSize: 14,
  messageDensity: 'comfortable',
  notifications: {
    desktop: true,
    sound: true,
    types: ['mention', 'challenge', 'flag'],
    dndTime: null
  },
  network: {
    transport: 'arp',
    autoReconnect: true,
    heartbeatInterval: 30,
    interface: 'auto'
  },
  advanced: {
    devMode: false,
    logLevel: 'info'
  }
})

const shortcuts = [
  { name: '发送消息', description: '快速发送消息', key: 'Ctrl+Enter' },
  { name: '全局搜索', description: '打开搜索框', key: 'Ctrl+K' },
  { name: '@提及', description: '提及成员', key: '@ + 用户名' },
  { name: '插入代码', description: '插入代码块', key: 'Ctrl+Shift+C' },
  { name: '上传文件', description: '打开文件选择', key: 'Ctrl+U' },
  { name: '表情选择', description: '打开表情面板', key: 'Ctrl+E' }
]

const saveSettings = () => {
  emit('save', settings.value)
  message.success('设置已保存')
  visible.value = false
}

const clearCache = () => {
  Modal.confirm({
    title: '确认清除缓存？',
    content: '这将清除所有本地缓存数据，但不会删除消息记录。',
    okText: '确认',
    cancelText: '取消',
    onOk() {
      message.success('缓存已清除')
    }
  })
}

const exportData = () => {
  message.info('导出功能开发中...')
}

const exportSettings = () => {
  const dataStr = JSON.stringify(settings.value, null, 2)
  const blob = new Blob([dataStr], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'crosswire-settings.json'
  link.click()
  URL.revokeObjectURL(url)
  message.success('设置已导出')
}

const resetSettings = () => {
  Modal.confirm({
    title: '确认恢复默认设置？',
    content: '这将重置所有设置为默认值，此操作不可撤销。',
    okText: '确认',
    cancelText: '取消',
    okType: 'danger',
    onOk() {
      // 重置为默认值
      message.success('已恢复默认设置')
    }
  })
}

const checkUpdate = () => {
  message.info('当前已是最新版本')
}
</script>

<style scoped>
.settings-tabs {
  min-height: 500px;
}

.settings-section {
  padding: 0 16px;
}

.section-title {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
}

.help-text {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: rgba(0, 0, 0, 0.45);
}

.cache-info {
  padding: 8px 12px;
  background-color: #fafafa;
  border-radius: 4px;
  font-size: 13px;
}

.about-section {
  padding: 24px;
  text-align: center;
}

.about-section h2 {
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 700;
}

.version {
  margin: 0 0 16px 0;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.45);
}

.description {
  margin: 0 0 24px 0;
  color: rgba(0, 0, 0, 0.65);
  line-height: 1.6;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  text-align: left;
}

.settings-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 16px 24px 0;
  border-top: 1px solid #f0f0f0;
}
</style>

