<template>
  <a-modal
    v-model:open="visible"
    title="用户资料"
    width="600px"
    :footer="null"
  >
    <div class="user-profile">
      <!-- 用户头像和基本信息 -->
      <div class="profile-header">
        <a-avatar
          :size="80"
          :src="user.avatar"
          :style="{ backgroundColor: getAvatarColor(user.name) }"
        >
          {{ user.name[0] }}
        </a-avatar>
        
        <div class="profile-info">
          <h3 class="user-name">
            {{ user.name }}
            <a-tag v-if="user.role === 'admin'" color="red">管理员</a-tag>
            <a-tag v-else-if="user.role === 'leader'" color="orange">队长</a-tag>
          </h3>
          <p class="user-email">{{ user.email }}</p>
          <a-space>
            <a-badge :status="user.online ? 'success' : 'default'" />
            <span class="status-text">{{ user.online ? '在线' : '离线' }}</span>
          </a-space>
        </div>

        <a-button v-if="isEditable" type="primary" @click="editMode = !editMode">
          <EditOutlined /> {{ editMode ? '取消编辑' : '编辑资料' }}
        </a-button>
      </div>

      <a-divider />

      <!-- 编辑模式 -->
      <div v-if="editMode" class="edit-form">
        <a-form layout="vertical">
          <a-form-item label="昵称">
            <a-input v-model:value="editData.nickname" placeholder="输入昵称" />
          </a-form-item>

          <a-form-item label="邮箱">
            <a-input v-model:value="editData.email" placeholder="输入邮箱" />
          </a-form-item>

          <a-form-item label="技能标签">
            <a-select
              v-model:value="editData.skills"
              mode="tags"
              placeholder="选择或输入技能"
              style="width: 100%"
            >
              <a-select-option value="Web">Web</a-select-option>
              <a-select-option value="Pwn">Pwn</a-select-option>
              <a-select-option value="Reverse">Reverse</a-select-option>
              <a-select-option value="Crypto">Crypto</a-select-option>
              <a-select-option value="Misc">Misc</a-select-option>
              <a-select-option value="Forensics">Forensics</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="个人简介">
            <a-textarea
              v-model:value="editData.bio"
              :rows="4"
              placeholder="介绍一下自己..."
              :maxlength="200"
            />
            <div class="char-count">{{ editData.bio.length }}/200</div>
          </a-form-item>

          <a-form-item>
            <a-space>
              <a-button type="primary" @click="saveProfile">
                <SaveOutlined /> 保存
              </a-button>
              <a-button @click="editMode = false">取消</a-button>
            </a-space>
          </a-form-item>
        </a-form>
      </div>

      <!-- 查看模式 -->
      <div v-else class="profile-content">
        <!-- 技能标签 -->
        <div class="profile-section">
          <h4 class="section-title">
            <TagsOutlined /> 技能标签
          </h4>
          <a-space wrap>
            <a-tag
              v-for="skill in user.skills"
              :key="skill"
              :color="getSkillColor(skill)"
              size="large"
            >
              {{ skill }}
            </a-tag>
            <span v-if="user.skills.length === 0" class="empty-text">
              暂无技能标签
            </span>
          </a-space>
        </div>

        <!-- 个人简介 -->
        <div class="profile-section">
          <h4 class="section-title">
            <FileTextOutlined /> 个人简介
          </h4>
          <p class="bio-text">
            {{ user.bio || '这个人很懒，还没有写简介...' }}
          </p>
        </div>

        <!-- 统计信息 -->
        <div class="profile-section">
          <h4 class="section-title">
            <BarChartOutlined /> 解题统计
          </h4>
          <a-row :gutter="16">
            <a-col :span="8">
              <a-statistic
                title="已解题目"
                :value="user.stats.solved"
                suffix="题"
              >
                <template #prefix>
                  <TrophyOutlined style="color: #faad14" />
                </template>
              </a-statistic>
            </a-col>
            <a-col :span="8">
              <a-statistic
                title="获得积分"
                :value="user.stats.points"
                suffix="分"
              >
                <template #prefix>
                  <FireOutlined style="color: #f5222d" />
                </template>
              </a-statistic>
            </a-col>
            <a-col :span="8">
              <a-statistic
                title="团队排名"
                :value="user.stats.rank"
                suffix="名"
              >
                <template #prefix>
                  <CrownOutlined style="color: #1890ff" />
                </template>
              </a-statistic>
            </a-col>
          </a-row>
        </div>

        <!-- 最近活动 -->
        <div class="profile-section">
          <h4 class="section-title">
            <ClockCircleOutlined /> 最近活动
          </h4>
          <a-timeline>
            <a-timeline-item
              v-for="activity in user.activities"
              :key="activity.id"
              :color="getActivityColor(activity.type)"
            >
              <template #dot>
                <component :is="getActivityIcon(activity.type)" />
              </template>
              <div class="activity-content">
                <span class="activity-text">{{ activity.text }}</span>
                <span class="activity-time">{{ formatTime(activity.time) }}</span>
              </div>
            </a-timeline-item>
          </a-timeline>
        </div>

        <!-- 加入时间 -->
        <div class="profile-section">
          <h4 class="section-title">
            <CalendarOutlined /> 加入时间
          </h4>
          <p>{{ formatDate(user.joinedAt) }}</p>
        </div>
      </div>
    </div>
  </a-modal>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { message } from 'ant-design-vue'
import {
  EditOutlined,
  SaveOutlined,
  TagsOutlined,
  FileTextOutlined,
  BarChartOutlined,
  ClockCircleOutlined,
  CalendarOutlined,
  TrophyOutlined,
  FireOutlined,
  CrownOutlined,
  CheckCircleOutlined,
  MessageOutlined,
  FlagOutlined
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'

const props = defineProps({
  open: {
    type: Boolean,
    default: false
  },
  userId: {
    type: String,
    default: null
  },
  isEditable: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:open', 'update'])

const visible = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

const editMode = ref(false)

// 模拟用户数据
const user = ref({
  id: 'user_001',
  name: 'alice',
  nickname: 'Alice',
  email: 'alice@example.com',
  avatar: null,
  role: 'member',
  online: true,
  skills: ['Web', 'Crypto'],
  bio: '喜欢研究Web安全和密码学，擅长SQL注入和XSS攻击。',
  stats: {
    solved: 42,
    points: 3500,
    rank: 3
  },
  activities: [
    {
      id: '1',
      type: 'solve',
      text: '完成了题目 "SQL注入进阶"',
      time: new Date(Date.now() - 600000)
    },
    {
      id: '2',
      type: 'message',
      text: '发送了一条消息',
      time: new Date(Date.now() - 1200000)
    },
    {
      id: '3',
      type: 'flag',
      text: '提交了正确的 Flag',
      time: new Date(Date.now() - 1800000)
    }
  ],
  joinedAt: new Date('2024-01-15')
})

const editData = ref({
  nickname: user.value.nickname,
  email: user.value.email,
  skills: [...user.value.skills],
  bio: user.value.bio
})

watch(() => props.open, (newVal) => {
  if (newVal) {
    // 重置编辑数据
    editData.value = {
      nickname: user.value.nickname,
      email: user.value.email,
      skills: [...user.value.skills],
      bio: user.value.bio
    }
    editMode.value = false
  }
})

const saveProfile = () => {
  // 更新用户数据
  user.value.nickname = editData.value.nickname
  user.value.email = editData.value.email
  user.value.skills = editData.value.skills
  user.value.bio = editData.value.bio
  
  message.success('资料已保存')
  editMode.value = false
  emit('update', user.value)
}

const getAvatarColor = (name) => {
  const colors = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1']
  return colors[name.charCodeAt(0) % colors.length]
}

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

const getActivityColor = (type) => {
  const colors = {
    solve: 'green',
    message: 'blue',
    flag: 'orange'
  }
  return colors[type] || 'gray'
}

const getActivityIcon = (type) => {
  const icons = {
    solve: CheckCircleOutlined,
    message: MessageOutlined,
    flag: FlagOutlined
  }
  return icons[type] || CheckCircleOutlined
}

const formatTime = (time) => {
  return dayjs(time).fromNow()
}

const formatDate = (date) => {
  return dayjs(date).format('YYYY年MM月DD日')
}
</script>

<style scoped>
.user-profile {
  padding: 8px 0;
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 24px;
}

.profile-info {
  flex: 1;
}

.user-name {
  margin: 0 0 8px 0;
  font-size: 20px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-email {
  margin: 0 0 8px 0;
  color: rgba(0, 0, 0, 0.65);
}

.status-text {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.65);
}

.profile-section {
  margin-bottom: 24px;
}

.section-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  gap: 8px;
}

.bio-text {
  color: rgba(0, 0, 0, 0.65);
  line-height: 1.6;
  margin: 0;
}

.empty-text {
  color: rgba(0, 0, 0, 0.25);
  font-size: 13px;
}

.activity-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.activity-text {
  color: rgba(0, 0, 0, 0.85);
}

.activity-time {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.char-count {
  text-align: right;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  margin-top: 4px;
}
</style>

