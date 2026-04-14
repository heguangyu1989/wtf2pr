<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
    <div class="w-full max-w-2xl max-h-[80vh] overflow-hidden rounded-lg bg-white dark:bg-gray-900 shadow-lg flex flex-col">
      <div class="flex items-center justify-between border-b px-4 py-3">
        <h2 class="text-base font-semibold">历史 Review</h2>
        <button class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" @click="$emit('close')">关闭</button>
      </div>

      <div class="flex-1 overflow-y-auto p-4 space-y-3">
        <div v-if="!reviews.length" class="text-sm text-gray-500 text-center py-8">暂无历史 Review</div>

        <div
          v-for="r in reviews"
          :key="r.reviewID"
          class="border rounded-lg p-3 hover:bg-gray-50 dark:hover:bg-gray-800 transition"
        >
          <div class="flex items-center justify-between">
            <div class="font-mono text-sm">{{ r.reviewID.substring(0, 8) }}</div>
            <div class="text-xs px-2 py-0.5 rounded bg-gray-100 dark:bg-gray-800">
              {{ r.type === 'commit' ? 'Commit' : 'Working' }}
            </div>
          </div>
          <div v-if="r.commit" class="text-xs text-gray-500 mt-1">
            Commit: {{ r.commit.substring(0, 7) }}
            <span v-if="r.commitMsg" class="ml-1 text-gray-400">— {{ r.commitMsg }}</span>
          </div>
          <div class="text-xs text-gray-500 mt-1">
            评论数: {{ r.commentCount }}
            <span v-if="r.updatedAt" class="ml-2">更新于: {{ formatTime(r.updatedAt) }}</span>
          </div>
          <div class="mt-2 flex items-center gap-2">
            <button
              v-if="r.type === 'commit'"
              class="px-2 py-1 text-xs rounded bg-blue-600 text-white hover:bg-blue-700"
              @click="$emit('switch', r)"
            >
              切换并编辑
            </button>
            <button
              class="px-2 py-1 text-xs rounded border hover:bg-gray-100 dark:hover:bg-gray-800"
              @click="$emit('view', r)"
            >
              查看详情
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  reviews: { type: Array, default: () => [] },
})
defineEmits(['close', 'switch', 'view'])

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  return d.toLocaleString()
}
</script>
