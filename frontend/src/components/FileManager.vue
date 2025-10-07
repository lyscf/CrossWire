<template>
  <a-drawer
    v-model:open="visible"
    title="文件管理器"
    placement="right"
    width="600"
    :body-style="{ padding: '16px' }"
  >
    <!-- 工具栏 -->
    <div class="file-toolbar">
      <SearchInput
        v-model="searchText"
        placeholder="搜索文件..."
        :debounce="300"
        class="file-search"
      />
      <a-upload
        :show-upload-list="false"
        :before-upload="handleUpload"
        multiple
      >
        <a-button type="primary">
          <UploadOutlined /> 上传文件
        </a-button>
      </a-upload>
    </div>

    <!-- 筛选器 -->
    <div class="file-filters">
      <a-space>
        <a-select
          v-model:value="filterType"
          placeholder="文件类型"
          style="width: 150px"
          allow-clear
        >
          <a-select-option value="all">所有文件</a-select-option>
          <a-select-option value="image">图片</a-select-option>
          <a-select-option value="document">文档</a-select-option>
          <a-select-option value="code">代码</a-select-option>
          <a-select-option value="archive">压缩包</a-select-option>
        </a-select>

        <a-select
          v-model:value="sortBy"
          style="width: 150px"
        >
          <a-select-option value="time">按时间</a-select-option>
          <a-select-option value="name">按名称</a-select-option>
          <a-select-option value="size">按大小</a-select-option>
        </a-select>

        <a-radio-group v-model:value="viewMode" button-style="solid" size="small">
          <a-radio-button value="grid">
            <AppstoreOutlined />
          </a-radio-button>
          <a-radio-button value="list">
            <BarsOutlined />
          </a-radio-button>
        </a-radio-group>
      </a-space>
    </div>

    <!-- 文件列表 - 网格视图 -->
    <div v-if="viewMode === 'grid'" class="file-grid">
      <div
        v-for="file in filteredFiles"
        :key="file.id"
        class="file-card"
        @click="selectFile(file)"
        :class="{ 'file-selected': selectedFile?.id === file.id }"
      >
        <div class="file-preview">
          <img
            v-if="file.type === 'image'"
            :src="file.url"
            :alt="file.name"
            class="preview-image"
          />
          <div v-else class="preview-icon">
            <component
              :is="getFileIcon(file.type)"
              :style="{ fontSize: '48px', color: getFileColor(file.type) }"
            />
          </div>
        </div>
        <div class="file-info">
          <div class="file-name" :title="file.name">{{ file.name }}</div>
          <div class="file-meta">
            <span class="file-size">{{ formatSize(file.size) }}</span>
            <span class="file-time">{{ formatTime(file.uploadedAt) }}</span>
          </div>
        </div>
        <div class="file-actions">
          <a-dropdown :trigger="['click']">
            <a-button type="text" size="small" @click.stop>
              <MoreOutlined />
            </a-button>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="downloadFile(file)">
                  <DownloadOutlined /> 下载
                </a-menu-item>
                <a-menu-item @click="previewFile(file)">
                  <EyeOutlined /> 预览
                </a-menu-item>
                <a-menu-item @click="shareFile(file)">
                  <ShareAltOutlined /> 分享
                </a-menu-item>
                <a-menu-divider />
                <a-menu-item danger @click="deleteFile(file)">
                  <DeleteOutlined /> 删除
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </div>

      <a-empty
        v-if="filteredFiles.length === 0"
        description="暂无文件"
        :image="Empty.PRESENTED_IMAGE_SIMPLE"
      />
    </div>

    <!-- 文件列表 - 列表视图 -->
    <div v-else class="file-list">
      <a-table
        :data-source="filteredFiles"
        :columns="columns"
        :pagination="{ pageSize: 20 }"
        :row-selection="rowSelection"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <div class="file-name-cell">
              <component
                :is="getFileIcon(record.type)"
                :style="{ color: getFileColor(record.type), marginRight: '8px' }"
              />
              <span>{{ record.name }}</span>
            </div>
          </template>
          <template v-else-if="column.key === 'size'">
            {{ formatSize(record.size) }}
          </template>
          <template v-else-if="column.key === 'uploadedAt'">
            {{ formatTime(record.uploadedAt) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" @click="downloadFile(record)">
                下载
              </a-button>
              <a-button type="link" size="small" @click="previewFile(record)">
                预览
              </a-button>
              <a-button type="link" size="small" danger @click="deleteFile(record)">
                删除
              </a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </div>

    <!-- 文件详情侧边栏 -->
    <a-drawer
      v-model:open="showDetails"
      title="文件详情"
      placement="right"
      width="400"
      :get-container="false"
      :style="{ position: 'absolute' }"
    >
      <div v-if="selectedFile" class="file-details">
        <div class="detail-preview">
          <img
            v-if="selectedFile.type === 'image'"
            :src="selectedFile.url"
            :alt="selectedFile.name"
            class="detail-image"
          />
          <component
            v-else
            :is="getFileIcon(selectedFile.type)"
            :style="{ fontSize: '80px', color: getFileColor(selectedFile.type) }"
          />
        </div>

        <a-descriptions :column="1" size="small" bordered>
          <a-descriptions-item label="文件名">
            {{ selectedFile.name }}
          </a-descriptions-item>
          <a-descriptions-item label="类型">
            {{ getFileTypeName(selectedFile.type) }}
          </a-descriptions-item>
          <a-descriptions-item label="大小">
            {{ formatSize(selectedFile.size) }}
          </a-descriptions-item>
          <a-descriptions-item label="上传者">
            {{ selectedFile.uploader }}
          </a-descriptions-item>
          <a-descriptions-item label="上传时间">
            {{ formatDate(selectedFile.uploadedAt) }}
          </a-descriptions-item>
          <a-descriptions-item label="下载次数">
            {{ selectedFile.downloads || 0 }} 次
          </a-descriptions-item>
        </a-descriptions>

        <a-divider />

        <a-space direction="vertical" style="width: 100%">
          <a-button type="primary" block @click="downloadFile(selectedFile)">
            <DownloadOutlined /> 下载文件
          </a-button>
          <a-button block @click="shareFile(selectedFile)">
            <ShareAltOutlined /> 生成分享链接
          </a-button>
          <a-button danger block @click="deleteFile(selectedFile)">
            <DeleteOutlined /> 删除文件
          </a-button>
        </a-space>
      </div>
    </a-drawer>

    <!-- 底部操作栏 -->
    <div class="file-footer">
      <div class="file-stats">
        <span>共 {{ files.length }} 个文件</span>
        <a-divider type="vertical" />
        <span>总大小 {{ formatSize(totalSize) }}</span>
      </div>
      <a-space>
        <a-button v-if="selectedFiles.length > 0" @click="batchDownload">
          <DownloadOutlined /> 批量下载 ({{ selectedFiles.length }})
        </a-button>
        <a-button v-if="selectedFiles.length > 0" danger @click="batchDelete">
          <DeleteOutlined /> 批量删除
        </a-button>
      </a-space>
    </div>
  </a-drawer>
</template>

<script setup>
import { ref, computed } from 'vue'
import { message, Modal, Empty } from 'ant-design-vue'
import SearchInput from '@/components/Common/SearchInput.vue'
import {
  UploadOutlined,
  DownloadOutlined,
  DeleteOutlined,
  EyeOutlined,
  ShareAltOutlined,
  MoreOutlined,
  AppstoreOutlined,
  BarsOutlined,
  FileTextOutlined,
  FileImageOutlined,
  FileZipOutlined,
  FilePdfOutlined,
  FileExcelOutlined,
  FileWordOutlined,
  FilePptOutlined,
  CodeOutlined as FileCodeOutlined
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'

const props = defineProps({
  open: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:open'])

const visible = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

const searchText = ref('')
const filterType = ref('all')
const sortBy = ref('time')
const viewMode = ref('grid')
const selectedFile = ref(null)
const showDetails = ref(false)
const selectedFiles = ref([])

// 文件数据（从后端加载）
const files = ref([])

const columns = [
  { title: '文件名', dataIndex: 'name', key: 'name', width: 300 },
  { title: '大小', dataIndex: 'size', key: 'size', width: 100 },
  { title: '上传者', dataIndex: 'uploader', key: 'uploader', width: 100 },
  { title: '上传时间', dataIndex: 'uploadedAt', key: 'uploadedAt', width: 150 },
  { title: '操作', key: 'actions', width: 180 }
]

const rowSelection = {
  selectedRowKeys: computed(() => selectedFiles.value.map(f => f.id)),
  onChange: (selectedRowKeys, selectedRows) => {
    selectedFiles.value = selectedRows
  }
}

const filteredFiles = computed(() => {
  let result = files.value

  // 搜索过滤
  if (searchText.value) {
    const query = searchText.value.toLowerCase()
    result = result.filter(f => f.name.toLowerCase().includes(query))
  }

  // 类型过滤
  if (filterType.value && filterType.value !== 'all') {
    result = result.filter(f => f.type === filterType.value)
  }

  // 排序
  result.sort((a, b) => {
    switch (sortBy.value) {
      case 'time':
        return b.uploadedAt - a.uploadedAt
      case 'name':
        return a.name.localeCompare(b.name)
      case 'size':
        return b.size - a.size
      default:
        return 0
    }
  })

  return result
})

const totalSize = computed(() => {
  return files.value.reduce((sum, file) => sum + file.size, 0)
})

const getFileIcon = (type) => {
  const icons = {
    image: FileImageOutlined,
    document: FileTextOutlined,
    code: FileCodeOutlined,
    archive: FileZipOutlined,
    pdf: FilePdfOutlined,
    excel: FileExcelOutlined,
    word: FileWordOutlined,
    ppt: FilePptOutlined
  }
  return icons[type] || FileTextOutlined
}

const getFileColor = (type) => {
  const colors = {
    image: '#52c41a',
    document: '#1890ff',
    code: '#722ed1',
    archive: '#faad14',
    pdf: '#f5222d',
    excel: '#13c2c2',
    word: '#2f54eb',
    ppt: '#eb2f96'
  }
  return colors[type] || '#8c8c8c'
}

const getFileTypeName = (type) => {
  const names = {
    image: '图片',
    document: '文档',
    code: '代码',
    archive: '压缩包',
    pdf: 'PDF',
    excel: 'Excel',
    word: 'Word',
    ppt: 'PowerPoint'
  }
  return names[type] || '未知'
}

const formatSize = (bytes) => {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

const formatTime = (time) => {
  return dayjs(time).fromNow()
}

const formatDate = (time) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

const selectFile = (file) => {
  selectedFile.value = file
  showDetails.value = true
}

const handleUpload = (file) => {
  message.success(`文件 ${file.name} 上传成功`)
  // TODO: 实现文件上传
  return false
}

const downloadFile = (file) => {
  message.info(`正在下载 ${file.name}`)
  // TODO: 实现文件下载
}

const previewFile = (file) => {
  message.info(`正在打开预览: ${file.name}`)
  // TODO: 实现文件预览
}

const shareFile = (file) => {
  message.success(`分享链接已复制到剪贴板`)
  // TODO: 实现文件分享
}

const deleteFile = (file) => {
  Modal.confirm({
    title: '确认删除？',
    content: `确定要删除文件 "${file.name}" 吗？此操作不可撤销。`,
    okText: '删除',
    cancelText: '取消',
    okType: 'danger',
    onOk() {
      const index = files.value.findIndex(f => f.id === file.id)
      if (index > -1) {
        files.value.splice(index, 1)
        message.success('文件已删除')
        if (selectedFile.value?.id === file.id) {
          selectedFile.value = null
          showDetails.value = false
        }
      }
    }
  })
}

const batchDownload = () => {
  message.info(`正在下载 ${selectedFiles.value.length} 个文件`)
  // TODO: 实现批量下载
}

const batchDelete = () => {
  Modal.confirm({
    title: '确认批量删除？',
    content: `确定要删除选中的 ${selectedFiles.value.length} 个文件吗？此操作不可撤销。`,
    okText: '删除',
    cancelText: '取消',
    okType: 'danger',
    onOk() {
      selectedFiles.value.forEach(file => {
        const index = files.value.findIndex(f => f.id === file.id)
        if (index > -1) {
          files.value.splice(index, 1)
        }
      })
      message.success(`已删除 ${selectedFiles.value.length} 个文件`)
      selectedFiles.value = []
    }
  })
}
</script>

<style scoped>
.file-toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.file-search {
  flex: 1;
}

.file-filters {
  margin-bottom: 16px;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
  margin-bottom: 60px;
}

.file-card {
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
  overflow: hidden;
}

.file-card:hover {
  border-color: #1890ff;
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.15);
}

.file-selected {
  border-color: #1890ff;
  background-color: #e6f7ff;
}

.file-preview {
  width: 100%;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #fafafa;
}

.preview-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: cover;
}

.preview-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-info {
  padding: 12px;
}

.file-name {
  font-size: 14px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 4px;
}

.file-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.file-actions {
  position: absolute;
  top: 8px;
  right: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}

.file-card:hover .file-actions {
  opacity: 1;
}

.file-name-cell {
  display: flex;
  align-items: center;
}

.file-details {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-preview {
  width: 100%;
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #fafafa;
  border-radius: 4px;
}

.detail-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.file-footer {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: white;
  border-top: 1px solid #f0f0f0;
}

.file-stats {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.65);
}
</style>

