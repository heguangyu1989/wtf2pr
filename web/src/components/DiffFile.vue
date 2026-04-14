<template>
  <div class="border rounded-lg overflow-hidden bg-white dark:bg-gray-900">
    <div class="px-4 py-2 bg-gray-100 dark:bg-gray-800 border-b text-sm font-medium flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span v-if="file.isNew" class="text-green-600">+</span>
        <span v-else-if="file.isDeleted" class="text-red-600">−</span>
        <span v-else class="text-gray-500">M</span>
        <span class="font-mono">{{ displayPath }}</span>
      </div>
      <button
        class="text-xs px-2 py-1 rounded bg-white dark:bg-gray-700 border hover:bg-gray-50"
        @click="addFileComment"
      >
        添加文件评论
      </button>
    </div>

    <!-- Existing comments for this file -->
    <div v-if="fileComments.length" class="border-b px-4 py-3 space-y-2 bg-yellow-50/50 dark:bg-yellow-900/10">
      <div v-for="c in fileComments" :key="c.id" class="text-sm">
        <div class="text-gray-500 text-xs mb-1">
          {{ c.lineKey ? `Line ${c.lineKey}` : 'File comment' }}
          <button class="ml-2 text-red-500 hover:underline" @click="removeComment(c.id)">删除</button>
        </div>
        <div class="whitespace-pre-wrap">{{ c.content }}</div>
      </div>
    </div>

    <!-- File-level editing box -->
    <div v-if="editing && editingLineKey === ''" class="border-b px-4 py-3 bg-white dark:bg-gray-900">
      <div class="text-xs text-gray-500 mb-2">Review file</div>
      <textarea
        v-model="editingContent"
        rows="3"
        class="w-full border rounded px-2 py-1 text-sm font-mono"
        placeholder="输入 review 内容..."
      />
      <div class="mt-2 flex gap-2">
        <button class="px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700" @click="submitComment">保存</button>
        <button class="px-3 py-1 border text-sm rounded hover:bg-gray-50" @click="cancelEdit">取消</button>
      </div>
    </div>

    <div v-if="file.isBinary" class="p-4 text-sm text-gray-500">Binary file</div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-sm border-collapse">
        <tbody>
          <template v-for="(hunk, hi) in file.hunks" :key="hi">
            <tr class="bg-gray-50 dark:bg-gray-800">
              <td colspan="3" class="px-2 py-1 text-gray-500 font-mono text-xs select-none">
                @@ -{{ hunk.oldStart }},{{ hunk.oldLines }} +{{ hunk.newStart }},{{ hunk.newLines }} @@
              </td>
            </tr>
            <template v-for="(line, li) in hunk.lines" :key="`${hi}-${li}`">
              <tr
                class="hover:bg-gray-50 dark:hover:bg-gray-800"
                :class="lineClass(line.type)"
              >
                <td class="w-12 text-right pr-2 text-gray-400 select-none font-mono text-xs border-r">
                  {{ line.oldLineNo > 0 ? line.oldLineNo : '' }}
                </td>
                <td class="w-12 text-right pr-2 text-gray-400 select-none font-mono text-xs border-r">
                  {{ line.newLineNo > 0 ? line.newLineNo : '' }}
                </td>
                <td class="pl-2 font-mono whitespace-pre cursor-pointer" @click="onLineClick(line)">
                  <span class="inline-block w-3 select-none mr-1">{{ linePrefix(line.type) }}</span>
                  <span>{{ line.content }}</span>
                </td>
              </tr>
              <!-- Inline editing row for this line -->
              <tr v-if="editing && editingLineKey === lineKey(line)" class="bg-white dark:bg-gray-900">
                <td colspan="3" class="px-4 py-3 border-b">
                  <div class="text-xs text-gray-500 mb-2">Review line {{ editingLineKey }}</div>
                  <textarea
                    v-model="editingContent"
                    rows="3"
                    class="w-full border rounded px-2 py-1 text-sm font-mono"
                    placeholder="输入 review 内容..."
                  />
                  <div class="mt-2 flex gap-2">
                    <button class="px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700" @click="submitComment">保存</button>
                    <button class="px-3 py-1 border text-sm rounded hover:bg-gray-50" @click="cancelEdit">取消</button>
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
