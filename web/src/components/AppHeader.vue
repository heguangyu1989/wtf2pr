<template>
  <header class="shrink-0 bg-[var(--ground)]">
    <div class="flex min-h-12 flex-wrap items-center gap-x-4 gap-y-2 px-4 py-2 lg:px-5">
      <div class="flex min-w-0 items-center gap-2.5">
        <svg class="h-6 w-6 text-[var(--accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M4 7h11m0 0-3-3m3 3-3 3" />
          <path d="M20 17H9m0 0 3-3m-3 3 3 3" />
          <path d="M4 12h16" opacity=".45" />
        </svg>
        <div class="leading-none font-semibold tracking-[-0.02em]">wtf2pr</div>
        <div class="hidden text-xs text-[var(--muted)] sm:block">Local code review</div>
      </div>
      <div class="flex items-center gap-2 text-xs">
        <span v-if="reviewLabel" class="max-w-44 truncate rounded bg-[var(--raised)] px-2 py-1 text-[var(--muted)]" :title="reviewLabel">{{ reviewLabel }}</span>
        <span class="rounded px-2 py-1" :class="saved ? 'bg-[var(--pill-positive-bg,#dcfae5)] text-[#247b50]' : 'bg-[#fffbeb] text-[#92400e]'" role="status">{{ saved ? '已保存' : '未保存' }}</span>
      </div>
      <div class="flex-1"></div>
      <div class="flex max-w-full flex-wrap items-center justify-end gap-2 py-0.5 text-xs sm:flex-nowrap sm:overflow-x-auto sm:whitespace-nowrap">
        <button type="button" class="min-h-8 rounded-md px-2.5 text-[var(--muted)] hover:bg-[var(--active)]" @click="$emit('showHelp')">帮助</button>
        <button type="button" class="min-h-8 rounded-md border border-[var(--control)] bg-[var(--surface)] px-3 enabled:hover:bg-[var(--raised)] disabled:cursor-not-allowed disabled:opacity-50" :disabled="saving" @click="$emit('newReview')">新建 Review</button>
        <label for="export-format" class="sr-only">导出格式</label>
        <select
          id="export-format"
          :value="exportFormat"
          class="min-h-8 rounded-md border border-[var(--control)] bg-[var(--surface)] px-2"
          @change="$emit('update:exportFormat', $event.target.value)"
        >
          <option value="markdown">Markdown</option>
          <option value="json">JSON</option>
          <option value="xml">XML</option>
          <option value="template">Template</option>
        </select>
        <button type="button" class="min-h-8 rounded-md border border-[var(--control)] bg-[var(--surface)] px-3 enabled:hover:bg-[var(--raised)] disabled:cursor-not-allowed disabled:opacity-50" :disabled="!canExport" title="导出当前对比的 Review" @click="$emit('doExport')">导出</button>
        <button type="button" class="control-primary min-h-8 rounded-md px-3" :disabled="!canSave" :title="canSave ? '保存当前 Commit Review' : saveReason" @click="$emit('saveReview')">保存 Review</button>
      </div>
    </div>
  </header>
</template>

<script setup>
defineProps({
  saved: { type: Boolean, default: true },
  exportFormat: { type: String, default: 'markdown' },
  reviewLabel: { type: String, default: '' },
  canSave: { type: Boolean, default: true },
  canExport: { type: Boolean, default: true },
  saving: { type: Boolean, default: false },
  saveReason: { type: String, default: '只有 Commit 模式支持保存 Review' },
})
defineEmits(['saveReview', 'doExport', 'newReview', 'showHelp', 'update:exportFormat'])
</script>
