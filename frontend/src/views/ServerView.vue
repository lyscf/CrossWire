<template>
  <div class="server-container">
    <div class="server-content">
      <a-card title="创建频道" class="config-card">
        <template #extra>
          <a-button type="link" @click="goBack">
            <ArrowLeftOutlined /> 返回
          </a-button>
        </template>

        <a-form
          :model="serverConfig"
          :label-col="{ span: 6 }"
          :wrapper-col="{ span: 18 }"
          @finish="handleStartServer"
        >
          <!-- 频道名称 -->
          <a-form-item
            label="频道名称"
            name="channelName"
            :rules="[{ required: true, message: '请输入频道名称' }]"
          >
            <a-input
              v-model:value="serverConfig.channelName"
              placeholder="CTF-Team-2025"
              :prefix="h(TeamOutlined)"
            />
          </a-form-item>

          <!-- 频道密码 -->
          <a-form-item
            label="频道密码"
            name="password"
            :rules="[{ required: true, message: '请输入频道密码' }]"
          >
            <a-input-password
              v-model:value="serverConfig.password"
              placeholder="设置加入密码"
              :prefix="h(LockOutlined)"
            >
              <template #suffix>
                <a-tooltip title="生成随机密码">
                  <ReloadOutlined @click="generatePassword" />
                </a-tooltip>
              </template>
            </a-input-password>
          </a-form-item>

          <!-- 传输模式 -->
          <a-form-item label="传输模式" name="transportMode">
            <a-radio-group v-model:value="serverConfig.transportMode">
              <a-radio-button value="arp">
                <ThunderboltOutlined /> ARP (推荐)
              </a-radio-button>
              <a-radio-button value="https">
                <GlobalOutlined /> HTTPS
              </a-radio-button>
              <a-radio-button value="mdns">
                <ApiOutlined /> mDNS
              </a-radio-button>
            </a-radio-group>
          </a-form-item>

          <!-- HTTPS 端口 -->
          <a-form-item
            v-if="serverConfig.transportMode === 'https'"
            label="HTTPS 端口"
            name="port"
          >
            <a-input-number
              v-model:value="serverConfig.port"
              :min="1024"
              :max="65535"
              placeholder="8443"
              style="width: 100%"
            />
          </a-form-item>

          <!-- 网络接口 (ARP) -->
          <a-form-item
            v-if="serverConfig.transportMode === 'arp'"
            label="网络接口"
            name="interface"
          >
            <a-select
              v-model:value="serverConfig.interface"
              placeholder="选择网卡"
              :loading="loadingInterfaces"
            >
              <a-select-option
                v-for="iface in interfaces"
                :key="iface.name"
                :value="iface.name"
              >
                {{ iface.name }} - {{ iface.ip }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <!-- 最大成员数 -->
          <a-form-item label="最大成员数" name="maxMembers">
            <a-slider
              v-model:value="serverConfig.maxMembers"
              :min="2"
              :max="100"
              :marks="{ 2: '2', 25: '25', 50: '50', 100: '100' }"
            />
          </a-form-item>

          <!-- 高级选项 -->
          <a-collapse>
            <a-collapse-panel key="1" header="高级选项">
              <a-form-item label="历史保留" name="historyRetention">
                <a-input-number
                  v-model:value="serverConfig.historyRetention"
                  :min="1"
                  :max="365"
                  :addon-after="'天'"
                  style="width: 100%"
                />
              </a-form-item>

              <a-form-item label="文件传输" name="allowFileUpload">
                <a-switch v-model:checked="serverConfig.allowFileUpload" />
              </a-form-item>

              <a-form-item label="最大文件大小" name="maxFileSize">
                <a-input-number
                  v-model:value="serverConfig.maxFileSize"
                  :min="1"
                  :max="1024"
                  :addon-after="'MB'"
                  :disabled="!serverConfig.allowFileUpload"
                  style="width: 100%"
                />
              </a-form-item>
            </a-collapse-panel>
          </a-collapse>

          <!-- 操作按钮 -->
          <a-form-item :wrapper-col="{ offset: 6, span: 18 }">
            <a-space>
              <a-button
                type="primary"
                html-type="submit"
                size="large"
                :loading="loading"
              >
                <CloudServerOutlined /> 启动服务端
              </a-button>
              <a-button size="large" @click="resetForm">
                重置
              </a-button>
            </a-space>
          </a-form-item>
        </a-form>
      </a-card>

      <!-- 提示信息 -->
      <a-card class="tips-card" title="提示">
        <a-space direction="vertical" :size="12">
          <div>
            <InfoCircleOutlined style="color: #1890ff" />
            <span class="tip-text">
              <strong>ARP 模式：</strong>需要管理员权限，提供 1-3ms 超低延迟和 50-100MB/s 传输速度
            </span>
          </div>
          <div>
            <InfoCircleOutlined style="color: #1890ff" />
            <span class="tip-text">
              <strong>HTTPS 模式：</strong>标准网络模式，支持跨网段通信，需要开放指定端口
            </span>
          </div>
          <div>
            <InfoCircleOutlined style="color: #1890ff" />
            <span class="tip-text">
              <strong>mDNS 模式：</strong>适用于极端受限网络，速度较慢但隐蔽性好
            </span>
          </div>
        </a-space>
      </a-card>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, h, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  ArrowLeftOutlined,
  TeamOutlined,
  LockOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
  GlobalOutlined,
  ApiOutlined,
  CloudServerOutlined,
  InfoCircleOutlined
} from '@ant-design/icons-vue'

const router = useRouter()
const loading = ref(false)
const loadingInterfaces = ref(false)
const interfaces = ref([])

const serverConfig = reactive({
  channelName: `CTF-Team-${new Date().getFullYear()}`,
  password: '',
  transportMode: 'arp',
  port: 8443,
  interface: '',
  maxMembers: 50,
  historyRetention: 7,
  allowFileUpload: true,
  maxFileSize: 100
})

const goBack = () => {
  router.push('/')
}

const generatePassword = () => {
  const chars = 'ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678'
  let password = ''
  for (let i = 0; i < 12; i++) {
    password += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  serverConfig.password = password
  message.success('已生成随机密码')
}

const loadNetworkInterfaces = async () => {
  loadingInterfaces.value = true
  try {
    // TODO: 调用 Wails API 获取网络接口列表
    // const ifaces = await GetNetworkInterfaces()
    // 模拟数据
    interfaces.value = [
      { name: 'eth0', ip: '192.168.1.100' },
      { name: 'wlan0', ip: '192.168.1.101' }
    ]
    if (interfaces.value.length > 0) {
      serverConfig.interface = interfaces.value[0].name
    }
  } catch (error) {
    message.error('获取网络接口失败')
  } finally {
    loadingInterfaces.value = false
  }
}

const handleStartServer = async () => {
  loading.value = true
  try {
    // TODO: 调用 Wails API 启动服务端
    // await StartServerMode(serverConfig)
    console.log('Starting server with config:', serverConfig)
    
    // 模拟启动
    await new Promise(resolve => setTimeout(resolve, 1500))
    
    message.success('服务端启动成功！')
    router.push('/chat')
  } catch (error) {
    message.error('启动服务端失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  serverConfig.channelName = `CTF-Team-${new Date().getFullYear()}`
  serverConfig.password = ''
  serverConfig.transportMode = 'arp'
  serverConfig.port = 8443
  serverConfig.maxMembers = 50
  serverConfig.historyRetention = 7
  serverConfig.allowFileUpload = true
  serverConfig.maxFileSize = 100
}

onMounted(() => {
  loadNetworkInterfaces()
})
</script>

<style scoped>
.server-container {
  width: 100%;
  height: 100vh;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  background-color: #f5f5f5;
  padding: 40px 20px;
  overflow-y: auto;
}

.server-content {
  width: 100%;
  max-width: 900px;
}

.config-card {
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.tips-card {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.tip-text {
  margin-left: 8px;
}
</style>

