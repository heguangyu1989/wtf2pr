<template>
  <div v-if="!diffData || !diffData.files.length" class="flex flex-1 items-center justify-center rounded-lg border bg-white p-6 text-sm text-gray-500 dark:border-gray-700 dark:bg-gray-900">当前对比没有 diff 数据</div>
  <div v-else class="grid h-full min-h-0 flex-1 grid-cols-1 grid-rows-[minmax(0,2fr)_minmax(0,3fr)] gap-3 lg:grid-cols-[280px_minmax(0,1fr)] lg:grid-rows-[minmax(0,1fr)]">
    <div class="h-full min-h-0">
      <FileExplorer
        :files="diffData.files"
        :comments="comments"
        :selected-path="selectedPath"
        @select="$emit('update:selectedPath', $event)"
      />
    </div>

    <section class="min-h-0 space-y-3 overflow-y-auto pb-6" aria-label="代码差异">
      <div v-if="commitInfo" class="rounded-lg border bg-white p-3 text-sm dark:border-gray-700 dark:bg-gray-900">
        <div class="font-medium">{{ commitInfo.message }}</div>
        <div class="mt-1 font-mono text-xs text-[var(--accent)]">{{ commitInfo.hash }}</div>
        <div class="mt-1 text-xs text-gray-500">{{ commitInfo.author }} · {{ commitInfo.date }}</div>
      </div>

      <ReadonlyDiffFile
        v-if="readonly && selectedFile"
        :file="selectedFile"
        :comments="comments"
        @view-full="openFullFile"
      />
      <DiffFile
        v-else-if="selectedFile"
        :key="getFilePath(selectedFile)"
        :file="selectedFile"
        :comments="comments"
        @view-full="openFullFile"
        @update:comments="$emit('update:comments', $event)"
      />
      <div v-else class="rounded-lg border bg-white p-6 text-sm text-gray-500 dark:border-gray-700 dark:bg-gray-900">选择一个文件查看差异</div>
    </section>

    <FileContentModal
      v-if="fullFile"
      :target="fullTarget"
      :file="fullFile"
      @close="fullFile = null"
    />
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import DiffFile from './DiffFile.vue'
import ReadonlyDiffFile from './ReadonlyDiffFile.vue'
import FileExplorer from './FileExplorer.vue'
import FileContentModal from './FileContentModal.vue'
import { getFilePath } from '../utils/fileExplorer.js'

const props = defineProps({
  diffData: { type: Object, default: null },
  comments: { type: Array, default: () => [] },
  readonly: { type: Boolean, default: false },
  commitInfo: { type: Object, default: null },
  selectedPath: { type: String, default: '' },
  target: { type: Object, default: () => ({ type: 'working', commit: '', base: '' }) },
})
defineEmits(['update:comments', 'update:selectedPath'])

const fullFile = ref(null)
const fullTarget = ref(null)
const selectedFile = computed(() => props.diffData?.files?.find(file => getFilePath(file) === props.selectedPath) || null)

function openFullFile() {
  fullTarget.value = { ...props.target }
  fullFile.value = selectedFile.value
}

watch(() => props.diffData, () => {
  fullFile.value = null
})
watch(() => props.target, () => {
  fullFile.value = null
}, { deep: true })
</script>
