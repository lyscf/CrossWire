<template>
  <a-modal
    v-model:open="visible"
    title="提交 Flag"
    width="500px"
    @ok="handleSubmit"
    @cancel="handleCancel"
  >
    <a-form layout="vertical">
      <!-- Flag 输入 -->
      <a-form-item label="Flag" required>
        <a-input
          v-model:value="flag"
          placeholder="输入 Flag（例如：flag{...}）"
          size="large"
          :prefix="h(FlagOutlined)"
          @pressEnter="handleSubmit"
        />
        <div class="help-text">
          <InfoCircleOutlined />
          请输入完整的 Flag，包括格式符号
        </div>
      </a-form-item>

      <!-- 解题思路 -->
      <a-form-item label="解题思路（可选）">
        <a-textarea
          v-model:value="solution"
          :rows="4"
          placeholder="分享你的解题思路和方法..."
          :maxlength="1000"
        />
        <div class="char-count">{{ solution.length }}/1000</div>
      </a-form-item>

      <!-- 使用工具 -->
      <a-form-item label="使用工具（可选）">
        <a-select
          v-model:value="tools"
          mode="tags"
          placeholder="选择或输入使用的工具"
          style="width: 100%"
        >
          <a-select-option value="Burp Suite">Burp Suite</a-select-option>
          <a-select-option value="sqlmap">sqlmap</a-select-option>
          <a-select-option value="pwntools">pwntools</a-select-option>
          <a-select-option value="IDA Pro">IDA Pro</a-select-option>
          <a-select-option value="Ghidra">Ghidra</a-select-option>
          <a-select-option value="Wireshark">Wireshark</a-select-option>
          <a-select-option value="自写脚本">自写脚本</a-select-option>
        </a-select>
      </a-form-item>

      <!-- 附加文件 -->
      <a-form-item label="附加文件（可选）">
        <a-upload
          v-model:file-list="fileList"
          :before-upload="beforeUpload"
          :max-count="3"
        >
          <a-button>
            <UploadOutlined /> 上传 Exploit/Writeup
          </a-button>
        </a-upload>
        <div class="help-text">
          支持上传解题脚本、Writeup 等文件，最多 3 个
        </div>
      </a-form-item>

      <!-- 提交历史 -->
      <a-collapse v-if="submissions.length > 0" style="margin-top: 16px">
        <a-collapse-panel key="1" header="查看提交历史">
          <a-timeline size="small">
            <a-timeline-item
              v-for="sub in submissions"
              :key="sub.id"
              :color="sub.correct ? 'green' : 'red'"
            >
              <div class="submission-item">
                <div class="submission-header">
                  <strong>{{ sub.submitter }}</strong>
                  <a-tag :color="sub.correct ? 'success' : 'error'" size="small">
                    {{ sub.correct ? '正确' : '错误' }}
                  </a-tag>
                </div>
                <div class="submission-flag">
                  Flag: <code>{{ sub.flag }}</code>
                </div>
                <div class="submission-time">
                  {{ formatTime(sub.timestamp) }}
                </div>
              </div>
            </a-timeline-item>
          </a-timeline>
        </a-collapse-panel>
      </a-collapse>
    </a-form>
  </a-modal>
</template>

<script setup>
import { ref, computed, h, onMounted, watch } from 'vue'
import { message } from 'ant-design-vue'
import {
  FlagOutlined,
  InfoCircleOutlined,
  UploadOutlined
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { getChallengeSubmissions } from '@/api/app'

const props = defineProps({
  open: {
    type: Boolean,
    default: false
  },
  challenge: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:open', 'submit'])

const visible = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

const flag = ref('')
const solution = ref('')
const tools = ref([])
const fileList = ref([])

// 提交历史（从后端加载）
const submissions = ref([])
const loading = ref(false)

// 加载提交历史
const loadSubmissions = async () => {
  if (!props.challenge?.id) return
  loading.value = true
  try {
    const data = await getChallengeSubmissions(props.challenge.id)
    console.log('Loaded submissions:', data)
    if (Array.isArray(data)) {
      submissions.value = data.map(s => ({
        id: s.id || s.ID,
        flag: s.flag || '',
        correct: s.correct || false,
        submitTime: s.submit_time ? new Date(s.submit_time * 1000) : new Date(),
        feedback: s.feedback || ''
      }))
    }
  } catch (error) {
    console.error('Failed to load submissions:', error)
  } finally {
    loading.value = false
  }
}

// 监听弹窗打开时加载数据
watch(() => props.open, (newVal) => {
  if (newVal && props.challenge?.id) {
    loadSubmissions()
  }
})

// 组件挂载时加载数据
onMounted(() => {
  if (props.open && props.challenge?.id) {
    loadSubmissions()
  }
})

const beforeUpload = (file) => {
  const isLt10M = file.size / 1024 / 1024 < 10
  if (!isLt10M) {
    message.error('文件大小不能超过 10MB')
  }
  return isLt10M || Upload.LIST_IGNORE
}

const handleSubmit = () => {
  if (!flag.value.trim()) {
    message.warning('请输入 Flag')
    return
  }

  // 检查 Flag 格式
  if (!flag.value.match(/^flag\{.*\}$/i)) {
    message.warning('Flag 格式可能不正确，请确认')
  }

  emit('submit', {
    flag: flag.value,
    solution: solution.value,
    tools: tools.value,
    files: fileList.value
  })

  handleCancel()
}

const handleCancel = () => {
  flag.value = ''
  solution.value = ''
  tools.value = []
  fileList.value = []
  visible.value = false
}

const formatTime = (timestamp) => {
  return dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss')
}
</script>

<style scoped>
.help-text {
  margin-top: 8px;
  font-size: 13px;
  color: rgba(0, 0, 0, 0.45);
}

.char-count {
  text-align: right;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  margin-top: 4px;
}

.submission-item {
  padding: 8px 0;
}

.submission-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.submission-flag {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 4px;
}

.submission-flag code {
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 2px;
  font-family: 'Consolas', monospace;
}

.submission-time {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}
</style>

