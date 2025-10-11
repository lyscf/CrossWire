<template>
  <div class="file-preview">
    <a-card :title="file.name" size="small">
      <template #extra>
        <a-space>
          <a-button size="small" @click="handleDownload">
            <DownloadOutlined /> 下载
          </a-button>
          <a-button size="small" @click="$emit('close')">
            <CloseOutlined />
          </a-button>
        </a-space>
      </template>

      <!-- 图片预览 -->
      <div v-if="isImage" class="preview-image">
        <img :src="file.url" :alt="file.name" />
      </div>

      <!-- 文本预览 -->
      <div v-else-if="isText" class="preview-text">
        <pre>{{ file.content || '加载中...' }}</pre>
      </div>

      <!-- PDF 预览 -->
      <div v-else-if="isPdf" class="preview-pdf">
        <iframe :src="file.url" frameborder="0"></iframe>
      </div>

      <!-- 其他文件 -->
      <div v-else class="preview-default">
        <a-empty description="无法预览此文件类型">
          <template #image>
            <FileUnknownOutlined style="font-size: 64px; color: #d9d9d9" />
          </template>
        </a-empty>
        <div class="file-info">
          <p><strong>文件名:</strong> {{ file.name }}</p>
          <p><strong>大小:</strong> {{ formatSize(file.size) }}</p>
          <p><strong>类型:</strong> {{ file.type }}</p>
        </div>
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import {
  DownloadOutlined,
  CloseOutlined,
  FileUnknownOutlined
} from '@ant-design/icons-vue'

const props = defineProps({
  file: {
    type: Object,
    required: true
  }
})

defineEmits(['close'])

const isImage = computed(() => {
  return /\.(jpg|jpeg|png|gif|bmp|webp)$/i.test(props.file.name)
})

const isText = computed(() => {
  return /\.(txt|md|log|json|xml|csv)$/i.test(props.file.name)
})

const isPdf = computed(() => {
  return /\.pdf$/i.test(props.file.name)
})

const formatSize = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

const handleDownload = () => {
  const link = document.createElement('a')
  link.href = props.file.url
  link.download = props.file.name
  link.click()
}
</script>

<style scoped>
.file-preview {
  width: 100%;
  height: 100%;
}

.preview-image img {
  max-width: 100%;
  height: auto;
  display: block;
}

.preview-text pre {
  max-height: 500px;
  overflow-y: auto;
  background-color: #fafafa;
  padding: 16px;
  border-radius: 4px;
  font-size: 13px;
}

.preview-pdf iframe {
  width: 100%;
  height: 600px;
}

.preview-default {
  text-align: center;
  padding: 40px 20px;
}

.file-info {
  margin-top: 24px;
  text-align: left;
}

.file-info p {
  margin: 8px 0;
  color: rgba(0, 0, 0, 0.65);
}
</style>

