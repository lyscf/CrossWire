<template>
  <div v-if="visible" class="mention-selector" :style="position">
    <div class="mention-header">
      <span class="mention-title">选择成员</span>
      <span class="mention-hint">↑↓选择 Enter确认 Esc取消</span>
    </div>
    
    <div class="mention-list">
      <div
        v-for="(member, index) in filteredMembers"
        :key="member.id"
        class="mention-item"
        :class="{ 'mention-item-active': index === activeIndex }"
        @click="selectMember(member)"
        @mouseenter="activeIndex = index"
      >
        <a-avatar
          :size="32"
          :style="{ backgroundColor: getAvatarColor(member.name) }"
        >
          {{ member.name[0] }}
        </a-avatar>
        
        <div class="member-info">
          <div class="member-name">
            <span class="name-text">{{ member.name }}</span>
            <a-badge
              v-if="member.online"
              status="success"
              text="在线"
              class="online-badge"
            />
          </div>
          <div v-if="member.task" class="member-task">
            {{ member.task }}
          </div>
        </div>

        <div v-if="member.skills && member.skills.length > 0" class="member-skills">
          <a-tag
            v-for="skill in member.skills.slice(0, 2)"
            :key="skill"
            size="small"
            :color="getSkillColor(skill)"
          >
            {{ skill }}
          </a-tag>
        </div>
      </div>

      <div v-if="filteredMembers.length === 0" class="mention-empty">
        <a-empty
          :image="Empty.PRESENTED_IMAGE_SIMPLE"
          description="没有匹配的成员"
        />
      </div>
    </div>

    <div class="mention-footer">
      找到 {{ filteredMembers.length }} 个成员
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { Empty } from 'ant-design-vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  searchText: {
    type: String,
    default: ''
  },
  members: {
    type: Array,
    default: () => []
  },
  position: {
    type: Object,
    default: () => ({ left: '0px', top: '0px' })
  }
})

const emit = defineEmits(['select', 'cancel'])

const activeIndex = ref(0)

// 过滤成员列表
const filteredMembers = computed(() => {
  if (!props.searchText) {
    return props.members
  }
  
  const search = props.searchText.toLowerCase()
  return props.members.filter(member => {
    return member.name.toLowerCase().includes(search) ||
           (member.task && member.task.toLowerCase().includes(search)) ||
           (member.skills && member.skills.some(skill => 
             skill.toLowerCase().includes(search)
           ))
  })
})

// 选择成员
const selectMember = (member) => {
  emit('select', member)
  activeIndex.value = 0
}

// 键盘导航
const handleKeyDown = (e) => {
  if (!props.visible) return

  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      activeIndex.value = Math.min(
        activeIndex.value + 1,
        filteredMembers.value.length - 1
      )
      scrollToActive()
      break
    
    case 'ArrowUp':
      e.preventDefault()
      activeIndex.value = Math.max(activeIndex.value - 1, 0)
      scrollToActive()
      break
    
    case 'Enter':
      e.preventDefault()
      if (filteredMembers.value[activeIndex.value]) {
        selectMember(filteredMembers.value[activeIndex.value])
      }
      break
    
    case 'Escape':
      e.preventDefault()
      emit('cancel')
      break
    
    case 'Tab':
      e.preventDefault()
      if (filteredMembers.value[activeIndex.value]) {
        selectMember(filteredMembers.value[activeIndex.value])
      }
      break
  }
}

// 滚动到选中项
const scrollToActive = () => {
  const container = document.querySelector('.mention-list')
  const activeItem = document.querySelector('.mention-item-active')
  
  if (container && activeItem) {
    const containerRect = container.getBoundingClientRect()
    const itemRect = activeItem.getBoundingClientRect()
    
    if (itemRect.bottom > containerRect.bottom) {
      activeItem.scrollIntoView({ block: 'end', behavior: 'smooth' })
    } else if (itemRect.top < containerRect.top) {
      activeItem.scrollIntoView({ block: 'start', behavior: 'smooth' })
    }
  }
}

// 获取头像颜色
const getAvatarColor = (name) => {
  const colors = [
    '#1890ff',
    '#52c41a',
    '#faad14',
    '#f5222d',
    '#722ed1',
    '#13c2c2',
    '#eb2f96'
  ]
  const index = name.charCodeAt(0) % colors.length
  return colors[index]
}

// 获取技能标签颜色
const getSkillColor = (skill) => {
  const colors = {
    Web: 'blue',
    Pwn: 'red',
    Reverse: 'purple',
    Crypto: 'orange',
    Misc: 'green',
    Forensics: 'cyan'
  }
  return colors[skill] || 'default'
}

// 监听搜索文本变化，重置选中索引
watch(() => props.searchText, () => {
  activeIndex.value = 0
})

// 监听可见性变化
watch(() => props.visible, (newVal) => {
  if (newVal) {
    activeIndex.value = 0
  }
})

onMounted(() => {
  document.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeyDown)
})
</script>

<style scoped>
.mention-selector {
  position: absolute;
  z-index: 1000;
  width: 320px;
  background: white;
  border-radius: 4px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  border: 1px solid #f0f0f0;
  overflow: hidden;
}

.mention-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background-color: #fafafa;
  border-bottom: 1px solid #f0f0f0;
}

.mention-title {
  font-size: 13px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
}

.mention-hint {
  font-size: 11px;
  color: rgba(0, 0, 0, 0.45);
}

.mention-list {
  max-height: 240px;
  overflow-y: auto;
}

.mention-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.mention-item:hover,
.mention-item-active {
  background-color: #f5f5f5;
}

.mention-item-active {
  background-color: #e6f7ff;
  border-left: 2px solid #1890ff;
}

.member-info {
  flex: 1;
  min-width: 0;
}

.member-name {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 2px;
}

.name-text {
  font-size: 14px;
  font-weight: 500;
  color: rgba(0, 0, 0, 0.85);
}

.online-badge {
  font-size: 11px;
}

.member-task {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.member-skills {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.mention-empty {
  padding: 24px;
}

.mention-footer {
  padding: 6px 12px;
  background-color: #fafafa;
  border-top: 1px solid #f0f0f0;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  text-align: center;
}

/* 滚动条样式 */
.mention-list::-webkit-scrollbar {
  width: 6px;
}

.mention-list::-webkit-scrollbar-track {
  background: transparent;
}

.mention-list::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.15);
  border-radius: 3px;
}

.mention-list::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.25);
}
</style>

