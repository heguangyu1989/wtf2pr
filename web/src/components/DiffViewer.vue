<template>
  <div v-if="loading" class="text-sm text-gray-500">加载中...</div>
  <div v-else-if="error" class="text-sm text-red-600">{{ error }}</div>
  <div v-else-if="!diffData || !diffData.files.length" class="text-sm text-gray-500">暂无 diff 数据</div>
  <div v-else class="grid grid-cols-1 lg:grid-cols-4 gap-4 h-full">
    <aside class="lg:col-span-1 h-full overflow-hidden">
      <div class="border rounded-lg bg-white dark:bg-gray-900 overflow-hidden h-full flex flex-col">
        <div class="px-3 py-2 border-b bg-gray-100 dark:bg-gray-800 text-sm font-medium shrink-0 flex items-center justify-between">
          <div>文件 ({{ diffData.files.length }})</div>
          <div v-if="totalFilePages > 1" class="flex items-center gap-1 text-xs">
            <button
              class="px-1.5 py-0.5 border rounded disabled:opacity-40 hover:bg-gray-50 dark:hover:bg-gray-800"
              :disabled="filePage <= 1"
              @click="filePage--"
            >
              ←
            </button>
            <span class="text-gray-500 whitespace-nowrap">{{ filePage }} / {{ totalFilePages }}</span>
            <button
              class="px-1.5 py-0.5 border rounded disabled:opacity-40 hover:bg-gray-50 dark:hover:bg-gray-800"
              :disabled="filePage >= totalFilePages"
              @click="filePage++"
            >
              →
            </button>
          </div>
        </div>
        <ul class="overflow-y-auto flex-1 pb-8">
          <li
            v-for="(f, i) in pagedFiles"
            :key="i"
            class="px-3 py-2 text-sm border-b last:border-b-0 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800"
            :class="{ 'bg-blue-50 dark:bg-blue-900/20': selectedIndex === startIndex + i }"
            @click="selectedIndex = startIndex + i"
          >
            <div class="flex items-center justify-between">
              <span class="truncate font-mono text-xs">{{ f.newFile || f.oldFile }}</span>
              <span
                v-if="fileReviewCount(f) > 0"
                class="ml-2 text-[10px] px-1.5 py-0.5 rounded-full bg-red-500 text-white"
              >
                {{ fileReviewCount(f) }}
              </span>
            </div>
            <div class="text-xs text-gray-500 mt-1">
              <span v-if="f.isNew" class="text-green-600">新增</span>
              <span v-else-if="f.isDeleted" class="text-red-600">删除</span>
              <span v-else>修改</span>
            </div>
          </li>
        </ul>
      </div>
    </aside>

    <section class="lg:col-span-3 h-full overflow-y-auto space-y-4 pr-1 pb-8">
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

const filePageSize = 20
const filePage = ref(1)
const selectedIndex = ref(0)

const totalFiles = computed(() => props.diffData?.files?.length || 0)
const totalFilePages = computed(() => Math.max(1, Math.ceil(totalFiles.value / filePageSize)))
const startIndex = computed(() => (filePage.value - 1) * filePageSize)
const pagedFiles = computed(() => props.diffData?.files?.slice(startIndex.value, startIndex.value + filePageSize) || [])
const selectedFile = computed(() => props.diffData?.files[selectedIndex.value] || null)

function fileReviewCount(file) {
  const path = file.newFile || file.oldFile || ''
  return props.comments.filter(c => c.filePath === path).length
}

watch(() => props.diffData, () => {
  filePage.value = 1
  selectedIndex.value = 0
}, { immediate: false })
</script>
