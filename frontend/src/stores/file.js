import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useFileStore = defineStore('file', () => {
  // 文件列表
  const files = ref([])

  // 正在上传的文件
  const uploadingFiles = ref([])

  // Getters
  const getFileById = computed(() => (fileId) => {
    return files.value.find(f => f.id === fileId)
  })

  const uploadProgress = computed(() => (fileId) => {
    const file = uploadingFiles.value.find(f => f.id === fileId)
    return file ? file.progress : 0
  })

  const totalFileSize = computed(() => {
    return files.value.reduce((sum, file) => sum + file.size, 0)
  })

  // Actions
  const addFile = (file) => {
    const exists = files.value.find(f => f.id === file.id)
    if (!exists) {
      files.value.push(file)
    }
  }

  const removeFile = (fileId) => {
    const index = files.value.findIndex(f => f.id === fileId)
    if (index !== -1) {
      files.value.splice(index, 1)
    }
  }

  const startUpload = (file) => {
    uploadingFiles.value.push({
      id: file.id,
      name: file.name,
      size: file.size,
      progress: 0,
      status: 'uploading'
    })
  }

  const updateUploadProgress = (fileId, progress) => {
    const file = uploadingFiles.value.find(f => f.id === fileId)
    if (file) {
      file.progress = progress
      if (progress >= 100) {
        file.status = 'completed'
      }
    }
  }

  const completeUpload = (fileId, fileInfo) => {
    // 从上传列表移除
    const uploadIndex = uploadingFiles.value.findIndex(f => f.id === fileId)
    if (uploadIndex !== -1) {
      uploadingFiles.value.splice(uploadIndex, 1)
    }
    
    // 添加到文件列表
    addFile(fileInfo)
  }

  const failUpload = (fileId, error) => {
    const file = uploadingFiles.value.find(f => f.id === fileId)
    if (file) {
      file.status = 'failed'
      file.error = error
    }
  }

  const cancelUpload = (fileId) => {
    const index = uploadingFiles.value.findIndex(f => f.id === fileId)
    if (index !== -1) {
      uploadingFiles.value.splice(index, 1)
    }
  }

  const reset = () => {
    files.value = []
    uploadingFiles.value = []
  }

  return {
    files,
    uploadingFiles,
    getFileById,
    uploadProgress,
    totalFileSize,
    addFile,
    removeFile,
    startUpload,
    updateUploadProgress,
    completeUpload,
    failUpload,
    cancelUpload,
    reset
  }
})

