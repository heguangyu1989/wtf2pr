<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100">
    <header class="border-b bg-white dark:bg-gray-900 sticky top-0 z-10">
      <div class="max-w-7xl mx-auto px-4 py-3 flex flex-wrap items-center gap-3">
        <h1 class="text-lg font-semibold">wtf2pr</h1>
        <div class="flex items-center gap-2">
          <select v-model="diffType" class="border rounded px-2 py-1 text-sm bg-white dark:bg-gray-900">
            <option value="working">Working tree</option>
            <option value="commit">Commit</option>
          </select>
          <input
            v-if="diffType === 'commit'"
            v-model="commitHash"
            placeholder="commit hash"
            class="border rounded px-2 py-1 text-sm w-40 bg-white dark:bg-gray-900"
          />
          <button class="px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700" @click="loadDiff">加载</button>
        </div>
        <div class="flex-1"></div>
        <div class="flex items-center gap-2">
          <select v-model="exportFormat" class="border rounded px-2 py-1 text-sm bg-white dark:bg-gray-900">
            <option value="markdown">Markdown</option>
            <option value="json">JSON</option>
            <option value="xml">XML</option>
          </select>
          <button class="px-3 py-1 border text-sm rounded hover:bg-gray-50 dark:hover:bg-gray-800" @click="doExport">导出</button>
          <button class="px-3 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700" @click="saveReview">保存 Review</button>
        </div>
      </div>
    </header>

    <main class="max-w-7xl mx-auto px-4 py-4">
      <div v-if="loading" class="text-sm text-gray-500">加载中...</div>
      <div v-else-if="error" class="text-sm text-red-600">{{ error }}</div>
      <div v-else-if="!diffData || !diffData.files.length" class="text-sm text-gray-500">暂无 diff 数据</div>

      <div v-else class="grid grid-cols-1 lg:grid-cols-4 gap-4">
        <aside class="lg:col-span-1">
          <div class="border rounded-lg bg-white dark:bg-gray-900 overflow-hidden">
            <div class="px-3 py-2 border-b bg-gray-100 dark:bg-gray-800 text-sm font-medium">
              文件 ({{ diffData.files.length }})
            </div>
            <ul class="max-h-[calc(100vh-140px)] overflow-y-auto">
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

        <section class="lg:col-span-3 space-y-4">
          <div v-if="commitInfo" class="border rounded-lg p-3 bg-white dark:bg-gray-900 text-sm space-y-1">
            <div><span class="text-gray-500">Commit:</span> {{ commitInfo.hash }}</div>
            <div><span class="text-gray-500">Author:</span> {{ commitInfo.author }}</div>
            <div><span class="text-gray-500">Date:</span> {{ commitInfo.date }}</div>
            <div><span class="text-gray-500">Message:</span> {{ commitInfo.message }}</div>
          </div>

          <DiffFile
            v-if="selectedFile"
            :file="selectedFile"
            :comments="comments"
            @update:comments="comments = $event"
          />
        </section>
      </div>
    </main>

    <!-- Export Modal -->
    <div v-if="showExport" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 px-4">
      <div class="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-3xl max-h-[80vh] flex flex-col">
        <div class="px-4 py-3 border-b flex items-center justify-between">
          <div class="font-medium">导出结果 ({{ exportResult?.format }})</div>
          <button class="text-gray-500 hover:text-gray-700" @click="showExport = false">关闭</button>
        </div>
        <div class="p-4 overflow-auto flex-1">
          <pre class="text-xs font-mono whitespace-pre-wrap bg-gray-100 dark:bg-gray-800 p-3 rounded">{{ exportResult?.content }}</pre>
        </div>
        <div class="px-4 py-3 border-t flex justify-end gap-2">
          <button class="px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700" @click="copyExport">复制到剪贴板</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import DiffFile from './components/DiffFile.vue'
import { getDiff, getReview, saveReview as apiSaveReview, exportReview } from './api/client.js'

const diffType = ref('working')
const commitHash = ref('')
const diffData = ref(null)
const loading = ref(false)
const error = ref('')
const selectedIndex = ref(0)
const comments = ref([])
const exportFormat = ref('markdown')
const showExport = ref(false)
const exportResult = ref(null)

const commitInfo = computed(() => diffData.value?.commitInfo || null)
const selectedFile = computed(() => diffData.value?.files[selectedIndex.value] || null)

async function loadDiff() {
  loading.value = true
  error.value = ''
  try {
    diffData.value = await getDiff(diffType.value, commitHash.value)
    selectedIndex.value = 0
  } catch (e) {
    error.value = e.message
    diffData.value = null
  } finally {
    loading.value = false
  }
}

async function saveReview() {
  try {
    await apiSaveReview(comments.value)
    alert('Review 已保存')
  } catch (e) {
    alert('保存失败: ' + e.message)
  }
}

async function doExport() {
  try {
    exportResult.value = await exportReview(exportFormat.value, diffType.value, commitHash.value)
    showExport.value = true
  } catch (e) {
    alert('导出失败: ' + e.message)
  }
}

function copyExport() {
  if (!exportResult.value) return
  navigator.clipboard.writeText(exportResult.value.content).then(() => alert('已复制'))
}

onMounted(async () => {
  await loadDiff()
  try {
    comments.value = await getReview()
  } catch {
    comments.value = []
  }
})
</script>
