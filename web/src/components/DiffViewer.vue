<template>
  <div v-if="loading" class="text-sm text-gray-500">加载中...</div>
  <div v-else-if="error" class="text-sm text-red-600">{{ error }}</div>
  <div v-else-if="!diffData || !diffData.files.length" class="text-sm text-gray-500">暂无 diff 数据</div>
  <div v-else class="grid grid-cols-1 lg:grid-cols-4 gap-4 h-full">
    <aside class="lg:col-span-1 h-full overflow-hidden">
      <div class="border rounded-lg bg-white dark:bg-gray-900 overflow-hidden h-full flex flex-col">
        <div class="px-3 py-2 border-b bg-gray-100 dark:bg-gray-800 text-sm font-medium shrink-0">
          文件 ({{ diffData.files.length }})
        </div>
        <ul class="overflow-y-auto flex-1">
          <li
            v-for="(f, i) in diffData.files"
            :key="i"
            class="px-3 py-2 text-sm border-b last:border-b-0 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800"
            :class="{ 'bg-blue-50 dark:bg-blue-900/20': selectedIndex === i }"
            @click="selectedIndex = i"
          >
            <div class="truncate font-mono text-xs">{{ f.newFile || f.oldFile }}</div>
            <div class="text-xs text-gray-500 mt-1">
              <span v-if="f.isNew" class="text-green-600">新增</span>
              <span v-else-if="f.isDeleted" class="text-red-600">删除</span>
              <span v-else>修改</span>
            </div>
          </li>
        </ul>
      </div>
    </aside>

    <section class="lg:col-span-3 h-full overflow-y-auto space-y-4 pr-1">
      <div v-if="commitInfo" class="border rounded-lg p-3 bg-white dark:bg-gray-900 text-sm space-y-1 shrink-0">
        <div><span class="text-gray-500">Commit:</span> {{ commitInfo.hash }}</div>
        <div><span class="text-gray-500">Author:</span> {{ commitInfo.author }}</div>
        <div><span class="text-gray-500">Date:</span> {{ commitInfo.date }}</div>
        <div><span class="text-gray-500">Message:</span> {{ commitInfo.message }}</div>
      </div>

      <ReadonlyDiffFile
        v-if="readonly && selectedFile"
        :file="selectedFile"
        :comments="comments"
      />
      <DiffFile
        v-else-if="selectedFile"
        :file="selectedFile"
        :comments="comments"
        @update:comments="$emit('update:comments', $event)"
      />
    </section>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import DiffFile from './DiffFile.vue'
import ReadonlyDiffFile from './ReadonlyDiffFile.vue'

const props = defineProps({
  diffData: { type: Object, default: null },
  comments: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  error: { type: String, default: '' },
  readonly: { type: Boolean, default: false },
  commitInfo: { type: Object, default: null },
})
defineEmits(['update:comments'])

const selectedIndex = ref(0)
const selectedFile = computed(() => props.diffData?.files[selectedIndex.value] || null)

watch(() => props.diffData, () => {
  selectedIndex.value = 0
}, { immediate: false })
</script>
