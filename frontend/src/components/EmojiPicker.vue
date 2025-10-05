<template>
  <a-popover
    v-model:open="visible"
    trigger="click"
    placement="topLeft"
    overlayClassName="emoji-picker-popover"
  >
    <template #content>
      <div class="emoji-picker">
        <div class="emoji-header">
          <SearchInput
            v-model="searchText"
            placeholder="搜索表情..."
            size="small"
            :debounce="200"
          />
        </div>

        <a-tabs v-model:activeKey="activeCategory" size="small" class="emoji-tabs">
          <a-tab-pane
            v-for="category in categories"
            :key="category.key"
            :tab="category.icon"
          >
            <div class="emoji-grid">
              <span
                v-for="emoji in getFilteredEmojis(category.key)"
                :key="emoji.char"
                class="emoji-item"
                :title="emoji.name"
                @click="selectEmoji(emoji)"
              >
                {{ emoji.char }}
              </span>
            </div>
          </a-tab-pane>
        </a-tabs>

        <div class="emoji-footer">
          <span class="recently-used-title">最近使用</span>
          <div class="recently-used">
            <span
              v-for="emoji in recentlyUsed"
              :key="emoji.char"
              class="emoji-item"
              @click="selectEmoji(emoji)"
            >
              {{ emoji.char }}
            </span>
            <span v-if="recentlyUsed.length === 0" class="no-recent">
              暂无
            </span>
          </div>
        </div>
      </div>
    </template>

    <slot>
      <a-button type="text">
        <SmileOutlined />
      </a-button>
    </slot>
  </a-popover>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { SmileOutlined } from '@ant-design/icons-vue'
import SearchInput from '@/components/Common/SearchInput.vue'
import {
  EMOJI_CATEGORIES,
  getAllEmojis,
  getEmojisByCategory,
  searchEmojis as searchEmojisData
} from '@/assets/emojis/emoji-data.js'

const emit = defineEmits(['select'])

const visible = ref(false)
const searchText = ref('')
const activeCategory = ref('smileys')
const recentlyUsed = ref([])

// 使用本地化的分类数据
const categories = EMOJI_CATEGORIES.filter(cat => cat.key !== 'frequent')

// 加载最近使用的表情（从 localStorage）
onMounted(() => {
  const stored = localStorage.getItem('crosswire_recent_emojis')
  if (stored) {
    try {
      recentlyUsed.value = JSON.parse(stored)
    } catch (e) {
      console.error('Failed to load recent emojis:', e)
    }
  }
})

// 使用本地化数据

const getFilteredEmojis = (category) => {
  let emojis = []
  
  if (searchText.value) {
    // 搜索模式：使用搜索函数
    emojis = searchEmojisData(searchText.value)
  } else {
    // 正常模式：按分类获取
    emojis = getEmojisByCategory(category)
  }
  
  return emojis
}

const selectEmoji = (emoji) => {
  emit('select', emoji.char)
  
  // 添加到最近使用
  const index = recentlyUsed.value.findIndex(e => e.char === emoji.char)
  if (index > -1) {
    recentlyUsed.value.splice(index, 1)
  }
  recentlyUsed.value.unshift(emoji)
  if (recentlyUsed.value.length > 18) {
    recentlyUsed.value.pop()
  }
  
  // 保存到 localStorage
  try {
    localStorage.setItem('crosswire_recent_emojis', JSON.stringify(recentlyUsed.value))
  } catch (e) {
    console.error('Failed to save recent emojis:', e)
  }
  
  visible.value = false
}
</script>

<style scoped>
.emoji-picker {
  width: 360px;
  max-height: 400px;
}

.emoji-header {
  padding: 8px 12px;
  border-bottom: 1px solid #f0f0f0;
}

.emoji-tabs :deep(.ant-tabs-nav) {
  margin: 0;
  padding: 8px 12px 0;
}

.emoji-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 4px;
  padding: 12px;
  max-height: 240px;
  overflow-y: auto;
}

.emoji-item {
  font-size: 24px;
  padding: 4px;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 36px;
}

.emoji-item:hover {
  background-color: #f5f5f5;
  transform: scale(1.2);
}

.emoji-footer {
  padding: 12px;
  border-top: 1px solid #f0f0f0;
}

.recently-used-title {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  display: block;
  margin-bottom: 8px;
}

.recently-used {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.recently-used .emoji-item {
  font-size: 20px;
  height: 32px;
  width: 32px;
}

.no-recent {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.25);
}
</style>

