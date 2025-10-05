import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'home',
    component: () => import('@/views/HomeView.vue'),
    meta: { title: 'CrossWire - CTF 团队通讯' }
  },
  {
    path: '/server',
    name: 'server',
    component: () => import('@/views/ServerView.vue'),
    meta: { title: '创建频道 - CrossWire' }
  },
  {
    path: '/client',
    name: 'client',
    component: () => import('@/views/ClientView.vue'),
    meta: { title: '加入频道 - CrossWire' }
  },
  {
    path: '/chat',
    name: 'chat',
    component: () => import('@/views/ChatView.vue'),
    meta: { title: '聊天室 - CrossWire' }
  },
  {
    path: '/challenges',
    name: 'challenges',
    component: () => import('@/views/ChallengeView.vue'),
    meta: { title: '题目管理 - CrossWire' }
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  document.title = to.meta.title || 'CrossWire'
  next()
})

export default router

