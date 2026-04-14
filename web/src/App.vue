<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100 flex flex-col h-screen overflow-hidden">
    <AppHeader
      :saved="saved"
      :export-format="exportFormat"
      :review-label="reviewLabel"
      @save-review="saveReview"
      @do-export="doExport"
      @new-review="createNewReview"
      @show-help="showHelp = true"
      @update:export-format="exportFormat = $event"
    />

    <!-- Tabs -->
    <div class="max-w-7xl w-full mx-auto px-4 pt-2 shrink-0">
      <div class="flex gap-1 border-b dark:border-gray-700">
        <button
          v-for="t in tabs"
          :key="t.key"
          class="px-4 py-2 text-sm rounded-t transition"
          :class="activeTab === t.key ? 'bg-white dark:bg-gray-900 border-t border-x dark:border-gray-700 font-medium text-blue-600' : 'text-gray-500 hover:text-gray-700'"
          @click="activeTab = t.key"
        >
          {{ t.label }}
        </button>
      </div>
    </div>

    <main class="flex-1 max-w-7xl w-full mx-auto px-4 py-4 overflow-hidden">
      <!-- Working -->
      <template v-if="activeTab === 'working'">
        <DiffViewer
          :diff-data="diffData"
          :comments="[]"
          :loading="loading"
          :error="error"
          :readonly="true"
        />
      </template>

      <!-- Commit -->
      <template v-if="activeTab === 'commit'">
        <div class="mb-3 flex flex-wrap items-center gap-2 shrink-0">
          <select
            v-model="selectedCommitHash"
            class="border rounded px-2 py-1 text-sm w-64 bg-white dark:bg-gray-900"
          >
            <option value="">选择 Commit</option>
            <option v-for="c in commitList" :key="c.hash" :value="c.hash">
              {{ c.hash.substring(0, 7) }} - {{ c.message }}
            </option>
          </select>
          <div class="flex items-center gap-1 text-sm">
            <button
              class="px-2 py-1 border rounded disabled:opacity-40 hover:bg-gray-50 dark:hover:bg-gray-800"
              :disabled="commitPage <= 1"
              @click="changeCommitPage(-1)"
            >
              上一页
            </button>
            <span class="text-xs text-gray-500 whitespace-nowrap">{{ commitPage }} / {{ commitTotalPages }}</span>
            <button
              class="px-2 py-1 border rounded disabled:opacity-40 hover:bg-gray-50 dark:hover:bg-gray-800"
              :disabled="commitPage >= commitTotalPages"
              @click="changeCommitPage(1)"
            >
              下一页
            </button>
          </div>
        </div>
        <DiffViewer
          :diff-data="diffData"
          :comments="comments"
          :loading="loading"
          :error="error"
          :readonly="false"
          :commit-info="diffData?.commitInfo"
          @update:comments="onCommentsUpdate"
        />
      </template>

      <!-- History -->
      <template v-if="activeTab === 'history'">
        <div class="h-full overflow-y-auto">
          <div v-if="!reviewList.length" class="text-sm text-gray-500 text-center py-12">暂无历史 Review</div>
          <div v-else class="space-y-3 max-w-3xl">
            <div
              v-for="r in reviewList"
              :key="r.reviewID"
              class="border rounded-lg p-4 bg-white dark:bg-gray-900"
            >
              <div class="flex items-center justify-between">
                <div class="font-mono text-sm font-medium">{{ r.reviewID.substring(0, 8) }}</div>
                <div class="text-xs px-2 py-0.5 rounded bg-gray-100 dark:bg-gray-800">
                  {{ r.type === 'commit' ? 'Commit' : 'Working' }}
                </div>
              </div>
              <div v-if="r.commit" class="text-xs text-gray-500 mt-1">
                Commit: {{ r.commit.substring(0, 7) }}
                <span v-if="r.commitMsg" class="ml-1 text-gray-400">— {{ r.commitMsg }}</span>
                <span v-if="r.type === 'commit'" class="ml-2" :class="r.commitExists ? 'text-green-600' : 'text-red-500'">
                  {{ r.commitExists ? '可编辑' : '已丢失' }}
                </span>
              </div>
              <div class="text-xs text-gray-500 mt-1">
                评论数: {{ r.commentCount }}
                <span v-if="r.updatedAt" class="ml-2">更新于: {{ formatTime(r.updatedAt) }}</span>
              </div>
              <div class="mt-3 flex items-center gap-2">
                <button
                  v-if="r.type === 'commit' && r.commitExists"
                  class="px-3 py-1 text-xs rounded bg-blue-600 text-white hover:bg-blue-700"
                  @click="onSwitchHistoryReview(r)"
                >
                  切换并编辑
                </button>
                <button
                  class="px-3 py-1 text-xs rounded border hover:bg-gray-50 dark:hover:bg-gray-800"
                  @click="onViewHistoryReview(r)"
                >
                  查看详情
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>
    </main>

    <ExportModal
      v-if="showExport"
      :format="exportResult?.format"
      :content="exportResult?.content"
      @close="showExport = false"
    />
    <HelpModal v-if="showHelp" @close="showHelp = false" />
    <ReviewResultModal
      v-if="historyReviewData"
      :data="historyReviewData"
      @close="closeReviewResultModal"
      @export="onExportHistoryReview"
    />
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import AppHeader from './components/AppHeader.vue'
import DiffViewer from './components/DiffViewer.vue'
import ExportModal from './components/ExportModal.vue'
import HelpModal from './components/HelpModal.vue'
import ReviewResultModal from './components/ReviewResultModal.vue'
import { getDiff, getReview, saveReview as apiSaveReview, newReview as apiNewReview, switchReview as apiSwitchReview, getReviewDetail, getReviews, exportReview, getCommits, getConfig } from './api/client.js'

const tabs = [
  { key: 'working', label: 'Working tree' },
  { key: 'commit', label: 'Commit' },
  { key: 'history', label: '历史 Review' },
]

const activeTab = ref('working')
const selectedCommitHash = ref('')
const diffData = ref(null)
const loading = ref(false)
const error = ref('')
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
const reviewList = ref([])

const historyReviewData = ref(null)

watch(activeTab, async (tab) => {
  if (tab === 'working') {
    comments.value = []
    diffData.value = null
    await loadDiff('working')
  } else if (tab === 'commit') {
    if (commitList.value.length === 0) {
      await loadCommits()
    }
    if (selectedCommitHash.value) {
      await loadDiff('commit', selectedCommitHash.value)
    } else {
      diffData.value = null
    }
    try {
      const review = await getReview()
      comments.value = review.comments || []
    } catch {
      comments.value = []
    }
  } else if (tab === 'history') {
    await loadReviews()
  }
})

watch(selectedCommitHash, async (hash) => {
  if (activeTab.value !== 'commit') return
  if (!hash) {
    diffData.value = null
    return
  }
  await loadDiff('commit', hash)
})

async function loadDiff(type, commit = '') {
  loading.value = true
  error.value = ''
  try {
    if (type === 'commit' && !commit) {
      error.value = '请选择一个 Commit'
      diffData.value = null
      loading.value = false
      return
    }
    diffData.value = await getDiff(type, commit)
  } catch (e) {
    error.value = e.message
    diffData.value = null
  } finally {
    loading.value = false
  }
}

async function loadCommits() {
  try {
    const res = await getCommits(commitPage.value, 10)
    commitList.value = res.list || []
    commitPage.value = res.page || 1
    commitTotalPages.value = res.totalPages || 1
  } catch {
    commitList.value = []
  }
}

function changeCommitPage(delta) {
  commitPage.value += delta
  loadCommits()
}

async function loadReviews() {
  try {
    reviewList.value = await getReviews()
  } catch {
    reviewList.value = []
  }
}

function onCommentsUpdate(list) {
  comments.value = list
  saved.value = false
}

async function createNewReview() {
  try {
    const res = await apiNewReview()
    comments.value = []
    saved.value = true
    reviewLabel.value = res.reviewID ? `Review ID: ${res.reviewID}` : ''
    if (activeTab.value === 'history') {
      await loadReviews()
    }
  } catch (e) {
    alert('新建 Review 失败: ' + e.message)
  }
}

async function saveReview() {
  if (activeTab.value !== 'commit') return
  try {
    await apiSaveReview(comments.value, 'commit', selectedCommitHash.value)
    saved.value = true
  } catch (e) {
    alert('保存失败: ' + e.message)
  }
}

async function doExport() {
  try {
    if (activeTab.value === 'working') {
      exportResult.value = await exportReview(exportFormat.value, 'working', '')
    } else if (activeTab.value === 'commit') {
      exportResult.value = await exportReview(exportFormat.value, 'commit', selectedCommitHash.value)
    }
    showExport.value = true
  } catch (e) {
    alert('导出失败: ' + e.message)
  }
}

async function onSwitchHistoryReview(reviewItem) {
  if (reviewItem.type !== 'commit' || !reviewItem.commitExists) return
  try {
    await apiSwitchReview(reviewItem.reviewID)
    selectedCommitHash.value = reviewItem.commit || ''
    activeTab.value = 'commit'
    reviewLabel.value = `Review ID: ${reviewItem.reviewID}`
    saved.value = true
  } catch (e) {
    alert('切换 Review 失败: ' + e.message)
  }
}

async function onViewHistoryReview(reviewItem) {
  try {
    const detail = await getReviewDetail(reviewItem.reviewID)
    historyReviewData.value = detail
  } catch (e) {
    alert('加载 Review 详情失败: ' + e.message)
  }
}

function closeReviewResultModal() {
  historyReviewData.value = null
}

async function onExportHistoryReview(format) {
  if (!historyReviewData.value) return
  const d = historyReviewData.value
  try {
    exportResult.value = await exportReview(format, d.type, d.commit || '', d.reviewID)
    showExport.value = true
  } catch (e) {
    alert('导出失败: ' + e.message)
  }
}

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  return d.toLocaleString()
}

onMounted(async () => {
  await loadDiff('working')
  try {
    const review = await getReview()
    if (review.type === 'commit' && review.commit) {
      selectedCommitHash.value = review.commit
    }
  } catch {
    // ignore
  }
  try {
    const cfg = await getConfig()
    if (cfg.reviewID) {
      reviewLabel.value = `Review ID: ${cfg.reviewID}`
    } else if (cfg.reviewFile) {
      reviewLabel.value = cfg.reviewFile === 'review.json' ? 'Review: default' : `Review: ${cfg.reviewFile}`
    }
  } catch {
    reviewLabel.value = ''
  }
  await loadReviews()
})
</script>
