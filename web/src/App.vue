<template>
  <TokenGate v-if="!authed" @authenticated="onAuthenticated" />

  <div v-else class="min-h-screen bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100 flex flex-col h-screen overflow-hidden">
    <AppHeader
      :saved="saved"
      :export-format="exportFormat"
      :review-label="reviewLabel"
      :work-dir="workDir"
      @save-review="saveReview"
      @do-export="doExport"
      @new-review="createNewReview"
      @show-help="showHelp = true"
      @update:export-format="exportFormat = $event"
      @switch-workdir="showWorkdir = true"
    />

    <!-- Tabs -->
    <div class="max-w-7xl w-full mx-auto px-4 pt-3 shrink-0">
      <div class="flex gap-1 border-b border-gray-200 dark:border-gray-800">
        <button
          v-for="t in tabs"
          :key="t.key"
          class="px-4 py-2 text-sm -mb-px border-b-2 transition"
          :class="activeTab === t.key
            ? 'border-blue-600 text-blue-600 dark:text-blue-400 dark:border-blue-400 font-medium'
            : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:border-gray-300 dark:hover:border-gray-600'"
          @click="activeTab = t.key"
        >
          {{ t.label }}
        </button>
      </div>
    </div>

    <main class="flex-1 max-w-7xl w-full mx-auto px-4 py-4 overflow-hidden">
      <!-- Working -->
      <template v-if="activeTab === 'working'">
        <div class="h-full">
          <DiffViewer
            :diff-data="diffData"
            :comments="[]"
            :loading="loading"
            :error="error"
            :readonly="true"
          />
        </div>
      </template>

      <!-- Commit -->
      <template v-if="activeTab === 'commit'">
        <div class="flex flex-col h-full gap-3">
          <div class="flex flex-wrap items-center gap-2 shrink-0 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl px-3 py-2 shadow-sm">
            <select
              v-model="selectedCommitHash"
              class="border border-gray-300 dark:border-gray-700 rounded-lg px-2 py-1.5 text-sm w-72 bg-white dark:bg-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
            >
              <option value="">选择 Commit</option>
              <option v-for="c in commitList" :key="c.hash" :value="c.hash">
                {{ c.hash.substring(0, 7) }} - {{ c.message }}
              </option>
            </select>
            <div class="flex items-center gap-1.5 text-sm">
              <button
                class="px-2.5 py-1.5 border border-gray-300 dark:border-gray-700 rounded-lg text-xs disabled:opacity-40 hover:bg-gray-50 dark:hover:bg-gray-800 transition"
                :disabled="commitPage <= 1"
                @click="changeCommitPage(-1)"
              >
                上一页
              </button>
              <span class="text-xs text-gray-500 whitespace-nowrap">{{ commitPage }} / {{ commitTotalPages }}</span>
              <button
                class="px-2.5 py-1.5 border border-gray-300 dark:border-gray-700 rounded-lg text-xs disabled:opacity-40 hover:bg-gray-50 dark:hover:bg-gray-800 transition"
                :disabled="commitPage >= commitTotalPages"
                @click="changeCommitPage(1)"
              >
                下一页
              </button>
            </div>
          </div>
          <div class="flex-1 overflow-hidden">
            <DiffViewer
              :diff-data="diffData"
              :comments="comments"
              :loading="loading"
              :error="error"
              :readonly="false"
              :commit-info="diffData?.commitInfo"
              @update:comments="onCommentsUpdate"
            />
          </div>
        </div>
      </template>

      <!-- History -->
      <template v-if="activeTab === 'history'">
        <div class="h-full overflow-y-auto">
          <div v-if="!reviewList.length" class="flex flex-col items-center justify-center py-20 text-gray-400 dark:text-gray-500">
            <svg class="w-12 h-12 mb-3 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" stroke-width="1" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div class="text-sm">当前工作目录暂无历史 Review</div>
          </div>
          <div v-else class="space-y-3 max-w-3xl">
            <div
              v-for="r in reviewList"
              :key="r.reviewID"
              class="border border-gray-200 dark:border-gray-800 rounded-xl p-4 bg-white dark:bg-gray-900 shadow-sm hover:shadow-md transition"
            >
              <div class="flex items-center justify-between">
                <div class="font-mono text-sm font-medium text-gray-800 dark:text-gray-200">{{ r.reviewID.substring(0, 8) }}</div>
                <div
                  class="text-xs px-2 py-0.5 rounded-full font-medium"
                  :class="r.type === 'commit'
                    ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                    : 'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-300'"
                >
                  {{ r.type === 'commit' ? 'Commit' : 'Working' }}
                </div>
              </div>
              <div v-if="r.commit" class="text-xs text-gray-500 mt-1.5">
                Commit: <span class="font-mono">{{ r.commit.substring(0, 7) }}</span>
                <span v-if="r.commitMsg" class="ml-1 text-gray-400">— {{ r.commitMsg }}</span>
                <span v-if="r.type === 'commit'" class="ml-2 font-medium" :class="r.commitExists ? 'text-green-600' : 'text-red-500'">
                  {{ r.commitExists ? '可编辑' : '已丢失' }}
                </span>
              </div>
              <div class="text-xs text-gray-500 mt-1">
                评论数: <span class="font-medium">{{ r.commentCount }}</span>
                <span v-if="r.updatedAt" class="ml-2">更新于: {{ formatTime(r.updatedAt) }}</span>
              </div>
              <div class="mt-3 flex items-center gap-2">
                <button
                  v-if="r.type === 'commit' && r.commitExists"
                  class="px-3 py-1.5 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition shadow-sm"
                  @click="onSwitchHistoryReview(r)"
                >
                  切换并编辑
                </button>
                <button
                  class="px-3 py-1.5 text-xs rounded-lg border border-gray-300 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800 transition"
                  @click="onViewHistoryReview(r)"
                >
                  查看详情
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- Template -->
      <template v-if="activeTab === 'template'">
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 h-full">
          <div class="lg:col-span-2 h-full flex flex-col bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl p-4 shadow-sm">
            <div class="mb-3 flex items-center justify-between shrink-0">
              <div class="text-sm font-medium">模板内容<span class="text-xs text-gray-400 ml-2 font-normal">存储于当前工作目录的数据区</span></div>
              <button class="px-3 py-1.5 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 transition shadow-sm" @click="onSaveTemplate">保存模板</button>
            </div>
            <textarea
              v-model="templateContent"
              class="flex-1 w-full border border-gray-300 dark:border-gray-700 rounded-lg p-3 text-sm font-mono bg-gray-50 dark:bg-gray-950 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
              placeholder="输入 Go text/template 模板..."
            />
          </div>
          <div class="lg:col-span-1 h-full overflow-y-auto border border-gray-200 dark:border-gray-800 rounded-xl p-4 bg-white dark:bg-gray-900 shadow-sm text-sm">
            <div class="font-medium mb-2">帮助说明</div>
            <p class="text-xs text-gray-500 mb-3">系统使用 Go 的 <code>text/template</code> 渲染模板，可用字段如下：</p>
            <pre v-pre class="text-xs bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 rounded-lg p-2 overflow-x-auto mb-3">.Type          // "working" 或 "commit"
.CommitInfo    // 可能为 nil
  .Hash
  .Author
  .Date
  .Message
.Files          // 仅包含有 review comment 的文件
  .Path
  .IsNew
  .IsDeleted
  .Comments
    .LineKey    // "old:12" / "new:15" / ""
    .LineNo     // 展示用的行号字符串
    .Content
    .CodeLine   // 对应行的代码内容
    .HunkHeader // "@@ -1,2 +1,3 @@"</pre>
            <div class="font-medium mb-2">示例模板</div>
            <pre v-pre class="text-xs bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 rounded-lg p-2 overflow-x-auto"># Review Report

Type: {{ .Type }}

{{if .CommitInfo}}Commit: {{.CommitInfo.Hash}}
Author: {{.CommitInfo.Author}}
Date: {{.CommitInfo.Date}}
Message: {{.CommitInfo.Message}}
{{end}}
{{range .Files}}
## {{.Path}}
{{if .IsNew}}(New file){{end}}
{{if .IsDeleted}}(Deleted file){{end}}
{{range .Comments}}
- {{.LineNo}}: {{.Content}}
{{if .CodeLine}}  Code: {{.CodeLine}}
{{end}}
{{end}}
{{end}}</pre>
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
    <WorkdirModal
      v-if="showWorkdir"
      @close="showWorkdir = false"
      @switched="onWorkdirSwitched"
    />
    <ReviewResultModal
      v-if="historyReviewData"
      :data="historyReviewData"
      @close="closeReviewResultModal"
      @export="onExportHistoryReview"
    />

    <footer class="shrink-0 border-t border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 px-4 py-2 text-xs text-gray-400 dark:text-gray-500 text-center">
      wtf2pr — code review made simple
    </footer>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import AppHeader from './components/AppHeader.vue'
import DiffViewer from './components/DiffViewer.vue'
import ExportModal from './components/ExportModal.vue'
import HelpModal from './components/HelpModal.vue'
import TokenGate from './components/TokenGate.vue'
import WorkdirModal from './components/WorkdirModal.vue'
import ReviewResultModal from './components/ReviewResultModal.vue'
import { getDiff, getReview, saveReview as apiSaveReview, newReview as apiNewReview, switchReview as apiSwitchReview, getReviewDetail, getReviews, exportReview, getCommits, getConfig, getTemplate, saveTemplate as apiSaveTemplate, checkAuth } from './api/client.js'

const tabs = [
  { key: 'working', label: 'Working tree' },
  { key: 'commit', label: 'Commit' },
  { key: 'history', label: '历史 Review' },
  { key: 'template', label: '导出模板' },
]

const authed = ref(false)
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
const showWorkdir = ref(false)
const saved = ref(true)
const reviewLabel = ref('')
const workDir = ref('')

const commitList = ref([])
const commitPage = ref(1)
const commitTotalPages = ref(1)
const reviewList = ref([])

const historyReviewData = ref(null)
const templateContent = ref('')

function guardAuth(e) {
  if (e && e.status === 401) {
    authed.value = false
    return true
  }
  return false
}

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
    } catch (e) {
      if (guardAuth(e)) return
      comments.value = []
    }
  } else if (tab === 'history') {
    await loadReviews()
  } else if (tab === 'template') {
    await loadTemplate()
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
    if (guardAuth(e)) return
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
  } catch (e) {
    if (guardAuth(e)) return
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
  } catch (e) {
    if (guardAuth(e)) return
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
    reviewLabel.value = res.reviewID ? `Review ID: ${res.reviewID.substring(0, 8)}` : ''
    if (activeTab.value === 'history') {
      await loadReviews()
    }
  } catch (e) {
    if (guardAuth(e)) return
    alert('新建 Review 失败: ' + e.message)
  }
}

async function saveReview() {
  if (activeTab.value !== 'commit') return
  try {
    await apiSaveReview(comments.value, 'commit', selectedCommitHash.value)
    saved.value = true
  } catch (e) {
    if (guardAuth(e)) return
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
    if (guardAuth(e)) return
    alert('导出失败: ' + e.message)
  }
}

async function onSwitchHistoryReview(reviewItem) {
  if (reviewItem.type !== 'commit' || !reviewItem.commitExists) return
  try {
    await apiSwitchReview(reviewItem.reviewID)
    selectedCommitHash.value = reviewItem.commit || ''
    activeTab.value = 'commit'
    reviewLabel.value = `Review ID: ${reviewItem.reviewID.substring(0, 8)}`
    saved.value = true
  } catch (e) {
    if (guardAuth(e)) return
    alert('切换 Review 失败: ' + e.message)
  }
}

async function onViewHistoryReview(reviewItem) {
  try {
    const detail = await getReviewDetail(reviewItem.reviewID)
    historyReviewData.value = detail
  } catch (e) {
    if (guardAuth(e)) return
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
    if (guardAuth(e)) return
    alert('导出失败: ' + e.message)
  }
}

async function loadTemplate() {
  try {
    const res = await getTemplate()
    templateContent.value = res.content || ''
  } catch (e) {
    if (guardAuth(e)) return
    templateContent.value = ''
  }
}

async function onSaveTemplate() {
  try {
    await apiSaveTemplate(templateContent.value)
    alert('模板保存成功')
  } catch (e) {
    if (guardAuth(e)) return
    alert('模板保存失败: ' + e.message)
  }
}

async function onWorkdirSwitched(info) {
  workDir.value = info.workDir || ''
  // 工作目录切换后重置全部状态并按新目录重新加载
  activeTab.value = 'working'
  selectedCommitHash.value = ''
  diffData.value = null
  comments.value = []
  commitList.value = []
  commitPage.value = 1
  commitTotalPages.value = 1
  reviewList.value = []
  reviewLabel.value = ''
  saved.value = true
  error.value = ''
  await initialize()
}

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  return d.toLocaleString()
}

async function initialize() {
  await loadDiff('working')
  try {
    const review = await getReview()
    if (review.type === 'commit' && review.commit) {
      selectedCommitHash.value = review.commit
    }
  } catch (e) {
    if (guardAuth(e)) return
  }
  try {
    const cfg = await getConfig()
    workDir.value = cfg.workDir || workDir.value
    if (cfg.reviewID) {
      reviewLabel.value = `Review ID: ${cfg.reviewID.substring(0, 8)}`
    } else if (cfg.reviewFile) {
      reviewLabel.value = cfg.reviewFile === 'review.json' ? 'Review: default' : `Review: ${cfg.reviewFile}`
    }
  } catch (e) {
    if (guardAuth(e)) return
    reviewLabel.value = ''
  }
  await loadReviews()
}

async function onAuthenticated() {
  authed.value = true
  await initialize()
}

onMounted(async () => {
  try {
    await checkAuth()
    authed.value = true
    await initialize()
  } catch (e) {
    if (e && e.status === 401) {
      authed.value = false
    } else {
      // 服务未启用鉴权或网络异常时仍尝试进入
      authed.value = true
      await initialize()
    }
  }
})
</script>
