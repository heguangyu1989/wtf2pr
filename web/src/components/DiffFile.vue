<template>
  <div class="border border-gray-200 dark:border-gray-800 rounded-xl overflow-hidden bg-white dark:bg-gray-900 shadow-sm">
    <div class="px-4 py-2.5 bg-gray-50 dark:bg-gray-800/60 border-b dark:border-gray-800 text-sm font-medium flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span v-if="file.isNew" class="w-5 h-5 rounded text-xs flex items-center justify-center bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300 font-bold">+</span>
        <span v-else-if="file.isDeleted" class="w-5 h-5 rounded text-xs flex items-center justify-center bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300 font-bold">−</span>
        <span v-else class="w-5 h-5 rounded text-xs flex items-center justify-center bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300 font-bold">M</span>
        <span class="font-mono text-xs">{{ displayPath }}</span>
      </div>
      <button
        class="text-xs px-2.5 py-1 rounded-lg bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-600 transition"
        @click="addFileComment"
      >
        添加文件评论
      </button>
    </div>

    <!-- Existing comments for this file -->
    <div v-if="fileComments.length" class="border-b dark:border-gray-800 px-4 py-3 space-y-2 bg-amber-50/60 dark:bg-amber-900/10">
      <div v-for="c in fileComments" :key="c.id" class="text-sm bg-white/60 dark:bg-gray-900/40 rounded-lg px-3 py-2 border border-amber-200/50 dark:border-amber-900/30">
        <div class="text-gray-500 text-xs mb-1 flex items-center justify-between">
          <span class="font-mono">{{ c.lineKey ? `Line ${c.lineKey}` : 'File comment' }}</span>
          <button class="text-red-500 hover:text-red-600 hover:underline" @click="removeComment(c.id)">删除</button>
        </div>
        <div class="whitespace-pre-wrap">{{ c.content }}</div>
      </div>
    </div>

    <!-- File-level editing box -->
    <div v-if="editing && editingLineKey === ''" class="border-b dark:border-gray-800 px-4 py-3 bg-blue-50/40 dark:bg-blue-900/10">
      <div class="text-xs text-gray-500 mb-2">Review file</div>
      <textarea
        v-model="editingContent"
        rows="3"
        class="w-full border border-gray-300 dark:border-gray-700 rounded-lg px-2.5 py-1.5 text-sm font-mono bg-white dark:bg-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
        placeholder="输入 review 内容..."
      />
      <div class="mt-2 flex gap-2">
        <button class="px-3 py-1.5 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 transition shadow-sm" @click="submitComment">保存</button>
        <button class="px-3 py-1.5 border border-gray-300 dark:border-gray-700 text-sm rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition" @click="cancelEdit">取消</button>
      </div>
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
                <td class="pl-2 font-mono whitespace-pre cursor-pointer" @click="onLineClick(line)">
                  <span class="inline-block w-3 select-none mr-1">{{ linePrefix(line.type) }}</span>
                  <span>{{ line.content }}</span>
                </td>
              </tr>
              <!-- Inline editing row for this line -->
              <tr v-if="editing && editingLineKey === lineKey(line)" class="bg-blue-50/40 dark:bg-blue-900/10">
                <td colspan="3" class="px-4 py-3 border-b dark:border-gray-800">
                  <div class="text-xs text-gray-500 mb-2">Review line {{ editingLineKey }}</div>
                  <textarea
                    v-model="editingContent"
                    rows="3"
                    class="w-full border border-gray-300 dark:border-gray-700 rounded-lg px-2.5 py-1.5 text-sm font-mono bg-white dark:bg-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
                    placeholder="输入 review 内容..."
                  />
                  <div class="mt-2 flex gap-2">
                    <button class="px-3 py-1.5 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 transition shadow-sm" @click="submitComment">保存</button>
                    <button class="px-3 py-1.5 border border-gray-300 dark:border-gray-700 text-sm rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition" @click="cancelEdit">取消</button>
                  </div>
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
import { computed, ref } from 'vue'

const props = defineProps({
  file: { type: Object, required: true },
  comments: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:comments'])

const displayPath = computed(() => props.file.newFile || props.file.oldFile || 'unknown')

const fileComments = computed(() => props.comments.filter(c => c.filePath === displayPath.value))

function lineClass(type) {
  if (type === 'addition') return 'bg-green-50 dark:bg-green-900/10'
  if (type === 'deletion') return 'bg-red-50 dark:bg-red-900/10'
  return ''
}
function linePrefix(type) {
  if (type === 'addition') return '+'
  if (type === 'deletion') return '−'
  return ' '
}
function lineKey(line) {
  if (line.type === 'deletion' && line.oldLineNo > 0) {
    return `old:${line.oldLineNo}`
  }
  if (line.newLineNo > 0) {
    return `new:${line.newLineNo}`
  }
  return ''
}

const editing = ref(false)
const editingLineKey = ref('')
const editingContent = ref('')

function addFileComment() {
  editing.value = true
  editingLineKey.value = ''
  editingContent.value = ''
}

function onLineClick(line) {
  const key = lineKey(line)
  if (!key) return
  editing.value = true
  editingLineKey.value = key
  editingContent.value = ''
}

function submitComment() {
  if (!editingContent.value.trim()) return
  const list = [...props.comments]
  list.push({
    id: `${Date.now()}_${Math.random()}`,
    filePath: displayPath.value,
    lineKey: editingLineKey.value,
    content: editingContent.value.trim(),
    createdAt: Date.now(),
  })
  emit('update:comments', list)
  editing.value = false
  editingContent.value = ''
  editingLineKey.value = ''
}

function cancelEdit() {
  editing.value = false
  editingContent.value = ''
  editingLineKey.value = ''
}

function removeComment(id) {
  emit('update:comments', props.comments.filter(c => c.id !== id))
}
</script>
