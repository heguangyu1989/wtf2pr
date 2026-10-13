<template>
  <div class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-[70] px-4" @click.self="$emit('close')">
    <div class="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-full max-w-3xl max-h-[80vh] flex flex-col border border-gray-200/60 dark:border-gray-700/60">
      <div class="px-4 py-3 border-b dark:border-gray-700 flex items-center justify-between">
        <div class="font-semibold text-sm">导出结果 <span class="text-xs text-gray-400 font-normal">({{ format }})</span></div>
        <button class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 text-xl leading-none px-1" @click="$emit('close')">×</button>
      </div>
      <div class="p-4 overflow-auto flex-1">
        <pre class="text-xs font-mono whitespace-pre-wrap bg-gray-50 dark:bg-gray-950 border border-gray-200 dark:border-gray-800 p-3 rounded-lg">{{ content }}</pre>
      </div>
      <div class="px-4 py-3 border-t dark:border-gray-700 flex justify-end gap-2">
        <button class="px-3 py-1.5 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 transition shadow-sm" @click="copy">
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
