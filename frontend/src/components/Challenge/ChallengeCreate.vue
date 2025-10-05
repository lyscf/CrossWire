<template>
  <a-modal
    v-model:open="visible"
    title="创建题目"
    width="600px"
    @ok="handleCreate"
    @cancel="handleCancel"
  >
    <a-form
      :model="formData"
      :label-col="{ span: 6 }"
      :wrapper-col="{ span: 18 }"
    >
      <a-form-item label="题目标题" required>
        <a-input
          v-model:value="formData.title"
          placeholder="输入题目标题"
        />
      </a-form-item>

      <a-form-item label="题目类型" required>
        <a-select v-model:value="formData.category">
          <a-select-option value="Web">Web</a-select-option>
          <a-select-option value="Pwn">Pwn</a-select-option>
          <a-select-option value="Reverse">Reverse</a-select-option>
          <a-select-option value="Crypto">Crypto</a-select-option>
          <a-select-option value="Misc">Misc</a-select-option>
          <a-select-option value="Forensics">Forensics</a-select-option>
        </a-select>
      </a-form-item>

      <a-form-item label="难度" required>
        <a-select v-model:value="formData.difficulty">
          <a-select-option value="Easy">Easy</a-select-option>
          <a-select-option value="Medium">Medium</a-select-option>
          <a-select-option value="Hard">Hard</a-select-option>
          <a-select-option value="Insane">Insane</a-select-option>
        </a-select>
      </a-form-item>

      <a-form-item label="分值" required>
        <a-input-number
          v-model:value="formData.points"
          :min="10"
          :max="1000"
          :step="10"
          style="width: 100%"
        />
      </a-form-item>

      <a-form-item label="题目描述" required>
        <a-textarea
          v-model:value="formData.description"
          :rows="4"
          placeholder="输入题目描述"
        />
      </a-form-item>

      <a-form-item label="Flag">
        <a-input
          v-model:value="formData.flag"
          placeholder="输入正确的 Flag（可选）"
        />
      </a-form-item>

      <a-form-item label="附件 URL">
        <a-input
          v-model:value="formData.attachmentUrl"
          placeholder="题目附件下载链接（可选）"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  open: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:open', 'created'])

const visible = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

const formData = ref({
  title: '',
  category: 'Web',
  difficulty: 'Medium',
  points: 100,
  description: '',
  flag: '',
  attachmentUrl: ''
})

const handleCreate = () => {
  // TODO: 调用 API 创建题目
  const newChallenge = {
    id: Date.now().toString(),
    ...formData.value,
    status: 'pending',
    assignedTo: [],
    progress: 0,
    solvedBy: null,
    createdAt: new Date().toISOString()
  }
  
  emit('created', newChallenge)
  handleCancel()
}

const handleCancel = () => {
  // 重置表单
  formData.value = {
    title: '',
    category: 'Web',
    difficulty: 'Medium',
    points: 100,
    description: '',
    flag: '',
    attachmentUrl: ''
  }
  visible.value = false
}
</script>

