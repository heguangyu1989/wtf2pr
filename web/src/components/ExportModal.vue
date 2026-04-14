<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 px-4">
    <div class="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-3xl max-h-[80vh] flex flex-col">
      <div class="px-4 py-3 border-b flex items-center justify-between">
        <div class="font-medium">导出结果 ({{ format }})</div>
        <button class="text-gray-500 hover:text-gray-700" @click="$emit('close')">关闭</button>
      </div>
      <div class="p-4 overflow-auto flex-1">
        <pre class="text-xs font-mono whitespace-pre-wrap bg-gray-100 dark:bg-gray-800 p-3 rounded">{{ content }}</pre>
      </div>
      <div class="px-4 py-3 border-t flex justify-end gap-2">
        <button class="px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700" @click="copy">
          {{ copied ? '已复制' : '复制到剪贴板' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  format: { type: String, default: '' },
  content: { type: String, default: '' },
})
const emit = defineEmits(['close'])

const copied = ref(false)

async function copy() {
  if (!props.content) return
  const ok = await copyToClipboard(props.content)
  if (ok) {
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  } else {
    alert('复制失败，请手动复制')
  }
}

async function copyToClipboard(text) {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // fallthrough
    }
  }
  // Fallback for non-secure contexts
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  textarea.style.top = '0'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  try {
    const successful = document.execCommand('copy')
    document.body.removeChild(textarea)
    return successful
  } catch {
    document.body.removeChild(textarea)
    return false
  }
}
</script>
