<template>
  <a-modal
    v-model:open="visible"
    title="分配题目"
    width="600px"
    @ok="handleAssign"
    @cancel="handleCancel"
  >
    <a-form layout="vertical">
      <!-- 选择成员 -->
      <a-form-item label="选择成员">
        <a-transfer
          v-model:target-keys="selectedMembers"
          :data-source="memberList"
          :titles="['可用成员', '已选成员']"
          :render="item => item.title"
          :show-search="true"
          :filter-option="filterOption"
          @change="handleChange"
        />
      </a-form-item>

      <!-- 分配类型 -->
      <a-form-item label="分配类型">
        <a-radio-group v-model:value="assignType">
          <a-radio value="individual">独立完成</a-radio>
          <a-radio value="collaborative">协作完成</a-radio>
        </a-radio-group>
        <div class="help-text">
          <InfoCircleOutlined />
          {{ assignType === 'individual' ? '每个成员独立解题' : '所有成员协作解题' }}
        </div>
      </a-form-item>

      <!-- 优先级 -->
      <a-form-item label="优先级">
        <a-select v-model:value="priority">
          <a-select-option value="low">
            <a-tag color="default">低</a-tag>
          </a-select-option>
          <a-select-option value="medium">
            <a-tag color="blue">中</a-tag>
          </a-select-option>
          <a-select-option value="high">
            <a-tag color="orange">高</a-tag>
          </a-select-option>
          <a-select-option value="urgent">
            <a-tag color="red">紧急</a-tag>
          </a-select-option>
        </a-select>
      </a-form-item>

      <!-- 截止时间 -->
      <a-form-item label="截止时间（可选）">
        <a-date-picker
          v-model:value="deadline"
          show-time
          format="YYYY-MM-DD HH:mm"
          placeholder="选择截止时间"
          style="width: 100%"
        />
      </a-form-item>

      <!-- 备注 -->
      <a-form-item label="备注">
        <a-textarea
          v-model:value="notes"
          :rows="3"
          placeholder="添加分配说明或要求"
        />
      </a-form-item>

      <!-- 已选成员预览 -->
      <a-form-item label="已选成员" v-if="selectedMembers.length > 0">
        <a-space wrap>
          <a-tag
            v-for="key in selectedMembers"
            :key="key"
            closable
            @close="removeSelectedMember(key)"
          >
            {{ getMemberName(key) }}
          </a-tag>
        </a-space>
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { InfoCircleOutlined } from '@ant-design/icons-vue'
import { getMembers } from '@/api/app'

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

const emit = defineEmits(['update:open', 'assign'])

const visible = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

// 成员列表（从后端加载）
const members = ref([])
const loading = ref(false)

// 加载成员列表
const loadMembers = async () => {
  loading.value = true
  try {
    const memberList = await getMembers()
    console.log('Loaded members:', memberList)
    if (Array.isArray(memberList)) {
      members.value = memberList.map(m => ({
        id: m.id || m.ID,
        name: m.nickname || m.Nickname || 'Unknown',
        role: m.role || m.Role || 'member',
        skills: [] // TODO: 如果后端支持技能标签，在这里解析
      }))
    }
  } catch (error) {
    console.error('Failed to load members:', error)
  } finally {
    loading.value = false
  }
}

// 监听弹窗打开时加载数据
watch(() => props.open, (newVal) => {
  if (newVal) {
    loadMembers()
  }
})

// 组件挂载时加载数据
onMounted(() => {
  if (props.open) {
    loadMembers()
  }
})

const memberList = computed(() => 
  members.value.map(m => ({
    key: m.id,
    title: `${m.name} (${m.skills.join(', ')})`,
    disabled: false
  }))
)

const selectedMembers = ref([])
const assignType = ref('collaborative')
const priority = ref('medium')
const deadline = ref(null)
const notes = ref('')

const filterOption = (inputValue, option) => {
  return option.title.toLowerCase().includes(inputValue.toLowerCase())
}

const handleChange = (targetKeys) => {
  selectedMembers.value = targetKeys
}

const removeSelectedMember = (key) => {
  selectedMembers.value = selectedMembers.value.filter(k => k !== key)
}

const getMemberName = (key) => {
  const member = members.value.find(m => m.id === key)
  return member ? member.name : key
}

const handleAssign = () => {
  if (selectedMembers.value.length === 0) {
    return
  }

  emit('assign', {
    members: selectedMembers.value,
    type: assignType.value,
    priority: priority.value,
    deadline: deadline.value,
    notes: notes.value
  })

  handleCancel()
}

const handleCancel = () => {
  selectedMembers.value = []
  assignType.value = 'collaborative'
  priority.value = 'medium'
  deadline.value = null
  notes.value = ''
  visible.value = false
}
</script>

<style scoped>
.help-text {
  margin-top: 8px;
  font-size: 13px;
  color: rgba(0, 0, 0, 0.45);
}
</style>

