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
                <a-menu-item v-if="canDelete(file)" danger @click="deleteFile(file)">
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
              <a-button v-if="canDelete(record)" type="link" size="small" danger @click="deleteFile(record)">
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
          <a-button v-if="canDelete(selectedFile)" danger block @click="deleteFile(selectedFile)">
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
import { ref, computed, onMounted, h } from 'vue'
import { EventsOn } from '../../wailsjs/runtime/runtime'
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
import { getFiles, uploadFile, downloadFile as downloadFileAPI, deleteFile as deleteFileAPI, selectFile as selectFileDialog, getFileContent } from '@/api/app'
import { useAppStore } from '@/stores/app'

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
const loading = ref(false)
const appStore = useAppStore()

// 加载文件列表
const loadFiles = async () => {
  loading.value = true
  try {
    const fileList = await getFiles(100, 0)
    console.log('Loaded files:', fileList)
    if (Array.isArray(fileList)) {
      files.value = fileList.map(f => ({
        id: f.id || f.ID,
        name: f.name || f.Name || 'unnamed',
        type: getFileTypeFromMime(f.mime_type || f.MimeType || ''),
        size: f.size || f.Size || 0,
        url: f.url || '',
        uploader: f.uploader_name || f.UploaderName || 'Unknown',
        uploaderId: f.uploader_id || f.UploaderID || f.sender_id || f.SenderID || '',
        uploadedAt: f.upload_time ? new Date(f.upload_time * 1000) : new Date(),
        downloads: 0
      }))
    }
  } catch (error) {
    console.error('Failed to load files:', error)
    message.warning('加载文件列表失败')
  } finally {
    loading.value = false
  }
}

// 根据MIME类型获取文件类型
const getFileTypeFromMime = (mimeType) => {
  if (mimeType.startsWith('image/')) return 'image'
  if (mimeType.startsWith('video/')) return 'video'
  if (mimeType.startsWith('audio/')) return 'audio'
  if (mimeType.includes('pdf')) return 'document'
  if (mimeType.includes('zip') || mimeType.includes('rar') || mimeType.includes('tar')) return 'archive'
  if (mimeType.includes('javascript') || mimeType.includes('python') || mimeType.includes('java')) return 'code'
  return 'document'
}

// 组件挂载时加载文件
onMounted(() => {
  loadFiles()

  // 监听文件删除事件
  EventsOn('file:deleted', (data) => {
    if (!data) return
    const fileId = data.file_id || data.id
    const idx = files.value.findIndex(f => f.id === fileId)
    if (idx !== -1) {
      const removed = files.value.splice(idx, 1)
      message.info(`文件已删除: ${data.filename || removed[0]?.name || fileId}`)
    }
  })

  // 监听来自 NotificationCenter 的全局删除事件（用于其他来源触发）
  window.addEventListener('cw:file:deleted', (e) => {
    const detail = e?.detail || {}
    const fileId = detail.fileId
    if (!fileId) return
    const idx = files.value.findIndex(f => f.id === fileId)
    if (idx !== -1) {
      files.value.splice(idx, 1)
    }
    if (selectedFile.value?.id === fileId) {
      selectedFile.value = null
      showDetails.value = false
    }
  })
})

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

const handleUpload = async () => {
  loading.value = true
  try {
    // 走系统选择文件，拿到路径后调用后端 UploadFile
    const sel = await selectFileDialog('选择要上传的文件', '所有文件 (*.*)')
    const filePath = sel?.path
    if (!filePath) {
      loading.value = false
      return false
    }
    message.loading({ content: '正在上传文件...', key: 'fm-upload', duration: 0 })
    const res = await uploadFile({ file_path: filePath })
    const filename = res?.filename || filePath.split(/[\\/]/).pop()
    message.success({ content: `已开始上传: ${filename}`, key: 'fm-upload' })
    await loadFiles()
  } catch (error) {
    console.error('Upload failed:', error)
    message.error({ content: `上传失败: ${error.message || '未知错误'}`, key: 'fm-upload' })
  } finally {
    loading.value = false
  }
  return false // 阻止默认上传行为
}

const downloadFile = async (file) => {
  try {
    message.info(`正在下载 ${file.name}`)
    await downloadFileAPI(file.id)
    message.success(`下载完成: ${file.name}`)
  } catch (error) {
    console.error('Download failed:', error)
    message.error(`下载失败: ${error.message || '未知错误'}`)
  }
}

const previewFile = async (file) => {
  try {
    const res = await getFileContent(file.id)
    if (res.mode === 'text') {
      Modal.info({
        title: `预览: ${file.name}`,
        width: 800,
        content: h(
          'pre',
          { style: { maxHeight: '60vh', overflow: 'auto', whiteSpace: 'pre-wrap' } },
          res.text
        )
      })
    } else if (res.mode === 'dataurl') {
      // 图片或PDF的简单预览（图片直接展示，PDF交由浏览器处理）
      if ((res.mime || '').startsWith('image/')) {
        Modal.info({
          title: `预览: ${file.name}`,
          width: 900,
          content: h('img', {
            src: res.dataUrl,
            alt: file.name,
            style: { maxWidth: '100%', maxHeight: '70vh', objectFit: 'contain' }
          })
        })
      } else {
        window.open(res.dataUrl, '_blank')
      }
    } else {
      message.warning('该文件类型暂不支持预览')
    }
  } catch (e) {
    console.error('Preview failed:', e)
    message.error('预览失败')
  }
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
    async onOk() {
      try {
        await deleteFileAPI(file.id)
        const index = files.value.findIndex(f => f.id === file.id)
        if (index > -1) {
          files.value.splice(index, 1)
        }
        message.success('文件已删除')
        if (selectedFile.value?.id === file.id) {
          selectedFile.value = null
          showDetails.value = false
        }
      } catch (error) {
        console.error('Delete failed:', error)
        message.error(`删除失败: ${error.message || '未知错误'}`)
      }
    }
  })
}

// 上传者或管理员可删除（后端也会再次校验）
const canDelete = (file) => {
  const myId = appStore.currentUser?.id || ''
  const myRole = (appStore.currentUser?.role || '').toLowerCase()
  const isAdmin = myRole === 'admin' || myRole === 'owner' || myRole === 'moderator'
  const uploaderId = file?.uploaderId || ''
  return isAdmin || (!!myId && myId === uploaderId)
}

const batchDownload = async () => {
  if (selectedFiles.value.length === 0) {
    message.warning('请先选择要下载的文件')
    return
  }

  message.loading({ content: `正在下载 ${selectedFiles.value.length} 个文件...`, key: 'batch-download', duration: 0 })
  const concurrency = 3
  let success = 0
  let fail = 0
  for (let i = 0; i < selectedFiles.value.length; i += concurrency) {
    const batch = selectedFiles.value.slice(i, i + concurrency)
    await Promise.all(batch.map(async (f) => {
      try {
        await downloadFileAPI(f.id)
        success++
      } catch (e) {
        console.error('Download failed:', e)
        fail++
      }
    }))
  }
  message.success({ content: `下载完成: 成功 ${success} 个，失败 ${fail} 个`, key: 'batch-download' })
  selectedFiles.value = []
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

