<template>
  <div class="challenge-container">
    <a-layout style="height: 100vh">
      <!-- 题目列表侧边栏 -->
      <a-layout-sider
        width="320"
        theme="light"
        class="challenge-sider"
      >
        <div class="sider-header">
          <h3 class="sider-title">题目管理</h3>
          <a-button
            v-if="isAdmin"
            type="primary"
            size="small"
            @click="showCreateModal = true"
          >
            <PlusOutlined /> 创建题目
          </a-button>
        </div>

        <!-- 筛选和统计 -->
        <div class="filter-section">
          <a-space direction="vertical" style="width: 100%">
            <a-select
              v-model:value="filterCategory"
              style="width: 100%"
              placeholder="题目类型"
            >
              <a-select-option value="all">全部类型</a-select-option>
              <a-select-option value="Web">Web</a-select-option>
              <a-select-option value="Pwn">Pwn</a-select-option>
              <a-select-option value="Reverse">Reverse</a-select-option>
              <a-select-option value="Crypto">Crypto</a-select-option>
              <a-select-option value="Misc">Misc</a-select-option>
            </a-select>

            <a-select
              v-model:value="filterStatus"
              style="width: 100%"
              placeholder="状态"
            >
              <a-select-option value="all">全部状态</a-select-option>
              <a-select-option value="pending">待分配</a-select-option>
              <a-select-option value="in_progress">进行中</a-select-option>
              <a-select-option value="solved">已解决</a-select-option>
            </a-select>
          </a-space>
        </div>

        <!-- 题目列表 -->
        <ChallengeList
          :challenges="filteredChallenges"
          :selected-id="selectedChallengeId"
          @select="handleSelectChallenge"
        />
      </a-layout-sider>

      <!-- 主内容区 -->
      <a-layout>
        <a-layout-content class="challenge-content">
          <ChallengeDetail
            v-if="selectedChallenge"
            :challenge="selectedChallenge"
            @assign="handleAssign"
            @submit="handleSubmit"
            @update-progress="handleUpdateProgress"
          />
          <a-empty
            v-else
            description="请选择一个题目"
            style="margin-top: 100px"
          />
        </a-layout-content>
      </a-layout>
    </a-layout>

    <!-- 创建题目弹窗 -->
    <ChallengeCreate
      v-model:open="showCreateModal"
      @created="handleChallengeCreated"
    />
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import ChallengeList from '@/components/Challenge/ChallengeList.vue'
import ChallengeDetail from '@/components/Challenge/ChallengeDetail.vue'
import ChallengeCreate from '@/components/Challenge/ChallengeCreate.vue'
import { useChallengeStore } from '@/stores/challenge'

const challengeStore = useChallengeStore()
const isAdmin = ref(true) // TODO: 从用户状态获取
const showCreateModal = ref(false)
const filterCategory = ref('all')
const filterStatus = ref('all')
const selectedChallengeId = ref(null)

// 模拟数据
const challenges = ref([
  {
    id: '1',
    title: 'SQL注入登录绕过',
    category: 'Web',
    difficulty: 'Easy',
    points: 100,
    status: 'in_progress',
    assignedTo: ['alice', 'bob'],
    progress: 60,
    solvedBy: null
  },
  {
    id: '2',
    title: '栈溢出提权',
    category: 'Pwn',
    difficulty: 'Medium',
    points: 200,
    status: 'in_progress',
    assignedTo: ['charlie'],
    progress: 30,
    solvedBy: null
  },
  {
    id: '3',
    title: 'RSA 加密破解',
    category: 'Crypto',
    difficulty: 'Hard',
    points: 300,
    status: 'pending',
    assignedTo: [],
    progress: 0,
    solvedBy: null
  }
])

const filteredChallenges = computed(() => {
  return challenges.value.filter(c => {
    if (filterCategory.value !== 'all' && c.category !== filterCategory.value) {
      return false
    }
    if (filterStatus.value !== 'all' && c.status !== filterStatus.value) {
      return false
    }
    return true
  })
})

const selectedChallenge = computed(() => {
  return challenges.value.find(c => c.id === selectedChallengeId.value)
})

const handleSelectChallenge = (id) => {
  selectedChallengeId.value = id
}

const handleChallengeCreated = (challenge) => {
  challenges.value.push(challenge)
  message.success('题目创建成功')
}

const handleAssign = (members) => {
  message.success(`已分配给 ${members.join(', ')}`)
}

const handleSubmit = (flag) => {
  message.info(`提交 Flag: ${flag}`)
}

const handleUpdateProgress = (progress) => {
  message.info(`进度更新: ${progress}%`)
}
</script>

<style scoped>
.challenge-container {
  width: 100%;
  height: 100vh;
  background-color: #f5f5f5;
}

.challenge-sider {
  background: white;
  border-right: 1px solid #f0f0f0;
  display: flex;
  flex-direction: column;
}

.sider-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.sider-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.filter-section {
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.challenge-content {
  background: white;
  padding: 24px;
  overflow-y: auto;
}
</style>

