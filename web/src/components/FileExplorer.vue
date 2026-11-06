<template>
  <aside class="flex h-full min-h-0 flex-col overflow-hidden rounded-lg border bg-white dark:border-gray-700 dark:bg-gray-900" aria-label="变更文件">
    <div class="shrink-0 border-b px-3 py-2.5 dark:border-gray-700">
      <div class="flex items-center justify-between gap-2">
        <h2 class="text-sm font-semibold">变更文件</h2>
        <span class="text-xs tabular-nums text-gray-500">{{ fileCountLabel }}</span>
      </div>
      <div class="mt-2 flex items-center gap-1" aria-label="文件视图">
        <button
          v-for="mode in viewModes"
          :key="mode.value"
          type="button"
          class="flex min-h-7 items-center gap-1 rounded px-2 py-1 text-xs"
          :class="viewMode === mode.value ? 'bg-gray-200 font-medium dark:bg-gray-700' : 'text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800'"
          :aria-pressed="viewMode === mode.value"
          @click="viewMode = mode.value"
        >
          <AppIcon :name="mode.value === 'tree' ? 'tree' : 'list'" class="h-3.5 w-3.5" />
          {{ mode.label }}
        </button>
      </div>
      <div class="mt-2 flex items-center gap-1.5">
        <label class="sr-only" for="file-sort">文件排序规则</label>
        <select id="file-sort" v-model="sortKey" class="min-w-0 flex-1 rounded border bg-white px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-900" aria-label="文件排序规则">
          <option value="path">默认路径</option>
          <option value="type">文件类型</option>
          <option value="test">测试文件优先</option>
          <option value="document">文档文件优先</option>
        </select>
        <button
          type="button"
          class="flex min-h-7 min-w-7 items-center justify-center rounded border text-gray-600 hover:bg-gray-100 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
          :aria-pressed="direction === 'desc'"
          aria-label="反向排序"
          :title="direction === 'asc' ? '切换为降序' : '切换为升序'"
          @click="direction = direction === 'asc' ? 'desc' : 'asc'"
        >
          <AppIcon :name="direction === 'asc' ? 'sort-asc' : 'sort-desc'" class="h-4 w-4" />
        </button>
      </div>
      <button
        type="button"
        class="mt-2 flex min-h-7 w-full items-center rounded border px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
        :aria-expanded="showFilters"
        aria-controls="file-filter-panel"
        @click="showFilters = !showFilters"
      >
        <AppIcon :name="showFilters ? 'chevron-down' : 'chevron-right'" class="mr-1 h-3.5 w-3.5" />
        <span>包含 / 排除</span>
        <span v-if="hasPathFilters" class="ml-auto text-[var(--accent)]">已启用</span>
      </button>
      <div v-if="showFilters" id="file-filter-panel" class="mt-2 space-y-2">
        <div class="flex items-center gap-1.5">
          <label for="file-include" class="w-7 shrink-0 text-xs text-gray-500">包含</label>
          <input
            id="file-include"
            v-model="includePatterns"
            type="text"
            class="min-w-0 flex-1 rounded border bg-white px-2 py-1 font-mono text-xs dark:border-gray-600 dark:bg-gray-900"
            placeholder="**/*.go, src/**"
            autocomplete="off"
            spellcheck="false"
            aria-describedby="file-filter-help"
          >
          <select class="w-[4.5rem] rounded border bg-white px-1 py-1 text-xs dark:border-gray-600 dark:bg-gray-900" aria-label="向包含规则添加内置规则" @change="appendPreset('include', $event)">
            <option value="">规则</option>
            <option v-for="preset in filterPresets" :key="preset.label" :value="preset.value">{{ preset.label }}</option>
          </select>
        </div>
        <div class="flex items-center gap-1.5">
          <label for="file-exclude" class="w-7 shrink-0 text-xs text-gray-500">排除</label>
          <input
            id="file-exclude"
            v-model="excludePatterns"
            type="text"
            class="min-w-0 flex-1 rounded border bg-white px-2 py-1 font-mono text-xs dark:border-gray-600 dark:bg-gray-900"
            placeholder="**/*_test.go, docs/**"
            autocomplete="off"
            spellcheck="false"
            aria-describedby="file-filter-help"
          >
          <select class="w-[4.5rem] rounded border bg-white px-1 py-1 text-xs dark:border-gray-600 dark:bg-gray-900" aria-label="向排除规则添加内置规则" @change="appendPreset('exclude', $event)">
            <option value="">规则</option>
            <option v-for="preset in filterPresets" :key="preset.label" :value="preset.value">{{ preset.label }}</option>
          </select>
        </div>
        <div class="flex items-center justify-between gap-2">
          <p id="file-filter-help" class="text-[11px] text-gray-500">逗号分隔；支持 *、**、?、[]、{}</p>
          <button v-if="hasPathFilters" type="button" class="shrink-0 text-[11px] text-[var(--accent)] hover:underline" @click="clearFilters">清空</button>
        </div>
      </div>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto p-1.5">
      <div v-if="!filteredFiles.length" class="px-3 py-6 text-center text-xs text-gray-500">没有匹配的文件</div>
      <ul v-else-if="viewMode === 'tree'" role="list">
        <FileTreeNode
          v-for="node in tree"
          :key="getNodeKey(node)"
          :node="node"
          :selected-path="selectedPath"
          :collapsed="collapsed"
          :comment-counts="commentCounts"
          @select="$emit('select', $event)"
          @toggle="toggleDirectory"
        />
      </ul>
      <ul v-else role="list">
        <li v-for="file in visibleFiles" :key="getFilePath(file)">
          <button
            type="button"
            class="flex min-h-8 w-full items-center gap-2 rounded px-2 py-1 text-left text-xs hover:bg-gray-100 dark:hover:bg-gray-800"
            :class="selectedPath === getFilePath(file) ? 'bg-[var(--selection-bg)] text-[var(--text)]' : 'text-gray-700 dark:text-gray-300'"
            :aria-current="selectedPath === getFilePath(file) ? 'page' : undefined"
            :title="getFilePath(file)"
            @click="$emit('select', getFilePath(file))"
          >
            <span class="w-4 shrink-0 text-center font-medium" :class="file.isNew ? 'text-green-700' : file.isDeleted ? 'text-red-700' : 'text-gray-500'">{{ file.isNew ? 'A' : file.isDeleted ? 'D' : 'M' }}</span>
            <span class="min-w-0 flex-1 truncate font-mono">{{ getFilePath(file) }}</span>
            <span v-if="commentCounts.get(getFilePath(file))" class="rounded-full bg-red-100 px-1.5 text-[11px] text-red-700 dark:bg-red-900/40 dark:text-red-300">{{ commentCounts.get(getFilePath(file)) }}</span>
          </button>
        </li>
      </ul>
    </div>
  </aside>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import AppIcon from './AppIcon.vue'
import FileTreeNode from './FileTreeNode.vue'
import { buildFileTree, filterFiles, getFilePath, getNodeKey, parseGlobPatterns, sortFiles } from '../utils/fileExplorer.js'

const props = defineProps({
  files: { type: Array, default: () => [] },
  comments: { type: Array, default: () => [] },
  selectedPath: { type: String, default: '' },
})
defineEmits(['select'])

const viewModes = [
  { value: 'tree', label: '树形' },
  { value: 'flat', label: '平铺' },
]
const filterPresets = [
  { label: 'Go 测试', value: '**/*_test.go' },
  { label: 'Markdown', value: '**/*.md' },
  { label: 'JS/TS 测试', value: '**/*.{test,spec}.{js,jsx,ts,tsx}' },
  { label: '文档目录', value: 'docs/**' },
]

function readPreference(key, fallback, allowed) {
  try {
    if (typeof localStorage === 'undefined') return fallback
    const value = localStorage.getItem(key)
    if (value === null) return fallback
    return !allowed || allowed.includes(value) ? value : fallback
  } catch {
    return fallback
  }
}

function writePreference(key, value) {
  try {
    if (typeof localStorage !== 'undefined') localStorage.setItem(key, value)
  } catch {
    return
  }
}

const viewMode = ref(readPreference('wtf2pr.fileView', 'tree', ['tree', 'flat']))
const sortKey = ref(readPreference('wtf2pr.fileSort', 'path', ['path', 'type', 'test', 'document']))
const direction = ref(readPreference('wtf2pr.fileDirection', 'asc', ['asc', 'desc']))
const includePatterns = ref(readPreference('wtf2pr.fileInclude', ''))
const excludePatterns = ref(readPreference('wtf2pr.fileExclude', ''))
const showFilters = ref(Boolean(includePatterns.value || excludePatterns.value))
const collapsed = ref(new Set())

const filteredFiles = computed(() => filterFiles(props.files, includePatterns.value, excludePatterns.value))
const visibleFiles = computed(() => sortFiles(filteredFiles.value, sortKey.value, direction.value))
const tree = computed(() => buildFileTree(filteredFiles.value, sortKey.value, direction.value))
const hasPathFilters = computed(() => Boolean(includePatterns.value.trim() || excludePatterns.value.trim()))
const fileCountLabel = computed(() => {
  if (!hasPathFilters.value) return `${props.files.length} 个文件`
  return `${filteredFiles.value.length} / ${props.files.length} 个文件`
})
const commentCounts = computed(() => {
  const counts = new Map()
  for (const comment of props.comments) {
    counts.set(comment.filePath, (counts.get(comment.filePath) || 0) + 1)
  }
  return counts
})

watch([viewMode, sortKey, direction, includePatterns, excludePatterns], ([view, sort, order, include, exclude]) => {
  writePreference('wtf2pr.fileView', view)
  writePreference('wtf2pr.fileSort', sort)
  writePreference('wtf2pr.fileDirection', order)
  writePreference('wtf2pr.fileInclude', include)
  writePreference('wtf2pr.fileExclude', exclude)
})

function appendPreset(field, event) {
  const target = field === 'include' ? includePatterns : excludePatterns
  const merged = parseGlobPatterns(target.value)
  for (const pattern of parseGlobPatterns(event.target.value)) {
    if (!merged.includes(pattern)) merged.push(pattern)
  }
  target.value = merged.join(', ')
  event.target.value = ''
}

function clearFilters() {
  includePatterns.value = ''
  excludePatterns.value = ''
}

function toggleDirectory(path) {
  const next = new Set(collapsed.value)
  if (next.has(path)) next.delete(path)
  else next.add(path)
  collapsed.value = next
}
</script>
