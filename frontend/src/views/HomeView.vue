<template>
  <div class="home-container">
    <div class="home-content">
      <!-- Header -->
      <div class="header">
        <h1 class="title">CrossWire</h1>
        <p class="subtitle">CTF 线下赛团队通讯系统</p>
        <a-divider />
      </div>

      <!-- Mode Selection Cards -->
      <div class="mode-selection">
        <a-card class="mode-card" hoverable @click="navigateToServer">
          <div class="card-content">
            <div class="card-icon">
              <CloudServerOutlined />
            </div>
            <div class="card-info">
              <h3 class="card-title">创建频道</h3>
              <p class="card-description">
                作为服务端启动，创建新的通讯频道，管理团队成员
              </p>
              <a-tag color="blue">服务端模式</a-tag>
            </div>
          </div>
        </a-card>

        <a-card class="mode-card" hoverable @click="navigateToClient">
          <div class="card-content">
            <div class="card-icon">
              <TeamOutlined />
            </div>
            <div class="card-info">
              <h3 class="card-title">加入频道</h3>
              <p class="card-description">
                作为客户端加入已存在的频道，与团队成员交流
              </p>
              <a-tag color="green">客户端模式</a-tag>
            </div>
          </div>
        </a-card>
      </div>

      <!-- Features -->
      <div class="features">
        <a-row :gutter="[16, 16]">
          <a-col :span="6">
            <div class="feature-item">
              <ThunderboltOutlined class="feature-icon" />
              <span class="feature-text">ARP 极速传输</span>
            </div>
          </a-col>
          <a-col :span="6">
            <div class="feature-item">
              <SafetyOutlined class="feature-icon" />
              <span class="feature-text">端到端加密</span>
            </div>
          </a-col>
          <a-col :span="6">
            <div class="feature-item">
              <FileProtectOutlined class="feature-icon" />
              <span class="feature-text">安全文件传输</span>
            </div>
          </a-col>
          <a-col :span="6">
            <div class="feature-item">
              <CodeOutlined class="feature-icon" />
              <span class="feature-text">代码高亮分享</span>
            </div>
          </a-col>
        </a-row>
      </div>

      <!-- Footer -->
      <div class="footer">
        <a-space :size="16">
          <span class="version">Version 1.0.0</span>
          <a-divider type="vertical" />
          <a-button type="link" size="small">文档</a-button>
          <a-button type="link" size="small">关于</a-button>
        </a-space>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import {
  CloudServerOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  SafetyOutlined,
  FileProtectOutlined,
  CodeOutlined
} from '@ant-design/icons-vue'

const router = useRouter()

const navigateToServer = () => {
  router.push('/server')
}

const navigateToClient = () => {
  router.push('/client')
}
</script>

<style scoped>
.home-container {
  width: 100%;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #f5f5f5;
}

.home-content {
  text-align: center;
  max-width: 1000px;
  padding: 60px 40px;
  background: white;
  border-radius: 2px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.header {
  margin-bottom: 48px;
}

.title {
  font-size: 42px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  margin-bottom: 8px;
  letter-spacing: -0.5px;
}

.subtitle {
  font-size: 16px;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 24px;
}

.mode-selection {
  display: flex;
  gap: 24px;
  margin-bottom: 48px;
  justify-content: center;
}

.mode-card {
  flex: 1;
  max-width: 400px;
  border: 1px solid #d9d9d9;
  transition: all 0.3s ease;
  cursor: pointer;
}

.mode-card:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 16px rgba(24, 144, 255, 0.15);
  transform: translateY(-4px);
}

.card-content {
  display: flex;
  align-items: flex-start;
  gap: 20px;
  padding: 8px;
}

.card-icon {
  flex-shrink: 0;
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #f5f5f5;
  border-radius: 4px;
  font-size: 32px;
  color: #1890ff;
}

.card-info {
  flex: 1;
  text-align: left;
}

.card-title {
  font-size: 18px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  margin-bottom: 8px;
}

.card-description {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.65);
  line-height: 1.6;
  margin-bottom: 12px;
}

.features {
  margin-bottom: 32px;
  padding: 24px;
  background-color: #fafafa;
  border-radius: 4px;
}

.feature-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px;
  background: white;
  border-radius: 4px;
  transition: all 0.3s;
}

.feature-item:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.feature-icon {
  font-size: 28px;
  color: #1890ff;
}

.feature-text {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.85);
  font-weight: 500;
}

.footer {
  padding-top: 24px;
  border-top: 1px solid #f0f0f0;
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
}

.version {
  color: rgba(0, 0, 0, 0.45);
}
</style>

