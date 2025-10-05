<!--
  通用搜索输入框组件
  使用场景：全局搜索、文件搜索、Emoji搜索、成员搜索等
-->
<template>
  <div class="search-input-wrapper" :class="wrapperClass">
    <a-input-search
      v-model:value="searchValue"
      :placeholder="placeholder"
      :size="size"
      :allow-clear="allowClear"
      :loading="loading"
      :disabled="disabled"
      :maxlength="maxLength"
      :class="inputClass"
      @search="handleSearch"
      @change="handleChange"
      @focus="handleFocus"
      @blur="handleBlur"
      @keydown="handleKeyDown"
    >
      <template #prefix>
        <component :is="prefixIcon" v-if="prefixIcon" />
      </template>
      <template #enterButton v-if="showSearchButton">
        <a-button :type="buttonType">
          {{ buttonText }}
        </a-button>
      </template>
    </a-input-search>

    <!-- 搜索建议/结果下拉面板 -->
    <div
      v-if="showDropdown && (suggestions.length > 0 || $slots.dropdown)"
      class="search-dropdown"
      :class="dropdownClass"
      :style="dropdownStyle"
    >
      <slot name="dropdown" :suggestions="suggestions" :search-text="searchValue">
        <!-- 默认建议列表 -->
        <div v-if="suggestions.length > 0" class="suggestion-list">
          <div
            v-for="(item, index) in suggestions"
            :key="index"
            class="suggestion-item"
            :class="{ 'suggestion-active': activeIndex === index }"
            @click="handleSelectSuggestion(item, index)"
            @mouseenter="activeIndex = index"
          >
            <slot name="suggestion" :item="item" :index="index">
              <span>{{ getSuggestionText(item) }}</span>
            </slot>
          </div>
        </div>

        <!-- 空状态 -->
        <div v-else-if="searchValue && showEmpty" class="search-empty">
          <a-empty
            :image="Empty.PRESENTED_IMAGE_SIMPLE"
            :description="emptyText"
          />
        </div>
      </slot>

      <!-- 快捷键提示 -->
      <div v-if="showShortcuts && shortcuts.length > 0" class="search-shortcuts">
        <span
          v-for="shortcut in shortcuts"
          :key="shortcut.key"
          class="shortcut-item"
        >
          <kbd>{{ shortcut.key }}</kbd>
          <span>{{ shortcut.label }}</span>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { SearchOutlined } from '@ant-design/icons-vue'
import { Empty } from 'ant-design-vue'

const props = defineProps({
  // 基础属性
  modelValue: {
    type: String,
    default: ''
  },
  placeholder: {
    type: String,
    default: '搜索...'
  },
  size: {
    type: String,
    default: 'middle', // small | middle | large
    validator: (value) => ['small', 'middle', 'large'].includes(value)
  },
  allowClear: {
    type: Boolean,
    default: true
  },
  maxLength: {
    type: Number,
    default: 100
  },
  disabled: {
    type: Boolean,
    default: false
  },
  loading: {
    type: Boolean,
    default: false
  },

  // 图标和按钮
  prefixIcon: {
    type: Object,
    default: () => SearchOutlined
  },
  showSearchButton: {
    type: Boolean,
    default: false
  },
  buttonText: {
    type: String,
    default: '搜索'
  },
  buttonType: {
    type: String,
    default: 'primary'
  },

  // 下拉建议
  suggestions: {
    type: Array,
    default: () => []
  },
  suggestionKey: {
    type: String,
    default: 'label' // 建议项的显示字段
  },
  showDropdown: {
    type: Boolean,
    default: false
  },
  showEmpty: {
    type: Boolean,
    default: true
  },
  emptyText: {
    type: String,
    default: '没有找到结果'
  },

  // 快捷键
  showShortcuts: {
    type: Boolean,
    default: false
  },
  shortcuts: {
    type: Array,
    default: () => []
  },
  globalShortcut: {
    type: String,
    default: '' // 例如: 'ctrl+k'
  },

  // 样式
  wrapperClass: {
    type: String,
    default: ''
  },
  inputClass: {
    type: String,
    default: ''
  },
  dropdownClass: {
    type: String,
    default: ''
  },
  dropdownStyle: {
    type: Object,
    default: () => ({})
  },

  // 行为
  debounce: {
    type: Number,
    default: 0 // 防抖延迟（毫秒）
  },
  autoFocus: {
    type: Boolean,
    default: false
  },
  clearOnSelect: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits([
  'update:modelValue',
  'search',
  'change',
  'focus',
  'blur',
  'select',
  'clear',
  'keydown'
])

const searchValue = ref(props.modelValue)
const activeIndex = ref(0)
const debounceTimer = ref(null)

// 监听 v-model
watch(() => props.modelValue, (newVal) => {
  searchValue.value = newVal
})

// 监听搜索值变化
watch(searchValue, (newVal) => {
  emit('update:modelValue', newVal)
  
  if (props.debounce > 0) {
    clearTimeout(debounceTimer.value)
    debounceTimer.value = setTimeout(() => {
      emit('change', newVal)
    }, props.debounce)
  } else {
    emit('change', newVal)
  }
})

// 获取建议文本
const getSuggestionText = (item) => {
  if (typeof item === 'string') return item
  if (typeof item === 'object' && props.suggestionKey) {
    return item[props.suggestionKey] || ''
  }
  return String(item)
}

// 处理搜索
const handleSearch = () => {
  emit('search', searchValue.value)
}

// 处理变化
const handleChange = (e) => {
  // 已在 watch 中处理
}

// 处理聚焦
const handleFocus = (e) => {
  emit('focus', e)
}

// 处理失焦
const handleBlur = (e) => {
  emit('blur', e)
}

// 处理键盘事件
const handleKeyDown = (e) => {
  emit('keydown', e)

  // 如果有下拉建议，处理键盘导航
  if (props.showDropdown && props.suggestions.length > 0) {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        activeIndex.value = Math.min(activeIndex.value + 1, props.suggestions.length - 1)
        break
      case 'ArrowUp':
        e.preventDefault()
        activeIndex.value = Math.max(activeIndex.value - 1, 0)
        break
      case 'Enter':
        if (activeIndex.value >= 0 && activeIndex.value < props.suggestions.length) {
          e.preventDefault()
          handleSelectSuggestion(props.suggestions[activeIndex.value], activeIndex.value)
        }
        break
      case 'Escape':
        e.preventDefault()
        if (props.allowClear) {
          searchValue.value = ''
          emit('clear')
        }
        break
    }
  }
}

// 选择建议
const handleSelectSuggestion = (item, index) => {
  emit('select', item, index)
  
  if (props.clearOnSelect) {
    searchValue.value = ''
  } else {
    searchValue.value = getSuggestionText(item)
  }
  
  activeIndex.value = 0
}

// 全局快捷键
const handleGlobalShortcut = (e) => {
  if (!props.globalShortcut) return
  
  const keys = props.globalShortcut.toLowerCase().split('+')
  const isMatch = keys.every(key => {
    switch (key) {
      case 'ctrl':
      case 'control':
        return e.ctrlKey
      case 'shift':
        return e.shiftKey
      case 'alt':
        return e.altKey
      case 'meta':
      case 'cmd':
        return e.metaKey
      default:
        return e.key.toLowerCase() === key
    }
  })
  
  if (isMatch) {
    e.preventDefault()
    focus()
  }
}

// 公开方法
const focus = () => {
  const input = document.querySelector('.search-input-wrapper input')
  if (input) input.focus()
}

const clear = () => {
  searchValue.value = ''
  emit('clear')
}

const blur = () => {
  const input = document.querySelector('.search-input-wrapper input')
  if (input) input.blur()
}

// 生命周期
onMounted(() => {
  if (props.autoFocus) {
    focus()
  }
  
  if (props.globalShortcut) {
    document.addEventListener('keydown', handleGlobalShortcut)
  }
})

onUnmounted(() => {
  if (props.globalShortcut) {
    document.removeEventListener('keydown', handleGlobalShortcut)
  }
  
  if (debounceTimer.value) {
    clearTimeout(debounceTimer.value)
  }
})

// 暴露方法
defineExpose({
  focus,
  clear,
  blur
})
</script>

<style scoped>
.search-input-wrapper {
  position: relative;
  width: 100%;
  padding: 2px 0; /* 添加上下padding */
}

.search-dropdown {
  position: absolute;
  top: calc(100% + 6px); /* 增加间距以适应padding */
  left: 0;
  right: 0;
  background: white;
  border-radius: 4px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  max-height: 400px;
  overflow: hidden;
  z-index: 1000;
  display: flex;
  flex-direction: column;
}

.suggestion-list {
  flex: 1;
  overflow-y: auto;
}

.suggestion-item {
  padding: 8px 12px;
  cursor: pointer;
  transition: background-color 0.2s;
  color: rgba(0, 0, 0, 0.85);
}

.suggestion-item:hover,
.suggestion-active {
  background-color: #f5f5f5;
}

.search-empty {
  padding: 24px;
  text-align: center;
}

.search-shortcuts {
  display: flex;
  gap: 12px;
  padding: 8px 12px;
  background-color: #fafafa;
  border-top: 1px solid #f0f0f0;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.shortcut-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

kbd {
  display: inline-block;
  padding: 2px 6px;
  background-color: white;
  border: 1px solid #d9d9d9;
  border-radius: 2px;
  font-family: 'SFMono-Regular', 'Consolas', monospace;
  font-size: 11px;
  line-height: 1;
  box-shadow: 0 1px 0 rgba(0, 0, 0, 0.1);
}

/* 确保输入框 padding 正确 */
.search-input-wrapper :deep(.ant-input-affix-wrapper) {
  padding: 4px 11px;
}

.search-input-wrapper :deep(.ant-input-affix-wrapper-lg) {
  padding: 6.5px 11px;
}

.search-input-wrapper :deep(.ant-input-affix-wrapper-sm) {
  padding: 0px 7px;
}

.search-input-wrapper :deep(.ant-input-affix-wrapper > input.ant-input) {
  padding: 0;
}

/* 修复搜索按钮高度 */
.search-input-wrapper :deep(.ant-input-search-button) {
  height: 100% !important;
  display: flex;
  align-items: center;
}

.search-input-wrapper :deep(.ant-input-group-addon) {
  padding: 0;
}

.search-input-wrapper :deep(.ant-input-group-addon .ant-btn) {
  height: 100%;
  border-radius: 0 2px 2px 0;
}

/* 确保输入框组对齐 */
.search-input-wrapper :deep(.ant-input-group) {
  display: flex;
  align-items: stretch;
}

.search-input-wrapper :deep(.ant-input-group > .ant-input-affix-wrapper) {
  flex: 1;
}

/* 不同尺寸的按钮对齐 */
.search-input-wrapper :deep(.ant-input-search-large .ant-input-group-addon .ant-btn) {
  height: 40px;
  padding: 6.5px 15px;
  font-size: 16px;
}

.search-input-wrapper :deep(.ant-input-search-middle .ant-input-group-addon .ant-btn) {
  height: 32px;
  padding: 4px 15px;
  font-size: 14px;
}

.search-input-wrapper :deep(.ant-input-search-small .ant-input-group-addon .ant-btn) {
  height: 24px;
  padding: 0px 7px;
  font-size: 14px;
}
</style>

