<template>
  <div class="flex h-dvh min-h-0 flex-col overflow-hidden bg-[var(--ground)] text-[var(--text)]">
    <AppHeader
      :saved="saved"
      :export-format="exportFormat"
      :review-label="reviewLabel"
      :can-save="canSave"
      :can-export="canExport"
      :saving="saving"
      save-reason="只有选择具体 Commit 后才能保存 Review"
      @save-review="saveReview"
      @do-export="doExport"
      @new-review="createNewReview"
      @show-help="showHelp = true"
      @update:export-format="exportFormat = $event"
    />

    <div class="w-full shrink-0 overflow-x-auto px-4 lg:px-5">
      <nav class="flex min-w-max gap-5" aria-label="主导航">
        <button
          v-for="t in tabs"
          :key="t.key"
          class="min-h-9 border-b-2 px-0.5 text-sm transition"
          :class="activeTab === t.key ? 'border-[var(--accent)] font-medium text-[var(--text)]' : 'border-transparent text-[var(--muted)] hover:text-[var(--text)]'"
          :aria-current="activeTab === t.key ? 'page' : undefined"
          @click="activeTab = t.key"
        >
          {{ t.label }}
        </button>
      </nav>
    </div>

    <main class="app-sheet mx-3 mb-3 min-h-0 w-auto flex-1 overflow-hidden p-4">
      <template v-if="activeTab === 'review'">
        <ReviewWorkspace
          v-model:target="reviewTarget"
          :comments="comments"
          :review-hydrated="reviewHydrated"
          :review-load-error="reviewLoadError"
          :work-dir="currentWorkdir"
          :workdir-changing="workdirChanging"
          :workdir-error="workdirError"
          @switch-workdir="onSwitchWorkdir"
          @retry-review-load="retryReviewLoad"
          @update:comments="onCommentsUpdate"
        />
      </template>

      <template v-if="activeTab === 'history'">
        <div class="h-full overflow-y-auto">
          <header class="mb-5">
            <h1 class="text-[22px] font-semibold tracking-[-0.02em]">历史 Review</h1>
            <p class="mt-1 text-xs text-[var(--muted)]">继续一次 Commit 审查，或查看已经保存的意见</p>
          </header>
          <div v-if="!reviewList.length" class="rounded-lg border border-dashed py-12 text-center text-sm text-[var(--muted)]">暂无历史 Review</div>
          <div v-else class="grid items-start gap-3 md:grid-cols-2 2xl:grid-cols-3">
            <article
              v-for="r in reviewList"
              :key="r.reviewID"
              class="rounded-lg border bg-[var(--surface)] p-4 dark:border-gray-700"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="truncate font-mono text-sm font-medium" :title="r.reviewID">{{ r.reviewID.substring(0, 8) }}</div>
                <div class="shrink-0 rounded bg-[var(--raised)] px-2 py-0.5 text-xs text-[var(--muted)]">
                  {{ r.type === 'commit' ? 'Commit' : 'Working' }}
                </div>
              </div>
              <div v-if="r.commit" class="mt-2 text-xs text-[var(--muted)]">
                <span class="font-mono">{{ r.commit.substring(0, 7) }}</span>
                <span v-if="r.commitMsg" class="ml-1">— {{ r.commitMsg }}</span>
                <span v-if="r.type === 'commit'" class="ml-2" :class="r.commitExists ? 'text-[var(--positive)]' : 'text-[var(--negative)]'">
                  {{ r.commitExists ? '可编辑' : '已丢失' }}
                </span>
              </div>
              <div class="mt-2 text-xs tabular-nums text-[var(--muted)]">
                {{ r.commentCount }} 条评论
                <span v-if="r.updatedAt" class="ml-2">更新于 {{ formatTime(r.updatedAt) }}</span>
              </div>
              <div class="mt-4 flex items-center gap-2">
                <button
                  v-if="r.type === 'commit' && r.commitExists"
                  class="control-primary min-h-8 rounded-md px-3 text-xs disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="saving"
                  @click="onSwitchHistoryReview(r)"
                >
                  切换并编辑
                </button>
                <button
                  class="min-h-8 rounded-md border border-[var(--control)] px-3 text-xs hover:bg-[var(--raised)]"
                  @click="onViewHistoryReview(r)"
                >
                  查看详情
                </button>
              </div>
            </article>
          </div>
        </div>
      </template>

      <template v-if="activeTab === 'template'">
        <div class="flex h-full min-h-0 flex-col">
          <header class="mb-4 flex flex-wrap items-end justify-between gap-3">
            <div>
              <h1 class="text-[22px] font-semibold tracking-[-0.02em]">导出模板</h1>
              <p class="mt-1 text-xs text-[var(--muted)]">使用 Go text/template 决定 Review 报告的结构</p>
            </div>
            <button class="control-primary min-h-8 rounded-md px-3 text-xs" :disabled="templateLoading || templateSaving" @click="onSaveTemplate">{{ templateSaving ? '保存中…' : '保存模板' }}</button>
          </header>
          <div v-if="templateLoading" class="mb-3 text-xs text-[var(--muted)]" role="status">正在加载模板…</div>
          <div v-else-if="templateError" class="mb-3 flex items-center gap-2 text-xs text-[var(--negative)]" role="alert">
            <span>模板加载失败：{{ templateError }}。当前编辑内容已保留。</span>
            <button type="button" class="underline" @click="loadTemplate">重试</button>
          </div>
          <div class="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(280px,1fr)]">
            <div class="flex min-h-0 flex-col">
              <label for="export-template" class="mb-2 text-xs font-medium text-[var(--muted)]">模板内容 · ~/.wtf2pr/export.tpl</label>
              <textarea
                id="export-template"
                v-model="templateContent"
                class="min-h-0 w-full flex-1 rounded-lg border border-[var(--control)] bg-[var(--surface)] p-3 text-sm font-mono"
                placeholder="输入 Go text/template 模板..."
              />
            </div>
            <aside class="min-h-0 overflow-y-auto rounded-lg bg-[var(--raised)] p-4 text-sm" aria-label="模板字段说明">
              <h2 class="mb-2 font-medium">可用字段</h2>
              <p class="mb-3 text-xs text-[var(--muted)]">系统使用 Go 的 <code>text/template</code> 渲染模板：</p>
            <pre v-pre class="text-xs bg-white dark:bg-gray-900 border rounded p-2 overflow-x-auto mb-3">.Type          // "working"、"commit" 或 "branch"
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
            <pre v-pre class="text-xs bg-white dark:bg-gray-900 border rounded p-2 overflow-x-auto"># Review Report

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
            </aside>
          </div>
        </div>
      </template>
    </main>

    <ExportModal
      v-if="showExport"
      :format="exportResult?.format"
      :content="exportResult?.content"
      @close="closeExportResult"
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
import { computed, ref, watch, onMounted, nextTick } from 'vue'
import AppHeader from './components/AppHeader.vue'
import ReviewWorkspace from './components/ReviewWorkspace.vue'
import ExportModal from './components/ExportModal.vue'
import HelpModal from './components/HelpModal.vue'
import ReviewResultModal from './components/ReviewResultModal.vue'
import { getReview, saveReview as apiSaveReview, newReview as apiNewReview, switchReview as apiSwitchReview, getReviewDetail, getReviews, exportReview, getConfig, getTemplate, saveTemplate as apiSaveTemplate, switchWorkdir as apiSwitchWorkdir } from './api/client.js'

const tabs = [
  { key: 'review', label: '代码审查' },
  { key: 'history', label: '历史 Review' },
  { key: 'template', label: '导出模板' },
]

const activeTab = ref('review')
const reviewTarget = ref({ type: 'working', commit: '', base: '' })
const comments = ref([])
const exportFormat = ref('markdown')
const showExport = ref(false)
const exportResult = ref(null)
const showHelp = ref(false)
const saved = ref(true)
const reviewLabel = ref('')
const reviewList = ref([])
const historyReviewData = ref(null)
const templateContent = ref('')
const templateLoading = ref(false)
const templateSaving = ref(false)
const templateError = ref('')
const templateDirty = ref(false)
const saving = ref(false)
const reviewHydrated = ref(false)
const reviewLoadError = ref('')
const currentWorkdir = ref('')
const workdirChanging = ref(false)
const workdirError = ref('')
let reviewLoadSerial = 0
let commentRevision = 0
let historyDetailSerial = 0
let templateLoadSerial = 0
let templateRevision = 0
let templatePendingLoads = 0

watch(reviewTarget, async target => {
  const serial = ++reviewLoadSerial
  const revision = commentRevision
  reviewHydrated.value = false
  reviewLoadError.value = ''
  comments.value = []
  saved.value = true
  if (target.type !== 'commit') {
    reviewHydrated.value = true
    return
  }
  if (!target.commit) return
  try {
    const review = await getReview()
    if (serial !== reviewLoadSerial) return
    reviewHydrated.value = true
    if (revision === commentRevision && review.type === 'commit' && review.commit === target.commit) {
      comments.value = review.comments || []
    }
  } catch (e) {
    if (serial !== reviewLoadSerial) return
    if (revision === commentRevision) {
      reviewLoadError.value = e.message
      reviewHydrated.value = false
    } else {
      reviewHydrated.value = true
    }
  }
}, { deep: true })

const canSave = computed(() => activeTab.value === 'review' && !workdirChanging.value && reviewTarget.value.type === 'commit' && Boolean(reviewTarget.value.commit) && reviewHydrated.value && !saving.value)
const canExport = computed(() => {
  if (activeTab.value !== 'review' || workdirChanging.value) return false
  if (reviewTarget.value.type === 'commit') return reviewHydrated.value && Boolean(reviewTarget.value.commit)
  if (reviewTarget.value.type === 'branch') return Boolean(reviewTarget.value.base)
  return true
})

watch(activeTab, async (tab) => {
  if (tab === 'history') {
    await loadReviews()
  } else if (tab === 'template') {
    await loadTemplate()
  }
})

async function loadReviews() {
  try {
    reviewList.value = await getReviews()
  } catch {
    reviewList.value = []
  }
}

function retryReviewLoad() {
  reviewTarget.value = { ...reviewTarget.value }
}

function onCommentsUpdate(list) {
  if (!reviewHydrated.value) return
  commentRevision++
  comments.value = list
  saved.value = false
}

async function onSwitchWorkdir(workDir) {
  if (workdirChanging.value || saving.value) return
  workdirChanging.value = true
  workdirError.value = ''
  exportSerial++
  showExport.value = false
  exportResult.value = null
  try {
    const result = await apiSwitchWorkdir(workDir)
    reviewLoadSerial++
    commentRevision++
    comments.value = []
    reviewTarget.value = { type: 'working', commit: '', base: '' }
    reviewHydrated.value = true
    reviewLoadError.value = ''
    saved.value = true
    currentWorkdir.value = result.workDir
  } catch (e) {
    workdirError.value = e.message
  } finally {
    workdirChanging.value = false
  }
}

async function createNewReview() {
  if (saving.value) return
  saving.value = true
  try {
    const res = await apiNewReview()
    reviewLoadSerial++
    commentRevision++
    comments.value = []
    reviewHydrated.value = true
    saved.value = true
    reviewLabel.value = res.reviewID ? `Review ID: ${res.reviewID}` : ''
    if (activeTab.value === 'history') {
      await loadReviews()
    }
  } catch (e) {
    alert('新建 Review 失败: ' + e.message)
  } finally {
    saving.value = false
  }
}

async function saveReview() {
  if (saving.value || !reviewHydrated.value || activeTab.value !== 'review' || reviewTarget.value.type !== 'commit' || !reviewTarget.value.commit) return
  const target = { ...reviewTarget.value }
  const revision = commentRevision
  const loadSerial = reviewLoadSerial
  const label = reviewLabel.value
  const snapshot = [...comments.value]
  saving.value = true
  try {
    await apiSaveReview(snapshot, 'commit', target.commit)
    const targetUnchanged = target.type === reviewTarget.value.type && target.commit === reviewTarget.value.commit && target.base === reviewTarget.value.base
    if (revision === commentRevision && loadSerial === reviewLoadSerial && label === reviewLabel.value && targetUnchanged) {
      saved.value = true
    }
  } catch (e) {
    alert('保存失败: ' + e.message)
  } finally {
    saving.value = false
  }
}

let exportSerial = 0

async function doExport() {
  if (!canExport.value) return
  const serial = ++exportSerial
  const { type, commit, base } = reviewTarget.value
  const snapshot = type === 'commit' ? [...comments.value] : []
  const format = exportFormat.value
  try {
    const result = await exportReview(format, type, commit, '', base, snapshot)
    if (serial === exportSerial) {
      exportResult.value = result
      showExport.value = true
    }
  } catch (e) {
    if (serial === exportSerial) alert('导出失败: ' + e.message)
  }
}

async function onSwitchHistoryReview(reviewItem) {
  if (saving.value || reviewItem.type !== 'commit' || !reviewItem.commitExists) return
  saving.value = true
  try {
    await apiSwitchReview(reviewItem.reviewID)
    reviewTarget.value = { type: 'commit', commit: reviewItem.commit || '', base: '' }
    activeTab.value = 'review'
    reviewLabel.value = `Review ID: ${reviewItem.reviewID}`
    saved.value = true
  } catch (e) {
    alert('切换 Review 失败: ' + e.message)
  } finally {
    saving.value = false
  }
}

async function onViewHistoryReview(reviewItem) {
  const serial = ++historyDetailSerial
  try {
    const detail = await getReviewDetail(reviewItem.reviewID)
    if (serial === historyDetailSerial && detail.reviewID === reviewItem.reviewID) {
      historyReviewData.value = detail
    }
  } catch (e) {
    if (serial === historyDetailSerial) alert('加载 Review 详情失败: ' + e.message)
  }
}

function closeExportResult() {
  exportSerial++
  showExport.value = false
}

function closeReviewResultModal() {
  historyDetailSerial++
  historyReviewData.value = null
}

async function onExportHistoryReview(format) {
  if (!historyReviewData.value) return
  const serial = ++exportSerial
  const d = historyReviewData.value
  try {
    const result = await exportReview(format, d.type, d.commit || '', d.reviewID)
    if (serial === exportSerial) {
      exportResult.value = result
      showExport.value = true
    }
  } catch (e) {
    if (serial === exportSerial) alert('导出失败: ' + e.message)
  }
}

let templateApplying = false
watch(templateContent, () => {
  if (!templateApplying) {
    templateRevision++
    templateDirty.value = true
  }
}, { flush: 'sync' })

async function loadTemplate() {
  const serial = ++templateLoadSerial
  templatePendingLoads++
  templateLoading.value = true
  templateError.value = ''
  try {
    const res = await getTemplate()
    if (serial === templateLoadSerial && !templateDirty.value) {
      templateApplying = true
      templateContent.value = res.content || ''
      templateApplying = false
    }
  } catch (e) {
    if (serial === templateLoadSerial) templateError.value = e.message
  } finally {
    templatePendingLoads--
    if (templatePendingLoads === 0) templateLoading.value = false
  }
}

async function onSaveTemplate() {
  if (templateSaving.value) return
  const snapshot = templateContent.value
  const revision = templateRevision
  templateSaving.value = true
  try {
    await apiSaveTemplate(snapshot)
    if (revision === templateRevision && snapshot === templateContent.value) {
      templateLoadSerial++
      templateDirty.value = false
      alert('模板保存成功')
    }
  } catch (e) {
    alert('模板保存失败: ' + e.message)
  } finally {
    templateSaving.value = false
  }
}

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  return d.toLocaleString()
}

onMounted(async () => {
  const initialTarget = { ...reviewTarget.value }
  const initialSerial = reviewLoadSerial
  const initialRevision = commentRevision
  let initialReviewApplied = false
  function initializationIsCurrent() {
    if (initialReviewApplied) return true
    return initialSerial === reviewLoadSerial && initialRevision === commentRevision && initialTarget.type === reviewTarget.value.type && initialTarget.commit === reviewTarget.value.commit && initialTarget.base === reviewTarget.value.base
  }
  try {
    const review = await getReview()
    if (initializationIsCurrent() && review.type === 'commit' && review.commit) {
      reviewTarget.value = { type: 'commit', commit: review.commit, base: '' }
      initialReviewApplied = true
    }
  } catch {
    if (initializationIsCurrent()) reviewTarget.value = { type: 'working', commit: '', base: '' }
  }
  await nextTick()
  const initialConfigLabel = reviewLabel.value
  const configSerial = reviewLoadSerial
  const configRevision = commentRevision
  function configIsCurrent() {
    return configSerial === reviewLoadSerial && configRevision === commentRevision
  }
  try {
    const cfg = await getConfig()
    if (!configIsCurrent() || initialConfigLabel !== reviewLabel.value) return
    currentWorkdir.value = cfg.workDir || ''
    if (cfg.reviewID) {
      reviewLabel.value = `Review ID: ${cfg.reviewID}`
    } else if (cfg.reviewFile) {
      reviewLabel.value = cfg.reviewFile === 'review.json' ? 'Review: default' : `Review: ${cfg.reviewFile}`
    }
  } catch {
    if (configIsCurrent() && initialConfigLabel === reviewLabel.value) reviewLabel.value = ''
  }
  await loadReviews()
})
</script>
