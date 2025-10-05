import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useChallengeStore = defineStore('challenge', () => {
  // 题目列表
  const challenges = ref([])

  // 题目分配关系
  const assignments = ref({})

  // 题目进度
  const progress = ref({})

  // 提交历史
  const submissions = ref([])

  // Getters
  const getChallengeById = computed(() => (challengeId) => {
    return challenges.value.find(c => c.id === challengeId)
  })

  const getChallengesByCategory = computed(() => (category) => {
    return challenges.value.filter(c => c.category === category)
  })

  const getChallengesByStatus = computed(() => (status) => {
    return challenges.value.filter(c => c.status === status)
  })

  const getAssignedChallenges = computed(() => (memberId) => {
    return challenges.value.filter(c => 
      c.assignedTo && c.assignedTo.includes(memberId)
    )
  })

  const getSolvedCount = computed(() => {
    return challenges.value.filter(c => c.status === 'solved').length
  })

  const getTotalPoints = computed(() => {
    return challenges.value.reduce((sum, c) => sum + c.points, 0)
  })

  const getSolvedPoints = computed(() => {
    return challenges.value
      .filter(c => c.status === 'solved')
      .reduce((sum, c) => sum + c.points, 0)
  })

  // Actions
  const setChallenges = (newChallenges) => {
    challenges.value = newChallenges
  }

  const addChallenge = (challenge) => {
    const exists = challenges.value.find(c => c.id === challenge.id)
    if (!exists) {
      challenges.value.push(challenge)
    }
  }

  const updateChallenge = (challengeId, updates) => {
    const index = challenges.value.findIndex(c => c.id === challengeId)
    if (index !== -1) {
      challenges.value[index] = {
        ...challenges.value[index],
        ...updates
      }
    }
  }

  const removeChallenge = (challengeId) => {
    const index = challenges.value.findIndex(c => c.id === challengeId)
    if (index !== -1) {
      challenges.value.splice(index, 1)
    }
  }

  const assignChallenge = (challengeId, memberIds) => {
    updateChallenge(challengeId, {
      assignedTo: memberIds,
      status: 'in_progress'
    })
    
    // 记录分配关系
    assignments.value[challengeId] = {
      members: memberIds,
      assignedAt: new Date().toISOString()
    }
  }

  const updateProgress = (challengeId, progressValue) => {
    updateChallenge(challengeId, { progress: progressValue })
    
    // 记录进度
    if (!progress.value[challengeId]) {
      progress.value[challengeId] = []
    }
    progress.value[challengeId].push({
      value: progressValue,
      timestamp: new Date().toISOString()
    })
  }

  const submitFlag = (challengeId, memberId, flag) => {
    const challenge = getChallengeById.value(challengeId)
    const correct = challenge && challenge.flag === flag
    
    const submission = {
      id: Date.now().toString(),
      challengeId,
      memberId,
      flag,
      correct,
      timestamp: new Date().toISOString()
    }
    
    submissions.value.push(submission)
    
    if (correct) {
      updateChallenge(challengeId, {
        status: 'solved',
        progress: 100,
        solvedBy: memberId,
        solvedAt: new Date().toISOString()
      })
    }
    
    return correct
  }

  const reset = () => {
    challenges.value = []
    assignments.value = {}
    progress.value = {}
    submissions.value = []
  }

  return {
    challenges,
    assignments,
    progress,
    submissions,
    getChallengeById,
    getChallengesByCategory,
    getChallengesByStatus,
    getAssignedChallenges,
    getSolvedCount,
    getTotalPoints,
    getSolvedPoints,
    setChallenges,
    addChallenge,
    updateChallenge,
    removeChallenge,
    assignChallenge,
    updateProgress,
    submitFlag,
    reset
  }
})

