<template>
  <div class="border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden bg-white dark:bg-gray-900 shadow-sm">
    <div class="px-4 py-2.5 bg-gray-50 dark:bg-gray-800/60 border-b dark:border-gray-800 text-sm font-medium flex items-center justify-between">
      <div class="flex items-center gap-2">
        <FileBadge :file="file" />
        <span class="font-mono text-xs">{{ displayPath }}</span>
      </div>
    </div>

    <!-- Existing comments for this file -->
    <div v-if="fileComments.length" class="border-b dark:border-gray-800 px-4 py-3 space-y-2 bg-amber-50/60 dark:bg-amber-900/10">
      <CommentCard v-for="c in fileComments" :key="c.id" :comment="c" />
    </div>

    <div v-if="file.isBinary" class="p-4 text-sm text-gray-500">Binary file</div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-sm border-collapse">
        <tbody>
          <template v-for="(hunk, hi) in file.hunks" :key="hi">
            <tr class="bg-gray-50 dark:bg-gray-800/70">
              <td colspan="3" class="px-2 py-1.5 text-gray-500 font-mono text-xs select-none">
                @@ -{{ hunk.oldStart }},{{ hunk.oldLines }} +{{ hunk.newStart }},{{ hunk.newLines }} @@
              </td>
            </tr>
            <template v-for="(line, li) in hunk.lines" :key="`${hi}-${li}`">
              <tr
                class="hover:bg-gray-50 dark:hover:bg-gray-800/60"
                :class="lineClass(line.type)"
              >
                <td class="w-12 text-right pr-2 text-gray-400 select-none font-mono text-xs border-r dark:border-gray-800">
                  {{ line.oldLineNo > 0 ? line.oldLineNo : '' }}
                </td>
                <td class="w-12 text-right pr-2 text-gray-400 select-none font-mono text-xs border-r dark:border-gray-800">
                  {{ line.newLineNo > 0 ? line.newLineNo : '' }}
                </td>
                <td class="pl-2 font-mono whitespace-pre">
                  <span class="inline-block w-3 select-none mr-1">{{ linePrefix(line.type) }}</span>
                  <span>{{ line.content }}</span>
                </td>
              </tr>
            </template>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import FileBadge from './FileBadge.vue'
import CommentCard from './CommentCard.vue'
import { lineClass, linePrefix, fileDisplayPath } from '../utils/diff.js'

const props = defineProps({
  file: { type: Object, required: true },
  comments: { type: Array, default: () => [] },
})

const displayPath = computed(() => fileDisplayPath(props.file))

const fileComments = computed(() => props.comments.filter(c => c.filePath === displayPath.value))
</script>
