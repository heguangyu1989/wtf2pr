<template>
  <div class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 backdrop-blur-sm p-4" @click.self="$emit('close')">
    <div class="w-full max-w-5xl max-h-[85vh] overflow-hidden rounded-xl bg-white dark:bg-gray-900 shadow-2xl flex flex-col border border-gray-200/60 dark:border-gray-700/60">
      <div class="flex items-center justify-between border-b px-4 py-3 shrink-0">
        <div>
          <h2 class="text-base font-semibold">Review 详情</h2>
          <div class="text-xs text-gray-500 mt-0.5">
            ID: {{ data.reviewID }}
            <span class="mx-1">|</span>
            类型: {{ data.type === 'commit' ? 'Commit' : 'Working' }}
            <span v-if="data.commit" class="mx-1">|</span>
            <span v-if="data.commit">Commit: {{ data.commit }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <select
            v-model="exportFormat"
            class="border rounded px-2 py-1 text-xs bg-white dark:bg-gray-900"
          >
            <option value="markdown">Markdown</option>
            <option value="json">JSON</option>
            <option value="xml">XML</option>
            <option value="template">Template</option>
          </select>
          <button class="px-2 py-1 border text-xs rounded hover:bg-gray-50 dark:hover:bg-gray-800" @click="doExport">导出</button>
          <button class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" @click="$emit('close')">关闭</button>
        </div>
      </div>

      <div class="flex-1 overflow-hidden">
        <!-- Commit review with diff snapshot：复用 DiffViewer（含截断提示与文件分页） -->
        <DiffViewer
          v-if="hasDiff"
          :diff-data="data.diff"
          :comments="data.comments || []"
          :readonly="true"
          :commit-info="data.diff?.commitInfo"
        />

        <!-- No diff: just show comments list -->
        <template v-else>
          <div class="h-full overflow-y-auto p-4">
            <div v-if="!data.comments || !data.comments.length" class="text-sm text-gray-500 text-center py-12">
              没有评论数据
            </div>
            <div v-else class="space-y-3 max-w-3xl mx-auto">
              <div
                v-for="c in data.comments"
                :key="c.id"
                class="border rounded-lg p-3 bg-gray-50 dark:bg-gray-800"
              >
                <div class="text-xs text-gray-500 mb-1">
                  {{ c.filePath }}
                  <span v-if="c.lineKey" class="ml-1 text-gray-400">— {{ c.lineKey }}</span>
                </div>
                <div class="text-sm whitespace-pre-wrap">{{ c.content }}</div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import DiffViewer from './DiffViewer.vue'

const props = defineProps({
  data: { type: Object, required: true },
})
const emit = defineEmits(['close', 'export'])

const exportFormat = ref('markdown')

const hasDiff = computed(() => props.data.type === 'commit' && props.data.diff && props.data.diff.files && props.data.diff.files.length > 0)

function doExport() {
  emit('export', exportFormat.value)
}
</script>
