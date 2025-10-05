<template>
  <a-config-provider :theme="themeConfig">
    <router-view />
  </a-config-provider>
</template>

<script setup>
import { ref } from 'vue'
import { theme } from 'ant-design-vue'

// Ant Design 标准主题配置
const themeConfig = ref({
  token: {
    colorPrimary: '#1890ff',
    colorSuccess: '#52c41a',
    colorWarning: '#faad14',
    colorError: '#ff4d4f',
    colorInfo: '#1890ff',
    fontSize: 14,
    borderRadius: 2,
    colorBgContainer: '#ffffff',
    colorBgLayout: '#f5f5f5',
  },
  algorithm: theme.defaultAlgorithm,
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

#app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}
</style>

