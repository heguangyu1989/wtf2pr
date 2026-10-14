<template>
  <TokenGate v-if="!authed" @authenticated="onAuthenticated" />

  <div v-else class="min-h-screen bg-gray-50 dark:bg-gray-950 text-gray-900 dark:text-gray-100 flex flex-col h-screen overflow-hidden">
    <AppHeader
      :saved="saved"
      :saving="saving"
      :export-format="exportFormat"
      :review-label="reviewLabel"
      :work-dir="workDir"
      :active-tab="activeTab"
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
          @click="onTabClick(t.key)"
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
              class="border border-gray-300 dark:border-gray-700 rounded-lg px-2 py-1.5 text-sm w-72 bg-white dark:bg-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500/40 disabled:opacity-50"
              :disabled="!!switchingReview"
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
                  class="px-3 py-1.5 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition shadow-sm disabled:opacity-50"
                  :disabled="!!switchingReview"
                  @click="onSwitchHistoryReview(r)"
                >
                  {{ switchingReview === r.reviewID ? '切换中…' : '切换并编辑' }}
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
              @input="templateDirty = true"
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
      :has-unsaved="hasUnsavedAll"
      @close="showWorkdir = false"
      @switched="onWorkdirSwitched"
      @unauthorized="authed = false; showWorkdir = false"
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
import { ref, computed, watch, onMounted } from 'vue'
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
const saving = ref(false)
const reviewLabel = ref('')
const workDir = ref('')

const commitList = ref([])
const commitPage = ref(1)
const commitTotalPages = ref(1)
const reviewList = ref([])

const historyReviewData = ref(null)
const templateContent = ref('')
const templateDirty = ref(false)
const savingTemplate = ref(false)
const switchingReview = ref('')
const commitsLoaded = ref(false)

function guardAuth(e) {
  if (e && e.status === 401) {
    authed.value = false
    return true
  }
  return false
}

// 上下文世代号：在 tab 切换 / workdir 切换 / 重新认证 / 切换历史 review 时 +1，用于守卫评论、配置、历史列表等上下文数据的回写
let epoch = 0
// 重置流程计数（支持嵌套），>0 时抑制 watch，由重置方显式顺序加载
let resetting = 0
function stale(e) {
  return e !== epoch
}

// diff 加载序号：仅最新一次 loadDiff 负责写入 diffData 与复位 loading
let diffSeq = 0
// commit 列表加载序号：仅最新一次 loadCommits 生效
let commitSeq = 0

// 脏状态：评论有未保存的变更（含删光评论后的删除意图；统一判定，供确认拦截/头部徽标使用）
const isDirty = computed(() => !saved.value)
// 含模板草稿在内的全部未保存内容（供 workdir 切换确认与 beforeunload 拦截）
const hasUnsavedAll = computed(() => isDirty.value || templateDirty.value)

// 有未保存评论时请求用户确认
function confirmDiscardIfDirty() {
  if (isDirty.value) {
    return window.confirm('当前有未保存的评论，继续操作将丢弃，确认继续？')
  }
  return true
}

function onTabClick(key) {
  if (key === activeTab.value) return
  if (activeTab.value === 'commit') {
    if (!confirmDiscardIfDirty()) return
    // 用户已确认丢弃，评论与脏标记同步清理，避免后续误判未保存
    comments.value = []
    saved.value = true
  }
  if (activeTab.value === 'template' && templateDirty.value) {
    if (!window.confirm('模板有未保存的修改，离开将丢失，确认继续？')) return
    templateDirty.value = false
  }
  activeTab.value = key
}

watch(activeTab, async (tab) => {
  if (resetting > 0) return
  await loadTab(tab)
})

// 消费式跳过标记：程序性回写 hash 时置位，watcher 触发时消费一次（同步 resetting 窗口抑制不了异步 watcher）
let skipHashWatch = false

watch(selectedCommitHash, async (hash, oldHash) => {
  if (skipHashWatch) {
    skipHashWatch = false
    return
  }
  if (resetting > 0) return
  if (activeTab.value !== 'commit') return
  // 任何 commit 上下文变化（含清空为占位项）都先确认丢弃未保存评论
  if (!confirmDiscardIfDirty()) {
    skipHashWatch = true
    selectedCommitHash.value = oldHash
    return
  }
  comments.value = []
  saved.value = true
  if (!hash) {
    cancelDiff()
    return
  }
  await loadDiff('commit', hash)
})

// 标签页数据加载（watch 与重置补偿共用同一入口）
async function loadTab(tab) {
  const e = ++epoch
  if (tab === 'working') {
    comments.value = []
    saved.value = true
    await loadDiff('working')
  } else if (tab === 'commit') {
    if (!commitsLoaded.value) {
      await loadCommits()
    }
    if (selectedCommitHash.value) {
      await loadDiff('commit', selectedCommitHash.value)
    } else {
      cancelDiff()
    }
    try {
      const review = await getReview()
      if (stale(e)) return
      if (!selectedCommitHash.value && review.type === 'commit' && review.commit) {
        // 未选择 commit 时从当前 review 恢复上下文（初始化竞态下也能在此补齐）
        skipHashWatch = true
        selectedCommitHash.value = review.commit
        comments.value = review.comments || []
        ensureCommitOption(review.commit, review.diff?.commitInfo?.message)
        await loadDiff('commit', review.commit)
      } else if (review.type === 'commit' && review.commit && review.commit !== selectedCommitHash.value) {
        // 服务端评论归属于其他 commit，不展示，避免错位保存
        comments.value = []
      } else {
        comments.value = review.comments || []
      }
      // 评论来自服务端快照，与磁盘一致，标记为已保存
      saved.value = true
    } catch (err) {
      if (guardAuth(err)) return
      if (stale(e)) return
      comments.value = []
    }
  } else if (tab === 'history') {
    await loadReviews(e)
  } else if (tab === 'template') {
    await loadTemplate(e)
  }
}

// 恢复的 hash 不在当前 commit 列表页内时插入临时选项，避免下拉框显示空白
function ensureCommitOption(hash, message = '') {
  if (hash && !commitList.value.some(c => c.hash === hash)) {
    commitList.value = [{ hash, message: message || '(当前 review)', author: '', date: '' }, ...commitList.value]
  }
}

async function loadDiff(type, commit = '') {
  const seq = ++diffSeq
  loading.value = true
  error.value = ''
  try {
    const data = await getDiff(type, commit)
    if (seq !== diffSeq) return
    diffData.value = data
  } catch (err) {
    if (guardAuth(err)) return
    if (seq !== diffSeq) return
    error.value = err.message
    diffData.value = null
  } finally {
    if (seq === diffSeq) loading.value = false
  }
}

// 取消在途 diff 加载并清空展示
function cancelDiff() {
  diffSeq++
  diffData.value = null
  loading.value = false
  error.value = ''
}

async function loadCommits() {
  const seq = ++commitSeq
  try {
    const res = await getCommits(commitPage.value, 10)
    if (seq !== commitSeq) return
    commitList.value = res.list || []
    commitPage.value = res.page || 1
    commitTotalPages.value = res.totalPages || 1
    commitsLoaded.value = true
    // 列表整体替换后，恢复出的页外 hash 需要重新插入临时选项，避免下拉空白
    ensureCommitOption(selectedCommitHash.value)
  } catch (err) {
    if (guardAuth(err)) return
    if (seq !== commitSeq) return
    // 失败时清空页码状态，再次进入 commit 页可重试
    commitList.value = []
    commitPage.value = 1
    commitTotalPages.value = 1
  }
}

function changeCommitPage(delta) {
  commitPage.value += delta
  loadCommits()
}

async function loadReviews(e) {
  try {
    const list = await getReviews()
    if (stale(e)) return
    reviewList.value = list
  } catch (err) {
    if (guardAuth(err)) return
    if (stale(e)) return
    reviewList.value = []
  }
}

function onCommentsUpdate(list) {
  comments.value = list
  saved.value = false
}

async function createNewReview() {
  if (!confirmDiscardIfDirty()) return
  try {
    const res = await apiNewReview()
    comments.value = []
    saved.value = true
    reviewLabel.value = res.reviewID ? formatReviewLabel({ reviewID: res.reviewID }) : ''
    if (activeTab.value === 'history') {
      await loadReviews(epoch)
    }
  } catch (e) {
    if (guardAuth(e)) return
    alert('新建 Review 失败: ' + e.message)
  }
}

async function saveReview() {
  if (activeTab.value !== 'commit') return
  if (!selectedCommitHash.value) {
    alert('请先选择一个 Commit')
    return
  }
  if (saving.value) return
  saving.value = true
  // 版本快照：响应到达时若评论已被继续编辑（数组引用变化），不得误标"已保存"
  const snapshot = comments.value
  try {
    await apiSaveReview(snapshot, 'commit', selectedCommitHash.value)
    if (comments.value === snapshot) {
      saved.value = true
    }
  } catch (e) {
    if (guardAuth(e)) return
    alert('保存失败: ' + e.message)
  } finally {
    saving.value = false
  }
}

async function doExport() {
  // 仅在 working / commit 标签页可用，避免弹出陈旧内容
  if (activeTab.value !== 'working' && activeTab.value !== 'commit') return
  if (activeTab.value === 'commit' && !selectedCommitHash.value) {
    alert('请先选择一个 Commit')
    return
  }
  const commit = activeTab.value === 'commit' ? selectedCommitHash.value : ''
  try {
    exportResult.value = await exportReview(exportFormat.value, activeTab.value, commit)
    showExport.value = true
  } catch (e) {
    if (guardAuth(e)) return
    alert('导出失败: ' + e.message)
  }
}

async function onSwitchHistoryReview(reviewItem) {
  if (reviewItem.type !== 'commit' || !reviewItem.commitExists) return
  if (switchingReview.value) return
  if (!confirmDiscardIfDirty()) return
  switchingReview.value = reviewItem.reviewID
  try {
    let review
    try {
      await apiSwitchReview(reviewItem.reviewID)
    } catch (e) {
      if (guardAuth(e)) return
      alert('切换 Review 失败: ' + e.message)
      return
    }
    try {
      review = await getReview()
    } catch (e) {
      if (guardAuth(e)) return
      // 服务端其实已切换成功，仅状态同步失败，避免误导用户重试切换
      alert('已切换，但加载 Review 数据失败: ' + e.message)
      return
    }
    // 两步 RPC 之间可能发生并发切换：以 reviewID 校验服务端当前 review 与目标一致，防止评论错位
    // （旧版文件 reviewID 可能为空，空值视为 default）
    const actualID = review.reviewID || 'default'
    if (actualID !== reviewItem.reviewID) {
      alert('切换结果与预期不一致，请重试')
      return
    }
    // 抑制双 watcher，显式按顺序加载，保证评论与 diff 一致落位
    resetting++
    epoch++
    try {
      selectedCommitHash.value = reviewItem.commit || ''
      activeTab.value = 'commit'
      reviewLabel.value = formatReviewLabel({ reviewID: reviewItem.reviewID })
      saved.value = true
      comments.value = review.comments || []
      if (!commitsLoaded.value) {
        await loadCommits()
      }
      ensureCommitOption(selectedCommitHash.value, reviewItem.commitMsg)
      if (selectedCommitHash.value) {
        await loadDiff('commit', selectedCommitHash.value)
      }
    } finally {
      resetting--
      if (resetting === 0 && activeTab.value !== 'commit') {
        // 窗口内被抑制的 tab 点击需要补偿加载
        await loadTab(activeTab.value)
      }
    }
  } finally {
    switchingReview.value = ''
  }
}

let viewSeq = 0

async function onViewHistoryReview(reviewItem) {
  const e = epoch
  const seq = ++viewSeq
  try {
    const detail = await getReviewDetail(reviewItem.reviewID)
    // workdir 切换会使旧详情失效；同 epoch 内快速连点只保留最后一次
    if (stale(e) || seq !== viewSeq) return
    historyReviewData.value = detail
  } catch (err) {
    if (guardAuth(err)) return
    alert('加载 Review 详情失败: ' + err.message)
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

async function loadTemplate(e) {
  try {
    const res = await getTemplate()
    if (stale(e)) return
    templateContent.value = res.content || ''
    templateDirty.value = false
  } catch (err) {
    if (guardAuth(err)) return
    if (stale(e)) return
    templateContent.value = ''
  }
}

async function onSaveTemplate() {
  if (savingTemplate.value) return
  savingTemplate.value = true
  // 版本快照：保存往返期间继续输入的内容不得被响应误标为已保存
  const snapshot = templateContent.value
  try {
    await apiSaveTemplate(snapshot)
    if (templateContent.value === snapshot) {
      templateDirty.value = false
    }
    alert('模板保存成功')
  } catch (e) {
    if (guardAuth(e)) return
    alert('模板保存失败: ' + e.message)
  } finally {
    savingTemplate.value = false
  }
}

// 重置全部页面状态（workdir 切换与重新认证共用）
function resetState() {
  activeTab.value = 'working'
  selectedCommitHash.value = ''
  diffData.value = null
  comments.value = []
  commitList.value = []
  commitsLoaded.value = false
  commitPage.value = 1
  commitTotalPages.value = 1
  reviewList.value = []
  reviewLabel.value = ''
  templateContent.value = ''
  templateDirty.value = false
  historyReviewData.value = null
  showExport.value = false
  exportResult.value = null
  saved.value = true
  error.value = ''
}

// 在重置窗口内执行加载：抑制 watch，窗口结束后补偿加载被抑制的 tab 切换
async function withResetWindow(fn) {
  resetting++
  epoch++
  try {
    await fn()
  } finally {
    resetting--
    if (resetting === 0 && activeTab.value !== 'working') {
      await loadTab(activeTab.value)
    }
  }
}

async function onWorkdirSwitched(info) {
  await withResetWindow(async () => {
    workDir.value = info.workDir || ''
    // 工作目录切换后重置全部状态并按新目录重新加载
    resetState()
    await initialize()
  })
}

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  return d.toLocaleString()
}

async function initialize() {
  const e = ++epoch
  await loadDiff('working')
  // selectedCommitHash 的恢复由 loadTab('commit') 在进入 commit 页时统一处理
  try {
    const cfg = await getConfig()
    if (stale(e)) return
    workDir.value = cfg.workDir || workDir.value
    reviewLabel.value = formatReviewLabel(cfg)
  } catch (err) {
    if (guardAuth(err)) return
    if (stale(e)) return
    reviewLabel.value = ''
  }
  await loadReviews(e)
}

// formatReviewLabel 生成头部 review 标识文案
function formatReviewLabel(cfg) {
  if (cfg.reviewID && cfg.reviewID !== 'default') {
    return `Review ID: ${cfg.reviewID.substring(0, 8)}`
  }
  if (cfg.reviewID === 'default' || cfg.reviewFile === 'review.json') {
    return 'Review: default'
  }
  if (cfg.reviewFile) {
    return `Review: ${cfg.reviewFile}`
  }
  return ''
}

async function onAuthenticated() {
  // 401 可能发生在任意标签页，重新认证后统一回到 working 视图，避免状态错位
  if (isDirty.value) {
    window.alert('服务认证已更新，未保存的评论已被重置，请重新确认后再提交')
  }
  await withResetWindow(async () => {
    resetState()
    authed.value = true
    await initialize()
  })
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

// 有未保存内容（评论或模板草稿）时拦截页面关闭/刷新
function onBeforeUnload(e) {
  if (hasUnsavedAll.value) {
    e.preventDefault()
    e.returnValue = ''
  }
}
window.addEventListener('beforeunload', onBeforeUnload)
</script>
