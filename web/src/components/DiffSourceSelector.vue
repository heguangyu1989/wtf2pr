<template>
  <div class="flex min-w-0 flex-wrap items-center gap-2">
    <div class="flex flex-wrap items-center gap-2">
      <div class="flex rounded-md bg-gray-100 p-0.5 dark:bg-gray-800" aria-label="Diff 来源">
        <button
          v-for="mode in modes"
          :key="mode.value"
          type="button"
          class="min-h-6 rounded px-2.5 py-0.5 text-[13px] font-medium"
          :class="target.type === mode.value ? 'bg-white text-gray-900 dark:bg-gray-700 dark:text-gray-100' : 'text-[var(--muted)] hover:text-gray-800 dark:hover:text-gray-200'"
          :aria-pressed="target.type === mode.value"
          @click="selectMode(mode.value)"
        >{{ mode.label }}</button>
      </div>
      <div class="text-xs text-gray-500">{{ scopeDescription }}</div>
    </div>

    <div v-if="target.type === 'commit'" class="mt-1 flex w-full flex-wrap items-center gap-2">
      <label for="commit-select" class="text-xs font-medium text-gray-600 dark:text-gray-300">指定 Commit</label>
      <select
        id="commit-select"
        :value="target.commit"
        class="min-h-8 min-w-0 flex-1 rounded border bg-white px-2 text-xs dark:border-gray-600 dark:bg-gray-900 sm:max-w-xl"
        @change="updateTarget('commit', $event.target.value)"
      >
        <option value="">选择 Commit</option>
        <option v-for="commit in commitOptions" :key="commit.hash" :value="commit.hash">
          {{ commit.hash.substring(0, 7) }} · {{ commit.message }}
        </option>
      </select>
      <div class="flex items-center gap-1 text-xs">
        <button type="button" class="min-h-8 rounded border px-2 disabled:opacity-40 dark:border-gray-600" :disabled="commitPage <= 1 || commitsLoading" @click="changeCommitPage(-1)">上一页</button>
        <span class="px-1 tabular-nums text-gray-500">{{ commitPage }} / {{ commitTotalPages }}</span>
        <button type="button" class="min-h-8 rounded border px-2 disabled:opacity-40 dark:border-gray-600" :disabled="commitPage >= commitTotalPages || commitsLoading" @click="changeCommitPage(1)">下一页</button>
      </div>
      <span v-if="commitsLoading" class="text-xs text-gray-500" role="status">加载中…</span>
      <span v-else-if="commitsError" class="text-xs text-red-700">
        {{ commitsError }}
        <button type="button" class="ml-1 underline" @click="loadCommits">重试</button>
      </span>
    </div>

    <div v-if="target.type === 'branch'" class="mt-1 flex w-full flex-wrap items-center gap-2 text-xs">
      <span class="font-medium text-gray-600 dark:text-gray-300">当前分支</span>
      <code class="rounded bg-gray-100 px-2 py-1 dark:bg-gray-800">{{ branches?.current || 'detached HEAD' }}</code>
      <span aria-hidden="true">→</span>
      <label for="branch-select" class="sr-only">目标分支</label>
      <span v-if="branchLoading" class="text-gray-500" role="status">正在加载分支…</span>
      <span v-else-if="branchError" class="text-red-700">
        {{ branchError }}
        <button type="button" class="ml-1 underline" @click="emit('retry-branches')">重试</button>
      </span>
      <template v-else>
        <select
          v-if="availableBranches.length"
          id="branch-select"
          :value="target.base"
          class="min-h-8 rounded border bg-white px-2 dark:border-gray-600 dark:bg-gray-900"
          @change="updateTarget('branch', '', $event.target.value)"
        >
          <option value="">选择目标分支</option>
          <option v-for="branch in availableBranches" :key="branch.name" :value="branch.name">{{ branch.name }}</option>
        </select>
        <span v-else class="text-gray-500">没有可对比的本地分支</span>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { getCommits } from '../api/client.js'

const props = defineProps({
  target: { type: Object, required: true },
  branches: { type: Object, default: null },
  branchError: { type: String, default: '' },
  branchLoading: { type: Boolean, default: false },
})
const emit = defineEmits(['update:target', 'retry-branches'])

const modes = [
  { value: 'working', label: '未提交' },
  { value: 'commit', label: 'Commit' },
  { value: 'branch', label: '分支对比' },
]
const commitList = ref([])
const commitPage = ref(1)
const commitTotalPages = ref(1)
const commitsLoading = ref(false)
const commitsError = ref('')
const lastCommit = ref(props.target.commit)
const lastBase = ref(props.target.base)

const availableBranches = computed(() => props.branches?.list?.filter(branch => !branch.current && branch.name !== props.branches?.current) || [])
const commitOptions = computed(() => {
  if (!props.target.commit || commitList.value.some(commit => commit.hash === props.target.commit)) return commitList.value
  return [{ hash: props.target.commit, message: '当前 Review Commit' }, ...commitList.value]
})
const scopeDescription = computed(() => {
  if (props.target.type === 'working') return '已暂存、未暂存与未跟踪变更'
  if (props.target.type === 'commit') return '查看一个提交的完整差异'
  return '目标分支...HEAD · 不含未提交修改'
})

watch(() => props.target, target => {
  if (target.commit) lastCommit.value = target.commit
  if (target.base) lastBase.value = target.base
}, { deep: true, immediate: true })

function updateTarget(type, commit = '', base = '') {
  if (props.target.type === type && props.target.commit === commit && props.target.base === base) return
  emit('update:target', { type, commit, base })
}

function selectMode(type) {
  if (type === 'commit') updateTarget(type, lastCommit.value)
  else if (type === 'branch') updateTarget(type, '', lastBase.value || props.branches?.default || availableBranches.value[0]?.name || '')
  else updateTarget(type)
}

async function loadCommits() {
  commitsLoading.value = true
  commitsError.value = ''
  try {
    const res = await getCommits(commitPage.value, 10)
    commitList.value = res.list || []
    commitPage.value = res.page || 1
    commitTotalPages.value = res.totalPages || 1
  } catch (error) {
    commitsError.value = error.message
    commitList.value = []
  } finally {
    commitsLoading.value = false
  }
}

function changeCommitPage(delta) {
  commitPage.value += delta
  loadCommits()
}

onMounted(loadCommits)
</script>
