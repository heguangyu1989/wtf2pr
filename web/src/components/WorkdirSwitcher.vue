<template>
  <div class="flex min-w-0 items-center text-[13px]">
    <button
      type="button"
      class="flex min-w-0 items-center gap-1.5 rounded px-1.5 py-1 text-left hover:bg-[var(--raised)]"
      :aria-label="`切换工作目录，当前 ${workDir}`"
      :title="workDir"
      @click="open"
    >
      <span class="shrink-0 text-[var(--muted)]">工作目录</span>
      <span class="max-w-44 truncate font-mono text-[var(--text)] 2xl:max-w-64">{{ workDir || '未设置' }}</span>
      <span class="shrink-0 text-[var(--accent)]">切换</span>
    </button>

    <div v-if="dialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4" @click.self="close" @keydown.esc="close">
      <form
        class="w-full max-w-xl rounded-lg bg-[var(--surface)] p-4 shadow-xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="workdir-dialog-title"
        :aria-busy="changing"
        @submit.prevent="submit"
      >
        <div class="mb-3 flex items-center justify-between gap-3">
          <h2 id="workdir-dialog-title" class="text-sm font-medium">切换工作目录</h2>
          <button type="button" class="min-h-7 rounded px-2 text-[13px] text-[var(--muted)] hover:bg-[var(--raised)]" :disabled="changing" @click="close">关闭</button>
        </div>
        <label for="workdir-input" class="mb-1.5 block text-[13px] font-medium text-[var(--muted)]">Git 仓库绝对路径</label>
        <input
          id="workdir-input"
          ref="workdirInput"
          v-model="draft"
          type="text"
          name="workDir"
          class="min-h-9 w-full rounded-md border border-[var(--control)] bg-[var(--surface)] px-2.5 font-mono text-[13px] text-[var(--text)]"
          :disabled="changing"
          :aria-invalid="Boolean(error)"
          :aria-describedby="error ? 'workdir-error' : undefined"
          autocomplete="off"
          autocorrect="off"
          autocapitalize="off"
          spellcheck="false"
          placeholder="输入 Git 仓库绝对路径"
          required
          @input="dirty = true"
        />
        <p v-if="error" id="workdir-error" class="mt-2 text-[13px] text-[var(--negative)]" role="alert">{{ error }}</p>
        <p v-if="changing" class="sr-only" role="status">正在切换工作目录…</p>
        <div class="mt-4 flex justify-end gap-2">
          <button type="button" class="min-h-8 rounded-md border border-[var(--control)] px-3 text-[13px] hover:bg-[var(--raised)] disabled:opacity-50" :disabled="changing" @click="close">取消</button>
          <button
            type="submit"
            class="control-primary min-h-8 shrink-0 rounded-md px-3 text-[13px] disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="changing || !draft.trim()"
          >{{ changing ? '切换中…' : '切换工作目录' }}</button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { nextTick, onMounted, ref, watch } from 'vue'

const props = defineProps({
  workDir: { type: String, default: '' },
  changing: { type: Boolean, default: false },
  error: { type: String, default: '' },
})
const emit = defineEmits(['switch'])
const draft = ref(props.workDir)
const dirty = ref(false)
const dialogOpen = ref(props.changing || Boolean(props.error))
const workdirInput = ref(null)

watch(() => props.workDir, value => {
  if (!dirty.value) draft.value = value
})
watch(() => props.changing, (changing, wasChanging) => {
  if (changing) dialogOpen.value = true
  if (wasChanging && !changing && !props.error) {
    draft.value = props.workDir
    dirty.value = false
    dialogOpen.value = false
  }
})
watch(() => props.error, error => {
  if (error) dialogOpen.value = true
})
onMounted(() => {
  if (dialogOpen.value) workdirInput.value?.focus?.()
})

async function open() {
  if (props.changing) return
  dialogOpen.value = true
  await nextTick()
  workdirInput.value?.focus?.()
}

function close() {
  if (props.changing) return
  dialogOpen.value = false
  draft.value = props.workDir
  dirty.value = false
}

function submit() {
  if (props.changing || !draft.value.trim()) return
  emit('switch', draft.value.trim())
}
</script>
