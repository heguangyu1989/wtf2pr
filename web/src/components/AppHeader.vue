<template>
  <header class="border-b bg-white dark:bg-gray-900 sticky top-0 z-10">
    <div class="max-w-7xl mx-auto px-4 py-3 flex flex-wrap items-center gap-3">
      <h1 class="text-lg font-semibold">wtf2pr</h1>
      <div class="flex-1"></div>
      <div class="flex items-center gap-3 text-sm">
        <div v-if="reviewLabel" class="text-xs text-gray-500 bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
          {{ reviewLabel }}
        </div>
        <div
          class="text-xs px-2 py-1 rounded"
          :class="saved ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300'"
        >
          {{ saved ? '已保存' : '未保存' }}
        </div>
        <button class="px-3 py-1 border text-sm rounded hover:bg-gray-50 dark:hover:bg-gray-800" @click="$emit('showHelp')">帮助</button>
        <button class="px-3 py-1 bg-purple-600 text-white text-sm rounded hover:bg-purple-700" @click="$emit('newReview')">新建 Review</button>
        <select
          :value="exportFormat"
          class="border rounded px-2 py-1 text-sm bg-white dark:bg-gray-900"
          @change="$emit('update:exportFormat', $event.target.value)"
        >
          <option value="markdown">Markdown</option>
          <option value="json">JSON</option>
          <option value="xml">XML</option>
        </select>
        <button class="px-3 py-1 border text-sm rounded hover:bg-gray-50 dark:hover:bg-gray-800" @click="$emit('doExport')">导出</button>
        <button class="px-3 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700" @click="$emit('saveReview')">保存 Review</button>
      </div>
    </div>
  </header>
</template>

<script setup>
defineProps({
  saved: { type: Boolean, default: true },
  exportFormat: { type: String, default: 'markdown' },
  reviewLabel: { type: String, default: '' },
})
defineEmits(['saveReview', 'doExport', 'newReview', 'showHelp', 'update:exportFormat'])
</script>
