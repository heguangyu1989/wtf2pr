<template>
  <div class="text-sm bg-white/60 dark:bg-gray-900/40 rounded-lg px-3 py-2 border border-amber-200/50 dark:border-amber-900/30">
    <div class="text-gray-500 text-xs mb-1 flex items-center justify-between">
      <span class="font-mono">{{ comment.lineKey ? `Line ${comment.lineKey}` : 'File comment' }}</span>
      <button v-if="removable" class="text-red-500 hover:text-red-600 hover:underline" @click="$emit('remove', comment.id)">删除</button>
    </div>
    <div class="whitespace-pre-wrap">{{ comment.content }}</div>
  </div>
</template>

<script setup>
defineProps({
  comment: { type: Object, required: true },
  removable: { type: Boolean, default: false },
})
defineEmits(['remove'])
</script>
