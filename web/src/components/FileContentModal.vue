<template>
  <dialog
    ref="dialogRef"
    closedby="any"
    aria-labelledby="file-content-title"
    class="h-[85vh] w-[calc(100vw-2rem)] max-w-6xl overflow-hidden rounded-xl border bg-white p-0 text-gray-900 shadow-xl dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100"
    @close="closeDialog"
    @cancel="closeDialog"
    @pointerdown="onPointerDown"
    @pointerup="onPointerUp"
  >
    <header class="flex shrink-0 items-center gap-3 border-b px-4 py-3 dark:border-gray-700">
      <div class="min-w-0 flex-1">
        <h2 id="file-content-title" class="text-sm font-semibold">完整文件</h2>
        <div class="mt-0.5 truncate font-mono text-xs text-gray-500" :title="filePath">{{ filePath }}</div>
      </div>
      <form method="dialog">
        <button type="submit" value="close" autofocus class="min-h-8 rounded border px-3 text-xs dark:border-gray-600" @click="closeDialog">关闭</button>
      </form>
    </header>

    <div class="flex shrink-0 flex-wrap items-center gap-2 border-b px-4 py-2 dark:border-gray-700">
      <div class="flex rounded-md bg-gray-100 p-1 dark:bg-gray-800" aria-label="文件版本">
        <button type="button" class="min-h-7 rounded px-3 text-xs" :class="side === 'old' ? 'bg-white dark:bg-gray-700' : 'text-gray-500'" :aria-pressed="side === 'old'" @click="side = 'old'">变更前</button>
        <button type="button" class="min-h-7 rounded px-3 text-xs" :class="side === 'new' ? 'bg-white dark:bg-gray-700' : 'text-gray-500'" :aria-pressed="side === 'new'" @click="side = 'new'">变更后</button>
      </div>
      <span class="text-xs text-gray-500">{{ side === 'old' ? 'Old' : 'New' }}</span>
      <div class="flex-1"></div>
      <span v-if="copyStatus" class="text-xs" :class="copyStatus === '已复制' ? 'text-green-700' : 'text-red-700'" role="status">{{ copyStatus }}</span>
      <button type="button" class="min-h-8 rounded border px-3 text-xs disabled:cursor-not-allowed disabled:opacity-40 dark:border-gray-600" :disabled="!hasText" @click="copyContent">复制文件</button>
    </div>

    <div class="min-h-0 flex-1 overflow-auto">
      <div v-if="loading" class="flex h-full items-center justify-center text-sm text-gray-500" role="status">正在加载完整文件…</div>
      <div v-else-if="error" class="flex h-full flex-col items-center justify-center gap-3 p-6" role="alert">
        <p class="text-sm text-red-700">{{ error }}</p>
        <button type="button" class="min-h-8 rounded border px-3 text-xs dark:border-gray-600" @click="loadContent">重试</button>
      </div>
      <div v-else-if="!content?.exists" class="flex h-full items-center justify-center text-sm text-gray-500">该版本不存在</div>
      <div v-else-if="content.isBinary" class="flex h-full items-center justify-center text-sm text-gray-500">二进制文件无法预览</div>
      <div v-else-if="content.content === ''" class="flex h-full items-center justify-center text-sm text-gray-500">空文件</div>
      <ol v-else class="min-w-max py-2 font-mono text-xs leading-6">
        <li v-for="(line, index) in lines" :key="index" class="grid grid-cols-[52px_minmax(0,1fr)] px-3 hover:bg-gray-50 dark:hover:bg-gray-800">
          <span class="select-none pr-3 text-right tabular-nums text-gray-400">{{ index + 1 }}</span>
          <code class="whitespace-pre">{{ line }}</code>
        </li>
      </ol>
    </div>
  </dialog>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { getFileContent } from '../api/client.js'
import { getFilePath, getFileSidePath } from '../utils/fileExplorer.js'

const props = defineProps({
  target: { type: Object, required: true },
  file: { type: Object, required: true },
})
const emit = defineEmits(['close'])

function closeDialog() {
  emit('close')
}

const dialogRef = ref(null)
const side = ref('new')
const content = ref(null)
const loading = ref(true)
const error = ref('')
const copyStatus = ref('')
let controller = null
let requestSerial = 0
let copyTimer = null
let pressedOutside = false

const filePath = computed(() => getFilePath(props.file))
const requestPath = computed(() => getFileSidePath(props.file, side.value))
const lines = computed(() => content.value?.content?.split('\n') || [])
const hasText = computed(() => Boolean(content.value?.exists && !content.value.isBinary && content.value.content))

async function loadContent() {
  controller?.abort()
  const serial = ++requestSerial
  controller = new AbortController()
  loading.value = true
  error.value = ''
  copyStatus.value = ''
  content.value = null
  try {
    const data = await getFileContent({ ...props.target, path: requestPath.value, side: side.value }, { signal: controller.signal })
    if (serial === requestSerial) content.value = data
  } catch (e) {
    if (serial === requestSerial && e.name !== 'AbortError') error.value = e.message
  } finally {
    if (serial === requestSerial) loading.value = false
  }
}

async function copyContent() {
  try {
    if (!navigator.clipboard) throw new Error('Clipboard unavailable')
    await navigator.clipboard.writeText(content.value.content)
    copyStatus.value = '已复制'
  } catch {
    copyStatus.value = '复制失败，请手动选择'
  }
  clearTimeout(copyTimer)
  copyTimer = setTimeout(() => { copyStatus.value = '' }, 2500)
}

function isOutside(event) {
  const rect = dialogRef.value.getBoundingClientRect()
  return event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom
}

function onPointerDown(event) {
  pressedOutside = isOutside(event)
}

function onPointerUp(event) {
  if (pressedOutside && isOutside(event)) {
    dialogRef.value.close()
    closeDialog()
  }
  pressedOutside = false
}

watch(side, loadContent)
watch(() => props.file, () => {
  side.value = 'new'
  loadContent()
})
onMounted(() => {
  dialogRef.value.showModal()
  loadContent()
})
onBeforeUnmount(() => {
  controller?.abort()
  clearTimeout(copyTimer)
})
</script>

<style scoped>
dialog[open] {
  display: flex;
  flex-direction: column;
}
dialog:modal {
  overscroll-behavior: contain;
}
dialog::backdrop {
  overflow: hidden;
  overscroll-behavior: contain;
  background: rgb(0 0 0 / 45%);
}
</style>
