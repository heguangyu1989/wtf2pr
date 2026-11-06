<template>
  <header class="shrink-0 bg-[var(--ground)]">
    <div class="flex min-h-12 flex-wrap items-center gap-x-3 gap-y-1 px-4 py-1.5 lg:px-5">
      <div class="flex min-w-0 items-center gap-2">
        <svg class="h-5 w-5 text-[var(--accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M4 7h11m0 0-3-3m3 3-3 3" />
          <path d="M20 17H9m0 0 3-3m-3 3 3 3" />
          <path d="M4 12h16" opacity=".45" />
        </svg>
        <div class="leading-none font-semibold tracking-[-0.02em]">wtf2pr</div>
        <div class="hidden text-[13px] text-[var(--muted)] xl:block">Local code review</div>
      </div>

      <nav class="flex items-center gap-1" aria-label="主导航">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="min-h-7 rounded-md px-2.5 text-[13px] transition"
          :class="activeTab === tab.key ? 'bg-[var(--active)] font-medium text-[var(--text)]' : 'text-[var(--muted)] hover:text-[var(--text)]'"
          :aria-current="activeTab === tab.key ? 'page' : undefined"
          @click="$emit('selectTab', tab.key)"
        >
          {{ tab.label }}
        </button>
      </nav>

      <div class="flex items-center gap-1.5 text-[13px]">
        <span v-if="reviewLabel" class="max-w-40 truncate rounded bg-[var(--raised)] px-2 py-1 text-[var(--muted)]" :title="reviewLabel">{{ reviewLabel }}</span>
        <span class="rounded px-2 py-1" :class="saved ? 'bg-[var(--pill-positive-bg,#dcfae5)] text-[#247b50]' : 'bg-[#fffbeb] text-[#92400e]'" role="status">{{ saved ? '已保存' : '未保存' }}</span>
      </div>
      <div class="min-w-2 flex-1"></div>

      <div class="flex max-w-full flex-wrap items-center justify-end gap-1.5 text-[13px] sm:flex-nowrap sm:whitespace-nowrap">
        <button type="button" class="min-h-7 rounded-md px-2 text-[var(--muted)] hover:bg-[var(--active)]" @click="$emit('showHelp')">帮助</button>
        <button type="button" class="min-h-7 rounded-md border border-[var(--control)] bg-[var(--surface)] px-2.5 enabled:hover:bg-[var(--raised)] disabled:cursor-not-allowed disabled:opacity-50" :disabled="saving" @click="$emit('newReview')">新建 Review</button>
        <details class="relative">
          <summary class="flex min-h-7 cursor-pointer list-none items-center rounded-md border border-[var(--control)] bg-[var(--surface)] px-2.5 hover:bg-[var(--raised)]" title="导出当前对比的 Review">导出</summary>
          <div class="absolute right-2 top-[calc(100%+4px)] z-40 w-48 rounded-lg border border-[var(--border)] bg-[var(--surface)] p-3 shadow-lg">
            <label for="export-format" class="mb-1.5 block text-[var(--muted)]">导出格式</label>
            <select
              id="export-format"
              :value="exportFormat"
              class="min-h-8 w-full rounded-md border border-[var(--control)] bg-[var(--surface)] px-2"
              @change="$emit('update:exportFormat', $event.target.value)"
            >
              <option value="markdown">Markdown</option>
              <option value="json">JSON</option>
              <option value="xml">XML</option>
              <option value="template">Template</option>
            </select>
            <button type="button" class="mt-2 min-h-8 w-full rounded-md border border-[var(--control)] bg-[var(--surface)] px-3 enabled:hover:bg-[var(--raised)] disabled:cursor-not-allowed disabled:opacity-50" :disabled="!canExport" @click="$emit('doExport')">导出 Review</button>
          </div>
        </details>
        <button type="button" class="control-primary min-h-7 rounded-md px-2.5" :disabled="!canSave" :title="canSave ? '保存当前 Commit Review' : saveReason" @click="$emit('saveReview')">保存 Review</button>
      </div>
    </div>
  </header>
</template>

<script setup>
defineProps({
  activeTab: { type: String, default: 'review' },
  saved: { type: Boolean, default: true },
  exportFormat: { type: String, default: 'markdown' },
  reviewLabel: { type: String, default: '' },
  canSave: { type: Boolean, default: true },
  canExport: { type: Boolean, default: true },
  saving: { type: Boolean, default: false },
  saveReason: { type: String, default: '只有 Commit 模式支持保存 Review' },
})
defineEmits(['saveReview', 'doExport', 'newReview', 'showHelp', 'selectTab', 'update:exportFormat'])

const tabs = [
  { key: 'review', label: '代码审查' },
  { key: 'history', label: '历史 Review' },
  { key: 'template', label: '导出模板' },
]
</script>
