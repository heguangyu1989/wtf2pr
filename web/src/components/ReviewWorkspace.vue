<template>
  <div class="flex h-full min-h-0 flex-col gap-2">
    <header class="shrink-0 rounded-lg border bg-white px-2 py-1.5 dark:border-gray-700 dark:bg-gray-900">
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
        <h1 class="shrink-0 px-1 text-sm font-semibold tracking-[-0.01em]">代码审查</h1>
        <WorkdirSwitcher
          :work-dir="workDir"
          :changing="workdirChanging"
          :error="workdirError"
          @switch="$emit('switch-workdir', $event)"
        />
        <span class="hidden h-4 w-px bg-[var(--border)] sm:block" aria-hidden="true"></span>
        <DiffSourceSelector
          :key="workDir"
          :target="target"
          :branches="branches"
          :branch-error="branchError"
          :branch-loading="branchLoading"
          @retry-branches="loadBranches"
          @update:target="$emit('update:target', $event)"
        />
        <div class="ml-auto flex flex-wrap items-center gap-x-2.5 text-[13px] text-[var(--muted)]">
          <span class="font-mono text-[var(--accent)]">{{ scopeSummary }}</span>
          <span v-if="diffData" class="tabular-nums">{{ diffData.files.length }} 个变更文件</span>
          <span v-if="diffData?.comparison" :title="`共同祖先 ${diffData.comparison.mergeBase}`">共同祖先 {{ diffData.comparison.mergeBase.slice(0, 8) }}</span>
        </div>
      </div>
    </header>

    <div v-if="target.type === 'commit' && target.commit && !reviewHydrated" class="shrink-0 rounded-lg border bg-white px-3 py-2 text-xs dark:border-gray-700 dark:bg-gray-900">
      <div v-if="reviewLoadError" class="flex items-center gap-2 text-[var(--negative)]" role="alert">
        <span>评论加载失败：{{ reviewLoadError }}</span>
        <button type="button" class="underline" @click="$emit('retry-review-load')">重试</button>
      </div>
      <div v-else class="text-[var(--muted)]" role="status">正在加载评论，完成后才能编辑、保存或导出…</div>
    </div>

    <div v-if="loading" class="flex flex-1 items-center justify-center rounded-lg border bg-white text-sm text-gray-500 dark:border-gray-700 dark:bg-gray-900" role="status">
      正在加载变更…
    </div>
    <div v-else-if="error" class="flex flex-1 flex-col items-center justify-center gap-3 rounded-lg border bg-white p-6 text-center dark:border-gray-700 dark:bg-gray-900" role="alert">
      <div class="text-sm text-red-700">{{ error }}</div>
      <button type="button" class="min-h-8 rounded border px-3 text-sm dark:border-gray-600" @click="loadDiff">重试</button>
    </div>
    <div v-else-if="needsCommit" class="flex flex-1 items-center justify-center rounded-lg border bg-white text-sm text-gray-500 dark:border-gray-700 dark:bg-gray-900">
      选择一个 Commit 开始审查
    </div>
    <div v-else-if="needsBase" class="flex flex-1 items-center justify-center rounded-lg border bg-white text-sm text-gray-500 dark:border-gray-700 dark:bg-gray-900">
      选择一个目标本地分支开始对比
    </div>
    <DiffViewer
      v-else
      :diff-data="diffData"
      :comments="visibleComments"
      :readonly="target.type !== 'commit' || !reviewHydrated"
      :commit-info="diffData?.commitInfo"
      :target="target"
      :selected-path="selectedPath"
      @update:selected-path="selectedPath = $event"
      @update:comments="$emit('update:comments', $event)"
    />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import DiffSourceSelector from './DiffSourceSelector.vue'
import DiffViewer from './DiffViewer.vue'
import WorkdirSwitcher from './WorkdirSwitcher.vue'
import { getBranches, getDiff } from '../api/client.js'
import { getDefaultFilePath, getFilePath } from '../utils/fileExplorer.js'

const props = defineProps({
  target: { type: Object, required: true },
  comments: { type: Array, default: () => [] },
  reviewHydrated: { type: Boolean, default: true },
  reviewLoadError: { type: String, default: '' },
  workDir: { type: String, default: '' },
  workdirChanging: { type: Boolean, default: false },
  workdirError: { type: String, default: '' },
})
defineEmits(['update:target', 'update:comments', 'retry-review-load', 'switch-workdir'])

const branches = ref(null)
const branchError = ref('')
const branchLoading = ref(true)
const diffData = ref(null)
const loading = ref(false)
const error = ref('')
const selectedPath = ref('')
let requestSerial = 0
let controller = null

const needsCommit = computed(() => props.target.type === 'commit' && !props.target.commit)
const needsBase = computed(() => props.target.type === 'branch' && !props.target.base)
const visibleComments = computed(() => props.target.type === 'commit' ? props.comments : [])
const scopeSummary = computed(() => {
  if (props.target.type === 'working') return 'HEAD → 工作区'
  if (props.target.type === 'commit') return props.target.commit ? `${props.target.commit.slice(0, 8)}^ → ${props.target.commit.slice(0, 8)}` : '选择 Commit'
  const comparison = diffData.value?.comparison
  if (comparison) return `${comparison.base}...${comparison.head}`
  return props.target.base ? `${props.target.base}...HEAD` : '选择目标分支'
})

let branchRequestSerial = 0

async function loadBranches() {
  const serial = ++branchRequestSerial
  branchLoading.value = true
  branchError.value = ''
  branches.value = null
  try {
    const data = await getBranches()
    if (serial === branchRequestSerial) branches.value = data
  } catch (e) {
    if (serial === branchRequestSerial) {
      branchError.value = e.message
      branches.value = null
    }
  } finally {
    if (serial === branchRequestSerial) branchLoading.value = false
  }
}

async function loadDiff() {
  controller?.abort()
  const serial = ++requestSerial
  controller = new AbortController()
  error.value = ''
  diffData.value = null
  if (needsCommit.value || needsBase.value) {
    loading.value = false
    selectedPath.value = ''
    return
  }
  loading.value = true
  try {
    const target = { ...props.target }
    const data = await getDiff(target.type, target.commit, target.base, { signal: controller.signal })
    if (serial !== requestSerial) return
    diffData.value = data
    if (!data.files.some(file => getFilePath(file) === selectedPath.value)) {
      selectedPath.value = getDefaultFilePath(data.files)
    }
  } catch (e) {
    if (serial === requestSerial && e.name !== 'AbortError') {
      diffData.value = null
      selectedPath.value = ''
      error.value = e.message
    }
  } finally {
    if (serial === requestSerial) loading.value = false
  }
}

watch(() => props.target, loadDiff, { deep: true })
watch(() => props.workDir, () => {
  selectedPath.value = ''
  loadBranches()
  loadDiff()
})
onMounted(() => {
  loadBranches()
  loadDiff()
})
onBeforeUnmount(() => controller?.abort())
</script>
