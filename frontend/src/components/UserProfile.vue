<template>
  <a-modal
    v-model:open="visible"
    title="用户资料"
    width="600px"
    :footer="null"
    :loading="loading"
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
            <div v-for="(skill, index) in editData.skill_details" :key="index" class="skill-edit-item">
              <a-select
                v-model:value="skill.category"
                placeholder="选择技能"
                style="width: 120px"
              >
                <a-select-option value="Web">Web</a-select-option>
                <a-select-option value="Pwn">Pwn</a-select-option>
                <a-select-option value="Reverse">Reverse</a-select-option>
                <a-select-option value="Crypto">Crypto</a-select-option>
                <a-select-option value="Misc">Misc</a-select-option>
                <a-select-option value="Forensics">Forensics</a-select-option>
              </a-select>

              <a-rate v-model:value="skill.level" :count="5" style="margin: 0 12px" />

              <a-button type="text" danger size="small" @click="removeSkill(index)">
                <DeleteOutlined />
              </a-button>
            </div>

            <a-button type="dashed" block @click="addSkill" style="margin-top: 8px">
              <PlusOutlined /> 添加技能
            </a-button>
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
          <a-space wrap direction="vertical" :size="12">
            <div v-for="skill in user.skill_details || []" :key="skill.category" class="skill-item">
              <a-tag :color="getSkillColor(skill.category)" size="large" style="min-width: 80px">
                {{ skill.category }}
              </a-tag>
              <a-rate :value="skill.level || 0" :count="5" disabled style="font-size: 14px; margin-left: 8px" />
              <span v-if="skill.experience > 0" class="experience-text">{{ skill.experience }} 题</span>
            </div>
            <span v-if="(user.skill_details || []).length === 0" class="empty-text">暂无技能标签</span>
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
            <BarChartOutlined /> 贡献统计
          </h4>
          <a-row :gutter="[16, 16]">
            <a-col :span="8">
              <a-statistic
                title="参与题目"
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
                title="贡献度"
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
                :value="user.stats.rank || '--'"
                :suffix="user.stats.rank ? '名' : ''"
              >
                <template #prefix>
                  <CrownOutlined style="color: #1890ff" />
                </template>
              </a-statistic>
            </a-col>
            <a-col :span="8">
              <a-statistic
                title="发送消息"
                :value="user.stats.messages"
                suffix="条"
              >
                <template #prefix>
                  <MessageOutlined style="color: #52c41a" />
                </template>
              </a-statistic>
            </a-col>
            <a-col :span="8">
              <a-statistic
                title="分享文件"
                :value="user.stats.files"
                suffix="个"
              >
                <template #prefix>
                  <FlagOutlined style="color: #722ed1" />
                </template>
              </a-statistic>
            </a-col>
            <a-col :span="8">
              <a-statistic
                title="在线时长"
                :value="user.stats.onlineTimeFormatted"
              >
                <template #prefix>
                  <ClockCircleOutlined style="color: #13c2c2" />
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
import { ref, computed, watch, onMounted } from 'vue'
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
  FlagOutlined,
  DeleteOutlined,
  PlusOutlined
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { getMember, getMyInfo, updateUserProfile } from '@/api/app'

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
const loading = ref(false)

// 用户数据（从后端加载）
const user = ref({
  id: props.userId || 'unknown',
  name: 'User',
  nickname: 'User',
  email: '',
  avatar: null,
  role: 'member',
  online: false,
  skills: [],
  skill_details: [],
  bio: '',
  stats: {
    solved: 0,
    points: 0,
    rank: 0,
    messages: 0,
    files: 0,
    onlineTime: 0,
    onlineTimeFormatted: '0小时'
  },
  activities: [],
  joinedAt: new Date()
})

const editData = ref({
  nickname: user.value.nickname,
  email: user.value.email,
  skills: [...user.value.skills],
  skill_details: [],
  bio: user.value.bio
})

// 加载用户数据
const loadUserData = async () => {
  loading.value = true
  console.log('[UserProfile] Starting to load user data...')
  console.log('[UserProfile] Props:', {
    userId: props.userId,
    isEditable: props.isEditable,
    open: props.open
  })
  
  try {
    let memberData
    
    // 如果有userId且不是当前用户，获取指定成员信息
    if (props.userId && !props.isEditable) {
      console.log('[UserProfile] Fetching member info for userId:', props.userId)
      memberData = await getMember(props.userId)
      console.log('[UserProfile] getMember returned:', memberData)
    } else {
      // 否则获取当前用户信息
      console.log('[UserProfile] Fetching current user info (getMyInfo)')
      memberData = await getMyInfo()
      console.log('[UserProfile] getMyInfo returned:', memberData)
    }
    
    // 更新用户数据
    if (memberData) {
      console.log('[UserProfile] Processing member data...')
      
      // 格式化在线时长
      const onlineTimeSeconds = memberData.online_time || 0
      const hours = Math.floor(onlineTimeSeconds / 3600)
      const minutes = Math.floor((onlineTimeSeconds % 3600) / 60)
      let onlineTimeFormatted = '0小时'
      if (hours > 0) {
        onlineTimeFormatted = minutes > 0 ? `${hours}小时${minutes}分钟` : `${hours}小时`
      } else if (minutes > 0) {
        onlineTimeFormatted = `${minutes}分钟`
      }
      
      user.value = {
        id: memberData.id || memberData.ID || props.userId,
        name: memberData.nickname || memberData.Nickname || 'User',
        nickname: memberData.nickname || memberData.Nickname || 'User',
        email: memberData.email || memberData.Email || '',
        avatar: memberData.avatar || memberData.Avatar || null,
        role: memberData.role || memberData.Role || 'member',
        online: memberData.is_online || memberData.IsOnline || false,
        skills: memberData.skills || memberData.Skills || [],
        skill_details: memberData.skill_details || memberData.SkillDetails || [],
        bio: memberData.bio || memberData.Bio || '',
        stats: {
          solved: memberData.solved_challenges || 0,
          points: memberData.total_points || 0,
          rank: memberData.rank || 0,
          messages: memberData.message_count || 0,
          files: memberData.files_shared || 0,
          onlineTime: onlineTimeSeconds,
          onlineTimeFormatted: onlineTimeFormatted
        },
        activities: [],
        joinedAt: memberData.join_time ? new Date(memberData.join_time * 1000) : new Date()
      }
      
      console.log('[UserProfile] User data updated:', user.value)
      
      // 更新编辑数据
      editData.value = {
        nickname: user.value.nickname,
        email: user.value.email,
        skills: [...user.value.skills],
        skill_details: (user.value.skill_details && user.value.skill_details.length)
          ? JSON.parse(JSON.stringify(user.value.skill_details))
          : (user.value.skills || []).map(cat => ({ category: cat, level: 3, experience: 0 })),
        bio: user.value.bio
      }
      
      console.log('[UserProfile] Successfully loaded user profile')
    } else {
      console.warn('[UserProfile] memberData is null or undefined')
    }
  } catch (error) {
    console.error('[UserProfile] Failed to load user data:', error)
    console.error('[UserProfile] Error details:', {
      message: error.message,
      stack: error.stack
    })
    message.warning('加载用户资料失败，显示默认数据: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
    console.log('[UserProfile] Loading complete')
  }
}

watch(() => props.open, async (newVal) => {
  if (newVal) {
    editMode.value = false
    await loadUserData()
  }
})

const saveProfile = async () => {
  loading.value = true
  try {
    console.log('[UserProfile] Saving profile data:', editData.value)
    
    // 调用后端API更新用户配置
    const profileData = {
      nickname: editData.value.nickname,
      email: editData.value.email,
    skills: editData.value.skills,
    skill_details: editData.value.skill_details,
      bio: editData.value.bio
    }
    
    const result = await updateUserProfile(profileData)
    console.log('[UserProfile] updateUserProfile returned:', result)
    
    // 重新加载用户数据以确保同步
    await loadUserData()
    
    message.success('资料已保存')
    editMode.value = false
    emit('update', user.value)
  } catch (error) {
    console.error('[UserProfile] Failed to save profile:', error)
    message.error('保存失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
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

// 技能编辑相关
const addSkill = () => {
  if (!editData.value.skill_details) editData.value.skill_details = []
  editData.value.skill_details.push({ category: '', level: 3, experience: 0 })
}

const removeSkill = (index) => {
  if (editData.value.skill_details && index >= 0 && index < editData.value.skill_details.length) {
    editData.value.skill_details.splice(index, 1)
  }
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

/* 技能相关 */
.skill-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.experience-text {
  color: #888;
  font-size: 12px;
}

.skill-edit-item {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}
</style>


