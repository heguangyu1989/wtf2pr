<template>
  <header class="border-b dark:border-gray-800 bg-white/90 dark:bg-gray-900/90 backdrop-blur sticky top-0 z-10 shadow-sm">
    <div class="max-w-7xl mx-auto px-4 py-2.5 flex flex-wrap items-center gap-3">
      <div class="flex items-center gap-2">
        <div class="w-7 h-7 rounded-lg bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center text-white text-xs font-bold shadow">
          w2
        </div>
        <h1 class="text-base font-semibold tracking-tight">wtf2pr</h1>
      </div>

      <button
        class="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 px-2.5 py-1.5 rounded-lg transition max-w-[16rem]"
        :title="workDir || '选择工作目录'"
        @click="$emit('switchWorkdir')"
      >
        <svg class="w-3.5 h-3.5 text-amber-500 shrink-0" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
        </svg>
        <span class="truncate font-mono">{{ workDirName || '选择目录' }}</span>
        <svg class="w-3 h-3 text-gray-400 shrink-0" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9" />
        </svg>
      </button>

      <div class="flex-1"></div>

      <div class="flex items-center gap-2 text-sm">
        <div v-if="reviewLabel" class="text-xs text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded-md font-mono">
          {{ reviewLabel }}
        </div>
        <div
          class="text-xs px-2 py-1 rounded-md font-medium"
          :class="saved ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'"
        >
          {{ saved ? '已保存' : '未保存' }}
        </div>
        <button class="px-3 py-1.5 border dark:border-gray-700 text-sm rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition" @click="$emit('showHelp')">帮助</button>
        <button class="px-3 py-1.5 bg-purple-600 text-white text-sm rounded-lg hover:bg-purple-700 transition shadow-sm" @click="$emit('newReview')">新建 Review</button>
        <select
          :value="exportFormat"
          class="border dark:border-gray-700 rounded-lg px-2 py-1.5 text-sm bg-white dark:bg-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
          @change="$emit('update:exportFormat', $event.target.value)"
        >
          <option value="markdown">Markdown</option>
          <option value="json">JSON</option>
          <option value="xml">XML</option>
          <option value="template">Template</option>
        </select>
        <button
          class="px-3 py-1.5 border dark:border-gray-700 text-sm rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition disabled:opacity-40 disabled:cursor-not-allowed"
          :disabled="!canExport"
          :title="canExport ? '' : '仅在 Working tree / Commit 页可导出'"
          @click="$emit('doExport')"
        >导出</button>
        <button
          class="px-3 py-1.5 bg-green-600 text-white text-sm rounded-lg hover:bg-green-700 transition shadow-sm disabled:opacity-40 disabled:cursor-not-allowed"
          :disabled="!canSave"
          :title="canSave ? '' : '仅在 Commit 页可保存 Review'"
          @click="$emit('saveReview')"
        >保存 Review</button>
      </div>
    </div>
  </header>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  saved: { type: Boolean, default: true },
  exportFormat: { type: String, default: 'markdown' },
  reviewLabel: { type: String, default: '' },
  workDir: { type: String, default: '' },
  activeTab: { type: String, default: 'working' },
})
defineEmits(['saveReview', 'doExport', 'newReview', 'showHelp', 'update:exportFormat', 'switchWorkdir'])

const workDirName = computed(() => {
  if (!props.workDir) return ''
  const parts = props.workDir.replace(/\/+$/, '').split('/')
  return parts[parts.length - 1] || props.workDir
})

const canExport = computed(() => props.activeTab === 'working' || props.activeTab === 'commit')
const canSave = computed(() => props.activeTab === 'commit')
</script>
