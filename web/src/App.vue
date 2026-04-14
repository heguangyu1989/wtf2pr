<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100 flex flex-col h-screen overflow-hidden">
    <AppHeader
      :diff-type="diffType"
      :selected-commit-hash="selectedCommitHash"
      :commit-list="commitList"
      :commit-page="commitPage"
      :commit-total-pages="commitTotalPages"
      :review-label="reviewLabel"
      :saved="saved"
      :export-format="exportFormat"
      @update:diff-type="diffType = $event"
      @update:selected-commit-hash="selectedCommitHash = $event"
      @change-commit-page="changeCommitPage"
      @show-help="showHelp = true"
      @update:export-format="exportFormat = $event"
      @do-export="doExport"
      @save-review="saveReview"
    />

    <main class="flex-1 max-w-7xl w-full mx-auto px-4 py-4 overflow-hidden">
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

          <DiffFile
            v-if="selectedFile"
            :file="selectedFile"
            :comments="comments"
            @update:comments="onCommentsUpdate"
          />
        </section>
      </div>
    </main>

    <ExportModal
      v-if="showExport"
      :format="exportResult?.format"
      :content="exportResult?.content"
      @close="showExport = false"
    />

    <HelpModal v-if="showHelp" @close="showHelp = false" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import AppHeader from './components/AppHeader.vue'
import DiffFile from './components/DiffFile.vue'
import ExportModal from './components/ExportModal.vue'
import HelpModal from './components/HelpModal.vue'
import { getDiff, getReview, saveReview as apiSaveReview, exportReview, getCommits, getConfig } from './api/client.js'

const diffType = ref('working')
const selectedCommitHash = ref('')
const diffData = ref(null)
const loading = ref(false)
const error = ref('')
const selectedIndex = ref(0)
const comments = ref([])
const exportFormat = ref('markdown')
const showExport = ref(false)
const exportResult = ref(null)
const showHelp = ref(false)
const saved = ref(true)
const reviewLabel = ref('')

const commitList = ref([])
const commitPage = ref(1)
const commitTotalPages = ref(1)

const commitInfo = computed(() => diffData.value?.commitInfo || null)
const selectedFile = computed(() => diffData.value?.files[selectedIndex.value] || null)

async function loadCommits() {
  if (diffType.value !== 'commit') return
  try {
    const res = await getCommits(commitPage.value, 10)
    commitList.value = res.list || []
    commitPage.value = res.page || 1
    commitTotalPages.value = res.totalPages || 1
  } catch (e) {
    commitList.value = []
  }
}

function changeCommitPage(delta) {
  commitPage.value += delta
  loadCommits()
}

watch(diffType, async (val) => {
  if (val === 'commit') {
    await loadCommits()
  }
  await loadDiff()
})

watch(selectedCommitHash, async () => {
  await loadDiff()
})

async function loadDiff() {
  loading.value = true
  error.value = ''
  try {
    if (diffType.value === 'commit' && !selectedCommitHash.value) {
      error.value = '请选择一个 Commit'
      diffData.value = null
      loading.value = false
      return
    }
    diffData.value = await getDiff(diffType.value, selectedCommitHash.value)
    selectedIndex.value = 0
  } catch (e) {
    error.value = e.message
    diffData.value = null
  } finally {
    loading.value = false
  }
}

function onCommentsUpdate(list) {
  comments.value = list
  saved.value = false
}

async function saveReview() {
  try {
    await apiSaveReview(comments.value)
    saved.value = true
  } catch (e) {
    alert('保存失败: ' + e.message)
  }
}

async function doExport() {
  try {
    exportResult.value = await exportReview(exportFormat.value, diffType.value, selectedCommitHash.value)
    showExport.value = true
  } catch (e) {
    alert('导出失败: ' + e.message)
  }
}

onMounted(async () => {
  await loadDiff()
  try {
    comments.value = await getReview()
  } catch {
    comments.value = []
  }
  try {
    const cfg = await getConfig()
    if (cfg.reviewID) {
      reviewLabel.value = `Review ID: ${cfg.reviewID}`
    } else if (cfg.reviewFile) {
      if (cfg.reviewFile === 'review.json') {
        reviewLabel.value = 'Review: default'
      } else {
        reviewLabel.value = `Review: ${cfg.reviewFile}`
      }
    }
  } catch {
    reviewLabel.value = ''
  }
})
</script>
