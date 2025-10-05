import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useMemberStore = defineStore('member', () => {
  // 成员列表
  const members = ref([])

  // Getters
  const onlineMembers = computed(() => {
    return members.value.filter(m => m.status !== 'offline')
  })

  const offlineMembers = computed(() => {
    return members.value.filter(m => m.status === 'offline')
  })

  const memberCount = computed(() => members.value.length)

  const onlineCount = computed(() => onlineMembers.value.length)

  const getMemberById = computed(() => (memberId) => {
    return members.value.find(m => m.id === memberId)
  })

  const getMembersBySkill = computed(() => (skill) => {
    return members.value.filter(m => 
      m.skills && m.skills.includes(skill)
    )
  })

  // Actions
  const setMembers = (newMembers) => {
    members.value = newMembers
  }

  const addMember = (member) => {
    const exists = members.value.find(m => m.id === member.id)
    if (!exists) {
      members.value.push(member)
    }
  }

  const removeMember = (memberId) => {
    const index = members.value.findIndex(m => m.id === memberId)
    if (index !== -1) {
      members.value.splice(index, 1)
    }
  }

  const updateMember = (memberId, updates) => {
    const index = members.value.findIndex(m => m.id === memberId)
    if (index !== -1) {
      members.value[index] = { ...members.value[index], ...updates }
    }
  }

  const updateMemberStatus = (memberId, status) => {
    updateMember(memberId, { status })
  }

  const updateMemberTask = (memberId, task) => {
    updateMember(memberId, { currentTask: task })
  }

  const reset = () => {
    members.value = []
  }

  return {
    members,
    onlineMembers,
    offlineMembers,
    memberCount,
    onlineCount,
    getMemberById,
    getMembersBySkill,
    setMembers,
    addMember,
    removeMember,
    updateMember,
    updateMemberStatus,
    updateMemberTask,
    reset
  }
})

