<template>
  <div class="client-container">
    <div class="client-content">
      <a-card title="加入频道" class="join-card">
        <template #extra>
          <a-button type="link" @click="goBack">
            <ArrowLeftOutlined /> 返回
          </a-button>
        </template>

        <a-tabs v-model:activeKey="activeTab">
          <!-- 自动扫描 -->
          <a-tab-pane key="scan" tab="自动扫描">
            <template #tab>
              <span>
                <RadarChartOutlined />
                自动扫描
              </span>
            </template>

            <div class="scan-section">
              <a-button
                type="primary"
                size="large"
                block
                :loading="scanning"
                @click="startScan"
              >
                <SearchOutlined /> 扫描局域网频道
              </a-button>

              <a-divider>扫描结果</a-divider>

              <a-spin :spinning="scanning">
                <a-list
                  v-if="discoveredServers.length > 0"
                  :data-source="discoveredServers"
                  bordered
                >
                  <template #renderItem="{ item }">
                    <a-list-item>
                      <a-list-item-meta
                        :title="item.channelName"
                        :description="`${item.ip} - ${item.transportMode.toUpperCase()} 模式`"
                      >
                        <template #avatar>
                          <a-avatar style="background-color: #1890ff">
                            <template #icon><CloudServerOutlined /></template>
                          </a-avatar>
                        </template>
                      </a-list-item-meta>
                      <template #actions>
                        <a-button type="link" @click="selectServer(item)">
                          连接
                        </a-button>
                      </template>
                    </a-list-item>
                  </template>
                </a-list>
                <a-empty v-else description="暂无发现频道，请点击扫描" />
              </a-spin>
            </div>
          </a-tab-pane>

          <!-- 手动输入 -->
          <a-tab-pane key="manual" tab="手动输入">
            <template #tab>
              <span>
                <EditOutlined />
                手动输入
              </span>
            </template>

            <a-form
              :model="manualConfig"
              :label-col="{ span: 6 }"
              :wrapper-col="{ span: 18 }"
              @finish="handleManualConnect"
            >
              <a-form-item
                label="服务器地址"
                name="serverAddress"
                :rules="[{ required: true, message: '请输入服务器地址' }]"
              >
                <a-input
                  v-model:value="manualConfig.serverAddress"
                  placeholder="192.168.1.100"
                  :prefix="h(GlobalOutlined)"
                />
              </a-form-item>

              <a-form-item label="端口" name="port">
                <a-input-number
                  v-model:value="manualConfig.port"
                  :min="1024"
                  :max="65535"
                  placeholder="8443"
                  style="width: 100%"
                />
              </a-form-item>

              <a-form-item label="传输模式" name="transportMode">
                <a-select v-model:value="manualConfig.transportMode">
                  <a-select-option value="arp">ARP</a-select-option>
                  <a-select-option value="https">HTTPS</a-select-option>
                  <a-select-option value="mdns">mDNS</a-select-option>
                </a-select>
              </a-form-item>

              <a-form-item :wrapper-col="{ offset: 6, span: 18 }">
                <a-button type="primary" html-type="submit" block size="large">
                  <LoginOutlined /> 连接服务器
                </a-button>
              </a-form-item>
            </a-form>
          </a-tab-pane>

          <!-- 二维码扫描 -->
          <a-tab-pane key="qrcode" tab="扫描二维码">
            <template #tab>
              <span>
                <QrcodeOutlined />
                扫描二维码
              </span>
            </template>

            <div class="qrcode-section">
              <a-empty description="二维码扫描功能开发中" />
            </div>
          </a-tab-pane>
        </a-tabs>
      </a-card>

      <!-- 用户信息表单 (连接后显示) -->
      <a-modal
        v-model:open="showUserInfoModal"
        title="填写个人信息"
        :footer="null"
        :closable="false"
        :maskClosable="false"
      >
        <a-form
          :model="userInfo"
          :label-col="{ span: 6 }"
          :wrapper-col="{ span: 18 }"
          @finish="handleJoinChannel"
        >
          <a-form-item
            label="频道密码"
            name="password"
            :rules="[
              { required: true, message: '请输入频道密码' },
              { min: 6, message: '密码长度至少为6个字符' }
            ]"
          >
            <a-input-password
              v-model:value="userInfo.password"
              placeholder="至少6个字符"
            />
          </a-form-item>

          <a-form-item
            label="昵称"
            name="nickname"
            :rules="[{ required: true, message: '请输入昵称' }]"
          >
            <a-input
              v-model:value="userInfo.nickname"
              placeholder="请输入昵称"
              :maxlength="20"
            />
          </a-form-item>

          <a-form-item label="角色" name="role">
            <a-select v-model:value="userInfo.role">
              <a-select-option value="队长">队长</a-select-option>
              <a-select-option value="队员">队员</a-select-option>
              <a-select-option value="替补">替补</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="技能标签" name="skills">
            <a-select
              v-model:value="userInfo.skills"
              mode="tags"
              placeholder="选择或输入技能标签"
            >
              <a-select-option value="Web">Web</a-select-option>
              <a-select-option value="Pwn">Pwn</a-select-option>
              <a-select-option value="Reverse">Reverse</a-select-option>
              <a-select-option value="Crypto">Crypto</a-select-option>
              <a-select-option value="Misc">Misc</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="个人简介" name="bio">
            <a-textarea
              v-model:value="userInfo.bio"
              :rows="3"
              placeholder="简单介绍一下自己（可选）"
              :maxlength="200"
            />
          </a-form-item>

          <a-form-item :wrapper-col="{ offset: 6, span: 18 }">
            <a-space>
              <a-button type="primary" html-type="submit" :loading="joining">
                加入频道
              </a-button>
              <a-button @click="showUserInfoModal = false">
                取消
              </a-button>
            </a-space>
          </a-form-item>
        </a-form>
      </a-modal>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, h } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  ArrowLeftOutlined,
  RadarChartOutlined,
  SearchOutlined,
  EditOutlined,
  QrcodeOutlined,
  GlobalOutlined,
  LoginOutlined,
  CloudServerOutlined
} from '@ant-design/icons-vue'

const router = useRouter()
const activeTab = ref('scan')
const scanning = ref(false)
const joining = ref(false)
const showUserInfoModal = ref(false)
const selectedServer = ref(null)

const discoveredServers = ref([])

const manualConfig = reactive({
  serverAddress: '',
  port: 8443,
  transportMode: 'https'
})

const userInfo = reactive({
  password: '',
  nickname: '',
  role: '队员',
  skills: [],
  bio: ''
})

const goBack = () => {
  router.push('/')
}

import { discoverServers, startClient } from '@/api/app'

const startScan = async () => {
  scanning.value = true
  try {
    const servers = await discoverServers(3)
    discoveredServers.value = (servers || []).map(s => ({
      channelName: s.channel_name || s.ChannelName || '未知频道',
      ip: s.ip || s.IP || s.address || '',
      port: s.port || 8443,
      transportMode: (s.transport_mode || s.TransportMode || 'https').toLowerCase(),
      members: s.members || s.MemberCount || 0
    }))
    message.success(`发现 ${discoveredServers.value.length} 个频道`)
  } catch (error) {
    message.error('扫描失败: ' + (error.message || ''))
  } finally {
    scanning.value = false
  }
}

const selectServer = (server) => {
  selectedServer.value = server
  showUserInfoModal.value = true
}

const handleManualConnect = () => {
  selectedServer.value = {
    ...manualConfig,
    channelName: '未知频道'
  }
  showUserInfoModal.value = true
}

const handleJoinChannel = async () => {
  joining.value = true
  try {
    await startClient({
      password: userInfo.password,
      transport_mode: manualConfig.transportMode,
      network_interface: '',
      server_address: selectedServer.value?.ip || manualConfig.serverAddress,
      port: selectedServer.value?.port || manualConfig.port,
      nickname: userInfo.nickname,
      avatar: '',
      auto_reconnect: true
    })
    message.success('成功加入频道！')
    router.push('/chat')
  } catch (error) {
    message.error('加入频道失败: ' + (error.message || ''))
  } finally {
    joining.value = false
  }
}
</script>

<style scoped>
.client-container {
  width: 100%;
  height: 100vh;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  background-color: #f5f5f5;
  padding: var(--spacing-xl) var(--spacing-lg);
  overflow-y: auto;
}

.client-content {
  width: 100%;
  max-width: 800px;
}

.join-card {
  background-color: white;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.scan-section {
  padding: var(--spacing-lg) 0;
}

.qrcode-section {
  padding: var(--spacing-xl) 0;
  text-align: center;
}
</style>

