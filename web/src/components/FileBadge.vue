<template>
  <span v-if="file.isNew" class="w-5 h-5 rounded text-xs flex items-center justify-center bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300 font-bold">+</span>
  <span v-else-if="file.isDeleted" class="w-5 h-5 rounded text-xs flex items-center justify-center bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300 font-bold">−</span>
  <span v-else class="w-5 h-5 rounded text-xs flex items-center justify-center bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300 font-bold">M</span>
</template>

<script setup>
defineProps({
  file: { type: Object, required: true },
})
</script>
