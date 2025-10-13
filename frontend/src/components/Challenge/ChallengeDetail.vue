<template>
  <div class="challenge-detail">
    <!-- 头部 -->
    <div class="detail-header">
      <div class="header-left">
        <h2 class="detail-title">{{ challenge.title }}</h2>
        <a-space>
          <a-tag :color="getCategoryColor(challenge.category)">
            {{ challenge.category }}
          </a-tag>
          <a-tag :color="getDifficultyColor(challenge.difficulty)">
            {{ challenge.difficulty }}
          </a-tag>
          <span class="points">
            <TrophyOutlined /> {{ challenge.points }} 分
          </span>
        </a-space>
      </div>
      <div class="header-right">
        <a-space>
          <a-button @click="showAssignModal = true">
            <UserAddOutlined /> 分配题目
          </a-button>
          <a-button @click="showProgressModal = true">
            <LineChartOutlined /> 查看进度
          </a-button>
          <a-button @click="goToRoom">
            <WechatOutlined /> 讨论室
          </a-button>
          <a-button type="primary" @click="showSubmitModal = true">
            <SendOutlined /> 提交 Flag
          </a-button>
        </a-space>
      </div>
    </div>

    <a-divider />

    <!-- 题目信息 -->
    <div class="detail-content">
      <!-- 状态与 Flag -->
      <div class="section">
        <h3 class="section-title">解题状态</h3>
        <div class="section-content">
          <a-space>
            <a-tag :color="challenge.isSolved ? 'success' : 'default'">
              {{ challenge.isSolved ? '已解出' : '未解出' }}
            </a-tag>
            <template v-if="displayFlag">
              <span>Flag:</span>
              <code class="flag-code">{{ displayFlag }}</code>
              <a-button size="small" @click="copyFlag">复制</a-button>
            </template>
          </a-space>
        </div>
      </div>
      <!-- 描述 -->
      <div class="section">
        <h3 class="section-title">题目描述</h3>
        <div class="section-content">
          <p>{{ challenge.description || '暂无描述' }}</p>
        </div>
      </div>

      <!-- 分配信息 -->
      <div class="section">
        <h3 class="section-title">分配信息</h3>
        <div class="section-content">
          <a-space v-if="challenge.assignedTo && challenge.assignedTo.length > 0">
            <a-avatar-group>
              <a-avatar
                v-for="member in challenge.assignedTo"
                :key="member"
              >
                {{ member[0] }}
              </a-avatar>
            </a-avatar-group>
            <span>{{ challenge.assignedTo.join(', ') }}</span>
          </a-space>
          <a-empty
            v-else
            :image="Empty.PRESENTED_IMAGE_SIMPLE"
            description="暂未分配"
          />
        </div>
      </div>

      <!-- 进度 -->
      <div class="section">
        <h3 class="section-title">解题进度</h3>
        <div class="section-content">
          <a-progress
            :percent="challenge.progress"
            :status="challenge.status === 'solved' ? 'success' : 'normal'"
          />
          <div style="margin-top: 12px">
            <a-slider
              v-model:value="localProgress"
              @change="handleProgressChange"
            />
          </div>
        </div>
      </div>

      <!-- 提交历史 -->
      <div class="section">
        <h3 class="section-title">提交历史</h3>
        <div class="section-content">
          <a-timeline v-if="submissions.length > 0">
            <a-timeline-item
              v-for="sub in submissions"
              :key="sub.id"
              color="blue"
            >
              <p>
                <strong>{{ sub.submitter }}</strong> 提交了 Flag: <code>{{ sub.flag }}</code>
              </p>
              <p class="timeline-time">{{ sub.timestamp }}</p>
            </a-timeline-item>
          </a-timeline>
          <a-empty
            v-else
            :image="Empty.PRESENTED_IMAGE_SIMPLE"
            description="暂无提交记录"
          />
        </div>
      </div>
    </div>

    <!-- 分配题目弹窗 -->
    <ChallengeAssign
      v-model:open="showAssignModal"
      :challenge="challenge"
      @assign="handleAssignOk"
    />

    <!-- 提交 Flag 弹窗 -->
    <ChallengeSubmit
      v-model:open="showSubmitModal"
      :challenge="challenge"
      @submit="handleSubmitOk"
    />

    <!-- 进度查看弹窗 -->
    <a-modal
      v-model:open="showProgressModal"
      title="题目进度"
      width="800px"
      :footer="null"
    >
      <ChallengeProgress :progress-data="progressData" />
    </a-modal>

    <!-- 题目讨论室 -->
    <a-drawer
      v-model:open="showRoomDrawer"
      title="题目讨论室"
      placement="right"
      width="80%"
      :body-style="{ padding: 0 }"
    >
      <ChallengeRoom :challenge="challenge" @back="showRoomDrawer = false" @refresh="onRoomRefresh" />
    </a-drawer>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue'
import { Empty, message } from 'ant-design-vue'
import {
  TrophyOutlined,
  UserAddOutlined,
  SendOutlined,
  LineChartOutlined,
  WechatOutlined
} from '@ant-design/icons-vue'
import ChallengeAssign from './ChallengeAssign.vue'
import ChallengeSubmit from './ChallengeSubmit.vue'
import ChallengeProgress from './ChallengeProgress.vue'
import ChallengeRoom from './ChallengeRoom.vue'
import { getChallengeProgress, getChallengeSubmissions } from '@/api/app'

const props = defineProps({
  challenge: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['assign', 'submit', 'update-progress'])

const showAssignModal = ref(false)
const showSubmitModal = ref(false)
const showProgressModal = ref(false)
const showRoomDrawer = ref(false)
const localProgress = ref(props.challenge.progress || 0)

// 真实数据（从后端加载）
const progressData = ref([])
const submissions = ref([])
const loading = ref(false)
const displayFlag = computed(() => {
  // 优先使用挑战对象上的 flag；否则用最新一次提交中的 flag
  if (props.challenge?.flag) return props.challenge.flag
  if (Array.isArray(submissions.value) && submissions.value.length > 0) {
    return submissions.value[0].flag || ''
  }
  return ''
})

// 加载题目进度数据
const loadProgressData = async () => {
  if (!props.challenge?.id) return
  loading.value = true
  try {
    const data = await getChallengeProgress(props.challenge.id)
    console.log('Loaded challenge progress:', data)
    if (Array.isArray(data)) {
      progressData.value = data.map(p => ({
        memberName: p.member_name || p.MemberName || p.member || p.Member || 'Unknown',
        progress: p.progress || 0,
        status: p.status || 'pending',
        updatedAt: p.updated_at ? new Date(p.updated_at * 1000) : (p.last_update ? new Date(p.last_update * 1000) : new Date()),
        summary: p.summary || ''
      }))
    } else if (data && typeof data === 'object') {
      const p = data
      progressData.value = [{
        memberName: p.member_name || p.member || p.MemberName || p.Member || '我',
        progress: p.progress || 0,
        status: p.status || 'pending',
        updatedAt: (p.updated_at || p.last_update) ? new Date((p.updated_at || p.last_update) * 1000) : new Date(),
        summary: p.summary || ''
      }]
    }
  } catch (error) {
    console.error('Failed to load progress data:', error)
  } finally {
    loading.value = false
  }
}

// 加载提交记录
const loadSubmissions = async () => {
  if (!props.challenge?.id) return
  loading.value = true
  try {
    const data = await getChallengeSubmissions(props.challenge.id)
    console.log('Loaded submissions:', data)
    if (Array.isArray(data)) {
      submissions.value = data.map(s => {
        const t = s.submitted_at || s.SubmittedAt
        let ts
        if (typeof t === 'number') {
          ts = new Date(t * 1000)
        } else if (typeof t === 'string') {
          const d = new Date(t)
          ts = isNaN(d.getTime()) ? new Date() : d
        } else if (t instanceof Date) {
          ts = t
        } else {
          ts = new Date()
        }
        return {
          id: s.id || s.ID,
          submitter: s.member_name || s.MemberName || s.member_id || s.MemberID || 'Unknown',
          flag: s.flag || s.Flag || '',
          timestamp: ts
        }
      })
    }
  } catch (error) {
    console.error('Failed to load submissions:', error)
  } finally {
    loading.value = false
  }
}

// 组件挂载时加载数据
onMounted(() => {
  loadProgressData()
  loadSubmissions()
})

// 当题目变化时重新加载数据
watch(() => props.challenge?.id, () => {
  if (props.challenge?.id) {
    loadProgressData()
    loadSubmissions()
  }
})

const getCategoryColor = (category) => {
  const colors = {
    Web: 'blue',
    Pwn: 'red',
    Reverse: 'purple',
    Crypto: 'orange',
    Misc: 'green'
  }
  return colors[category] || 'default'
}

const getDifficultyColor = (difficulty) => {
  const colors = {
    Easy: 'success',
    Medium: 'warning',
    Hard: 'error',
    Insane: 'purple'
  }
  return colors[difficulty] || 'default'
}

const handleAssignOk = (data) => {
  const memberIds = Array.isArray(data?.members) ? data.members : []
  if (memberIds.length > 0) {
    message.success(`已分配给 ${memberIds.length} 个成员`)
  }
  emit('assign', memberIds)
}

const handleSubmitOk = (data) => {
  message.success('Flag 提交成功')
  emit('submit', data)
}

const handleProgressChange = (value) => {
  emit('update-progress', value)
}

const goToRoom = () => {
  // 如果有子频道ID，则切换 ChatView 的当前频道；否则回退为抽屉
  const subId = props.challenge?.sub_channel_id || props.challenge?.SubChannelID
  if (subId) {
    // 通过 hash 路由附带参数，ChatView 可读取并切换
    window.location.hash = `#/chat?channel=${encodeURIComponent(subId)}`
  } else {
    showRoomDrawer.value = true
  }
}

const copyFlag = async () => {
  try {
    const flag = displayFlag.value
    if (!flag) {
      message.warning('无可复制的 Flag')
      return
    }
    await navigator.clipboard.writeText(flag)
    message.success('Flag 已复制到剪贴板')
  } catch (e) {
    message.error('复制失败')
  }
}

const onRoomRefresh = () => {
  // 重新加载进度与提交记录，让 UI 反映“已解出/flag”
  loadProgressData()
  loadSubmissions()
}
</script>

<style scoped>
.challenge-detail {
  max-width: 900px;
  margin: 0 auto;
}

.detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;
}

.header-left {
  flex: 1;
}

.detail-title {
  margin: 0 0 12px 0;
  font-size: 24px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
}

.points {
  color: rgba(0, 0, 0, 0.65);
  font-size: 14px;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.section-title {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
}

.section-content {
  color: rgba(0, 0, 0, 0.65);
  line-height: 1.6;
}

.timeline-time {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  margin: 4px 0 0 0;
}

.flag-code {
  font-family: 'Consolas', monospace;
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
}
</style>

