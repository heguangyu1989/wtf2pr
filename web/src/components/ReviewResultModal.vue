<template>
  <div class="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4">
    <div class="w-full max-w-5xl max-h-[85vh] overflow-hidden rounded-lg bg-white dark:bg-gray-900 shadow-xl flex flex-col">
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
        <button class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" @click="$emit('close')">关闭</button>
      </div>

      <div class="flex-1 overflow-hidden">
        <!-- Commit review with diff snapshot -->
        <template v-if="hasDiff">
          <div class="grid grid-cols-1 lg:grid-cols-4 gap-0 h-full">
            <aside class="lg:col-span-1 border-r h-full overflow-hidden bg-gray-50 dark:bg-gray-950">
              <div class="px-3 py-2 border-b bg-gray-100 dark:bg-gray-800 text-sm font-medium shrink-0">
                文件 ({{ data.diff.files.length }})
              </div>
              <ul class="overflow-y-auto h-full pb-4">
                <li
                  v-for="(f, i) in data.diff.files"
                  :key="i"
                  class="px-3 py-2 text-sm border-b last:border-b-0 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800"
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
            </aside>

            <section class="lg:col-span-3 h-full overflow-y-auto p-4 space-y-4">
              <div v-if="commitInfo" class="border rounded-lg p-3 bg-gray-50 dark:bg-gray-800 text-sm space-y-1 shrink-0">
                <div><span class="text-gray-500">Commit:</span> {{ commitInfo.hash }}</div>
                <div><span class="text-gray-500">Author:</span> {{ commitInfo.author }}</div>
                <div><span class="text-gray-500">Date:</span> {{ commitInfo.date }}</div>
                <div><span class="text-gray-500">Message:</span> {{ commitInfo.message }}</div>
              </div>

              <ReadonlyDiffFile
                v-if="selectedFile"
                :file="selectedFile"
                :comments="data.comments"
              />
            </section>
          </div>
        </template>

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
import ReadonlyDiffFile from './ReadonlyDiffFile.vue'

const props = defineProps({
  data: { type: Object, required: true },
})
defineEmits(['close'])

const hasDiff = computed(() => props.data.type === 'commit' && props.data.diff && props.data.diff.files && props.data.diff.files.length > 0)
const commitInfo = computed(() => props.data.diff?.commitInfo || null)
const selectedIndex = ref(0)
const selectedFile = computed(() => props.data.diff?.files[selectedIndex.value] || null)
</script>
