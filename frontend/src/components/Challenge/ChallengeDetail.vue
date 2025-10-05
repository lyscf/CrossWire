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
              :color="sub.correct ? 'green' : 'red'"
            >
              <p>
                <strong>{{ sub.submitter }}</strong> 提交了 Flag
                {{ sub.correct ? '✓ 正确' : '✗ 错误' }}
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
      <ChallengeRoom :challenge="challenge" @back="showRoomDrawer = false" />
    </a-drawer>
  </div>
</template>

<script setup>
import { ref } from 'vue'
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

// 模拟进度数据
const progressData = ref([
  {
    memberId: 'user1',
    memberName: 'alice',
    progress: 80,
    summary: '已找到注入点，正在绕过 WAF',
    updatedAt: new Date(Date.now() - 300000)
  },
  {
    memberId: 'user2',
    memberName: 'bob',
    progress: 100,
    summary: '已获取 Flag',
    updatedAt: new Date(Date.now() - 600000)
  }
])

// 模拟提交历史
const submissions = ref([
  {
    id: '1',
    submitter: 'alice',
    correct: false,
    timestamp: '2025-10-05 10:30'
  }
])

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
  message.success(`已分配给 ${data.members.length} 个成员`)
  emit('assign', data)
}

const handleSubmitOk = (data) => {
  message.success('Flag 提交成功')
  emit('submit', data)
}

const handleProgressChange = (value) => {
  emit('update-progress', value)
}

const goToRoom = () => {
  showRoomDrawer.value = true
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
</style>

