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
            :disabled="!isServerMode"
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
import { ref, computed, onMounted, onActivated, onBeforeUnmount } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import ChallengeList from '@/components/Challenge/ChallengeList.vue'
import ChallengeDetail from '@/components/Challenge/ChallengeDetail.vue'
import ChallengeCreate from '@/components/Challenge/ChallengeCreate.vue'
import { useChallengeStore } from '@/stores/challenge'
import { useAppStore } from '@/stores/app'
import { 
  getChallenges, 
  createChallenge, 
  assignChallenge, 
  submitFlag,
  getChallengeStats,
  updateChallengeProgress
} from '@/api/app'

const challengeStore = useChallengeStore()
const appStore = useAppStore()
const isServerMode = computed(() => (appStore.mode || '').toLowerCase() === 'server')
const isAdmin = ref(true) // 后续可从用户状态获取真实权限
const showCreateModal = ref(false)
const filterCategory = ref('all')
const filterStatus = ref('all')
const selectedChallengeId = ref(null)

// 真实数据（从后端加载）
const challenges = ref([])

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

const handleChallengeCreated = async (challenge) => {
  try {
    console.log('Creating challenge:', challenge)
    // 前端弹窗已直接调用 createChallenge；此处仅刷新
    message.success('题目创建成功')
    await loadChallenges()
    showCreateModal.value = false
  } catch (e) {
    console.error('Failed after create:', e)
  }
}

const handleAssign = async (members) => {
  if (!selectedChallengeId.value) return
  try {
    console.log('Assigning challenge to:', members)
    // 如果members是数组，循环分配给每个成员
    if (Array.isArray(members)) {
      for (const memberId of members) {
        await assignChallenge(selectedChallengeId.value, memberId)
      }
      message.success(`已分配给 ${members.length} 位成员`)
    } else {
      // 单个成员
      await assignChallenge(selectedChallengeId.value, members)
      message.success('分配成功')
    }
    await loadChallenges()
  } catch (e) {
    console.error('Failed to assign challenge:', e)
    message.error('分配失败: ' + (e.message || '未知错误'))
  }
}

const handleSubmit = async (payload) => {
  if (!selectedChallengeId.value) return
  const flag = typeof payload === 'string' ? payload : payload?.flag
  if (!flag) {
    message.warning('请输入 Flag')
    return
  }
  try {
    console.log('Submitting flag:', flag)
    const result = await submitFlag(selectedChallengeId.value, flag)
    message.success(result?.message || 'Flag 提交成功')
    await loadChallenges()
  } catch (e) {
    console.error('Failed to submit flag:', e)
    message.error('提交失败: ' + (e.message || '未知错误'))
  }
}

const handleUpdateProgress = (progress) => {
  message.info(`进度更新: ${progress}%`)
}

// 加载题目列表
const loadChallenges = async () => {
  try {
    console.log('[ChallengeView] Loading challenges from backend...')
    const list = await getChallenges()
    console.log('[ChallengeView] Loaded challenges count:', Array.isArray(list) ? list.length : 'N/A')
    if (Array.isArray(list)) {
      for (const ch of list) {
        console.log('[ChallengeView] item:', {
          id: ch?.id || ch?.ID,
          title: ch?.title || ch?.Title,
          status: ch?.status || ch?.Status,
          points: ch?.points || ch?.Points
        })
      }
    }
    
    if (Array.isArray(list)) {
      challenges.value = list.map(c => {
        const solvedByArr = c.solved_by || c.SolvedBy || []
        const isSolved = (c.is_solved ?? c.IsSolved) ?? (Array.isArray(solvedByArr) && solvedByArr.length > 0)
        let status = c.status || c.Status || 'pending'
        if (status === 'open') status = isSolved ? 'solved' : 'in_progress'
        return {
          id: c.id || c.ID,
          title: c.title || c.Title || '未命名题目',
          category: c.category || c.Category || 'Misc',
          difficulty: c.difficulty || c.Difficulty || 'Medium',
          points: c.points || c.Points || 100,
          status,
          isSolved: !!isSolved,
          assignedTo: c.assigned_to || c.AssignedTo || [],
          progress: c.progress || c.Progress || 0,
          solvedBy: solvedByArr,
          description: c.description || c.Description || '',
          flag: c.flag || c.Flag || ''
        }
      })
      console.log('[ChallengeView] Normalized challenges:', challenges.value)
    }
    
    if (challenges.value.length === 0) {
      console.log('No challenges found')
    }
  } catch (e) {
    console.error('Failed to load challenges:', e)
    message.warning('题目加载失败，请检查服务器状态')
  }
}

// 组件挂载与激活时加载数据
let onWinEvt
onMounted(async () => {
  console.log('ChallengeView mounted')
  await loadChallenges()
  // 监听全局挑战事件，实时刷新列表
  onWinEvt = (e) => {
    const { type } = (e?.detail || {})
    if (type === 'challenge:created' || type === 'challenge:updated' || type === 'challenge:assigned' || type === 'challenge:solved') {
      loadChallenges()
    }
  }
  window.addEventListener('cw:challenge:event', onWinEvt)
})

onActivated(async () => {
  await loadChallenges()
})

onBeforeUnmount(() => {
  if (onWinEvt) window.removeEventListener('cw:challenge:event', onWinEvt)
})
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
  padding: var(--spacing-md);
  border-bottom: 1px solid #f0f0f0;
}

.sider-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.filter-section {
  padding: var(--spacing-md);
  border-bottom: 1px solid #f0f0f0;
}

.challenge-content {
  background: white;
  padding: var(--spacing-lg);
  overflow-y: auto;
}
</style>

