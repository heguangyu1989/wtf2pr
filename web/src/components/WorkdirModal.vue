<template>
  <div class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-[80] px-4" @click.self="$emit('close')">
    <div class="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-full max-w-2xl max-h-[85vh] flex flex-col border border-gray-200/60 dark:border-gray-700/60">
      <div class="px-5 py-3.5 border-b dark:border-gray-700 flex items-center justify-between shrink-0">
        <div>
          <div class="font-semibold text-sm">切换工作目录</div>
          <div class="text-xs text-gray-500 mt-0.5">不同工作目录的 Review 数据相互独立存储</div>
        </div>
        <button class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 text-xl leading-none px-1" @click="$emit('close')">×</button>
      </div>

      <div class="flex-1 overflow-hidden flex flex-col min-h-0">
        <!-- 最近使用的工作目录 -->
        <div v-if="workspaces.length" class="px-5 pt-4 shrink-0">
          <div class="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">最近使用</div>
          <div class="flex flex-wrap gap-2 max-h-24 overflow-y-auto">
            <button
              v-for="ws in workspaces"
              :key="ws.id"
              class="px-2.5 py-1.5 text-xs rounded-lg border transition font-mono truncate max-w-full"
              :class="ws.current
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-300 cursor-default'
                : 'border-gray-200 dark:border-gray-700 hover:border-blue-400 hover:bg-blue-50/50 dark:hover:bg-blue-900/10'"
              :disabled="ws.current"
              :title="ws.path"
              @click="onSelect(ws.path)"
            >
              {{ shortPath(ws.path) }}<span v-if="ws.current" class="ml-1 font-sans">(当前)</span>
            </button>
          </div>
        </div>

        <!-- 目录浏览器 -->
        <div class="px-5 py-4 flex-1 overflow-hidden flex flex-col min-h-0">
          <div class="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2 shrink-0">浏览目录</div>
          <div class="flex items-center gap-2 mb-2 shrink-0">
            <button
              class="px-2 py-1 text-xs border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-40 shrink-0"
              :disabled="!parent"
              @click="browse(parent)"
            >
              ← 上级
            </button>
            <div class="flex-1 px-2.5 py-1.5 text-xs font-mono bg-gray-50 dark:bg-gray-800 rounded-md truncate" :title="currentPath">
              {{ currentPath }}
            </div>
            <span v-if="isGit" class="text-[10px] px-1.5 py-0.5 rounded bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300 shrink-0">git</span>
          </div>

          <div v-if="loading" class="text-sm text-gray-500 py-8 text-center">加载中…</div>
          <div v-else-if="browseError" class="text-sm text-red-500 py-8 text-center">{{ browseError }}</div>
          <ul v-else class="flex-1 overflow-y-auto border dark:border-gray-700 rounded-lg divide-y dark:divide-gray-800 min-h-[12rem]">
            <li v-if="!dirs.length" class="px-3 py-6 text-sm text-gray-400 text-center">没有子目录</li>
            <li
              v-for="d in dirs"
              :key="d.path"
              class="px-3 py-2 text-sm cursor-pointer hover:bg-blue-50 dark:hover:bg-blue-900/10 flex items-center gap-2 transition"
              @dblclick="browse(d.path)"
              @click="browse(d.path)"
            >
              <svg class="w-4 h-4 text-amber-500 shrink-0" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
              </svg>
              <span class="truncate">{{ d.name }}</span>
              <span v-if="d.isGit" class="text-[10px] px-1.5 py-0.5 rounded bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300 shrink-0">git</span>
            </li>
          </ul>
        </div>
      </div>

      <div class="px-5 py-3.5 border-t dark:border-gray-700 shrink-0">
        <p v-if="switchError" class="text-xs text-red-500 mb-2">{{ switchError }}</p>
        <div class="flex items-center justify-between gap-2">
          <div class="text-xs text-gray-400 truncate">切换后页面数据将按新目录重新加载</div>
          <div class="flex gap-2 shrink-0">
            <button class="px-3 py-1.5 text-sm border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition" @click="$emit('close')">取消</button>
            <button
              class="px-4 py-1.5 text-sm rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 transition shadow-sm"
              :disabled="switching || !isGit || currentPath === currentWorkdir"
              @click="onSelect(currentPath)"
            >
              {{ switching ? '切换中…' : '选择此目录' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getWorkdir, switchWorkdir, browseDirs } from '../api/client.js'

const emit = defineEmits(['close', 'switched'])

const workspaces = ref([])
const currentWorkdir = ref('')
const currentPath = ref('')
const parent = ref('')
const isGit = ref(false)
const dirs = ref([])
const loading = ref(false)
const browseError = ref('')
const switching = ref(false)
const switchError = ref('')

function shortPath(p) {
  const home = '~'
  const parts = p.split('/')
  if (parts.length <= 2) return p
  return home + '…/' + parts.slice(-2).join('/')
}

async function browse(path) {
  loading.value = true
  browseError.value = ''
  try {
    const res = await browseDirs(path || '')
    currentPath.value = res.path
    parent.value = res.parent || ''
    isGit.value = !!res.isGit
    dirs.value = res.dirs || []
  } catch (e) {
    browseError.value = e.message
    dirs.value = []
  } finally {
    loading.value = false
  }
}

async function onSelect(path) {
  if (switching.value) return
  switching.value = true
  switchError.value = ''
  try {
    const info = await switchWorkdir(path)
    emit('switched', info)
    emit('close')
  } catch (e) {
    switchError.value = e.message
  } finally {
    switching.value = false
  }
}

onMounted(async () => {
  try {
    const info = await getWorkdir()
    workspaces.value = info.workspaces || []
    currentWorkdir.value = info.workDir || ''
    await browse(info.workDir || '')
  } catch {
    await browse('')
  }
})
</script>
