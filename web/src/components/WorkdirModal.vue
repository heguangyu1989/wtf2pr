<template>
  <div class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-[80] px-4" @click.self="onOverlayClose">
    <div class="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-full max-w-2xl max-h-[85vh] flex flex-col border border-gray-200/60 dark:border-gray-700/60">
      <div class="px-5 py-3.5 border-b dark:border-gray-700 flex items-center justify-between shrink-0">
        <div>
          <div class="font-semibold text-sm">切换工作目录</div>
          <div class="text-xs text-gray-500 mt-0.5">不同工作目录的 Review 数据相互独立存储</div>
        </div>
        <button class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 text-xl leading-none px-1 disabled:opacity-40" :disabled="switching" @click="onOverlayClose">×</button>
      </div>

      <div class="flex-1 overflow-hidden flex flex-col min-h-0">
        <!-- 允许访问的目录（启用路径限制时展示） -->
        <div v-if="restricted" class="px-5 pt-4 shrink-0">
          <div class="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">允许访问的目录</div>
          <div class="flex flex-wrap gap-2 max-h-24 overflow-y-auto">
            <button
              v-for="dir in allowedDirs"
              :key="dir"
              class="px-2.5 py-1.5 text-xs rounded-lg border transition font-mono truncate max-w-full"
              :class="dir === currentWorkdir
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-300 cursor-default'
                : 'border-gray-200 dark:border-gray-700 hover:border-blue-400 hover:bg-blue-50/50 dark:hover:bg-blue-900/10'"
              :disabled="dir === currentWorkdir"
              :title="dir"
              @click="onSelect(dir)"
            >
              {{ dir }}<span v-if="dir === currentWorkdir" class="ml-1 font-sans">(当前)</span>
            </button>
          </div>
          <p class="text-[11px] text-gray-400 mt-1.5">管理员已通过配置文件限制可访问的路径范围，仅允许切换到上述目录之内</p>
        </div>

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
              <span v-if="currentPath">{{ currentPath }}</span>
              <span v-else class="text-gray-400 font-sans">请从上方"允许访问的目录"中选择一个路径</span>
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
              @click="browse(d.path)"
            >
              <svg class="w-4 h-4 text-amber-500 shrink-0" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
              </svg>
              <span class="truncate">{{ d.name }}</span>
              <span v-if="d.isGit" class="text-[10px] px-1.5 py-0.5 rounded bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300 shrink-0">git</span>
            </li>
          </ul>
          <div v-if="truncated" class="text-[11px] text-amber-600 dark:text-amber-400 mt-1 shrink-0">目录过多，仅显示前 500 条</div>
        </div>
      </div>

      <div class="px-5 py-3.5 border-t dark:border-gray-700 shrink-0">
        <p v-if="switchError" class="text-xs text-red-500 mb-2">{{ switchError }}</p>
        <div class="flex items-center justify-between gap-2">
          <div class="text-xs text-gray-400 truncate">切换后页面数据将按新目录重新加载</div>
          <div class="flex gap-2 shrink-0">
            <button class="px-3 py-1.5 text-sm border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition disabled:opacity-40" :disabled="switching" @click="onOverlayClose">取消</button>
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

const props = defineProps({
  hasUnsaved: { type: Boolean, default: false },
})
const emit = defineEmits(['close', 'switched', 'unauthorized'])

const workspaces = ref([])
const allowedDirs = ref([])
const restricted = ref(false)
const currentWorkdir = ref('')
const currentPath = ref('')
const parent = ref('')
const isGit = ref(false)
const dirs = ref([])
const truncated = ref(false)
const loading = ref(false)
const browseError = ref('')
const switching = ref(false)
const switchError = ref('')

function shortPath(p) {
  const parts = p.split('/')
  if (parts.length <= 2) return p
  return '…/' + parts.slice(-2).join('/')
}

// 切换进行中禁止关闭，避免响应到达后用户已离开弹窗却触发整页重置
function onOverlayClose() {
  if (switching.value) return
  emit('close')
}

// 浏览并发序号：仅最新一次浏览的结果生效，避免快速点击时后到响应覆盖
let browseSeq = 0
let unauth = false

async function browse(path) {
  const seq = ++browseSeq
  loading.value = true
  browseError.value = ''
  try {
    const res = await browseDirs(path || '')
    if (seq !== browseSeq) return false
    currentPath.value = res.path
    parent.value = res.parent || ''
    isGit.value = !!res.isGit
    dirs.value = res.dirs || []
    truncated.value = !!res.truncated
    return true
  } catch (e) {
    if (e && e.status === 401) {
      unauth = true
      emit('unauthorized')
      return false
    }
    if (seq !== browseSeq) return false
    browseError.value = e.message
    dirs.value = []
    return false
  } finally {
    if (seq === browseSeq) loading.value = false
  }
}

async function onSelect(path) {
  if (switching.value) return
  // 确认必须在发起切换之前，否则取消会造成前后端状态分裂
  if (props.hasUnsaved && !window.confirm('当前有未保存的评论，切换工作目录后将被丢弃，确认切换？')) {
    return
  }
  switching.value = true
  switchError.value = ''
  try {
    const info = await switchWorkdir(path)
    emit('switched', info)
    emit('close')
  } catch (e) {
    if (e && e.status === 401) {
      emit('unauthorized')
      return
    }
    switchError.value = e.message
  } finally {
    switching.value = false
  }
}

onMounted(async () => {
  try {
    const info = await getWorkdir()
    workspaces.value = (info.workspaces || []).filter(ws => !info.restricted || (info.allowedDirs || []).some(dir => ws.path === dir || dir === '/' || ws.path.startsWith(dir + '/')))
    allowedDirs.value = info.allowedDirs || []
    restricted.value = !!info.restricted
    currentWorkdir.value = info.workDir || ''
    const ok = await browse(info.workDir || '')
    if (!ok && !unauth) {
      // 当前目录不在允许范围内时，回退到允许目录列表
      await browse('')
    }
  } catch (e) {
    if ((e && e.status === 401) || unauth) {
      emit('unauthorized')
      return
    }
    await browse('')
  }
})
</script>
