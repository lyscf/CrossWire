<template>
  <a-layout-header class="chat-header">
    <Toolbar :gap="12" :height="64" padding-x="16px" background="#fff" :border="true">
      <template #left>
        <h3 class="current-channel">{{ currentChannelLabel }}</h3>
        <a-tag v-if="connected" color="green">
          <CheckCircleOutlined /> 已连接
        </a-tag>
      </template>

      <template #center>
        <div class="header-center">
          <SearchBar />
        </div>
      </template>

      <template #right>
        <a-space :size="8">
          <NotificationCenter />
          <a-tooltip title="文件管理">
            <a-button type="text" :icon="h(FileOutlined)" @click="$emit('open-file-manager')" />
          </a-tooltip>
          <a-tooltip title="成员列表">
            <a-button type="text" :icon="h(TeamOutlined)" @click="$emit('open-member-drawer')" />
          </a-tooltip>
          <a-dropdown>
            <a-avatar style="cursor: pointer; background-color: #1890ff" @click="$emit('open-user-profile')">
              {{ (currentUser?.name || 'U').charAt(0).toUpperCase() }}
            </a-avatar>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="$emit('open-user-profile')">
                  <UserOutlined /> 个人资料
                </a-menu-item>
                <a-menu-item @click="$emit('open-settings')">
                  <SettingOutlined /> 设置
                </a-menu-item>
                <a-menu-divider />
                <a-menu-item danger @click="$emit('disconnect')">
                  <PoweroffOutlined /> 断开连接
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </a-space>
      </template>
    </Toolbar>
  </a-layout-header>
  
</template>

<script setup>
import { h } from 'vue'
import Toolbar from '@/components/Common/Toolbar.vue'
import SearchBar from '@/components/SearchBar.vue'
import NotificationCenter from '@/components/NotificationCenter.vue'
import {
  CheckCircleOutlined,
  FileOutlined,
  TeamOutlined,
  UserOutlined,
  SettingOutlined,
  PoweroffOutlined
} from '@ant-design/icons-vue'

const props = defineProps({
  currentChannelLabel: {
    type: String,
    default: '主频道'
  },
  currentUser: {
    type: Object,
    default: () => ({ name: 'User' })
  },
  connected: {
    type: Boolean,
    default: true
  }
})

defineEmits([
  'open-file-manager',
  'open-member-drawer',
  'open-user-profile',
  'open-settings',
  'disconnect'
])
</script>

<style scoped>
.header-center {
  flex: 1;
  max-width: 600px;
  min-width: 200px;
  padding: 0 16px;
}

.current-channel {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (max-width: 1200px) {
  .header-center {
    max-width: 400px;
    padding: 0 12px;
  }
}

@media (max-width: 992px) {
  .header-center {
    max-width: 300px;
    min-width: 150px;
    padding: 0 8px;
  }
  .current-channel {
    font-size: 16px;
  }
}

@media (max-width: 768px) {
  .header-center {
    position: absolute;
    left: 50%;
    transform: translateX(-50%);
    max-width: 250px;
    min-width: 120px;
    padding: 0;
    z-index: 10;
  }
}

@media (max-width: 576px) {
  .header-center {
    display: none;
  }
  .current-channel {
    font-size: 14px;
    max-width: 120px;
  }
}
</style>


