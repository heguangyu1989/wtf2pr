<template>
  <form class="shrink-0 rounded-lg border bg-white px-3 py-2 dark:border-gray-700 dark:bg-gray-900" :aria-busy="changing" @submit.prevent="submit">
    <div class="flex flex-wrap items-center gap-2">
      <label for="workdir-input" class="shrink-0 text-xs font-medium text-[var(--muted)]">工作目录</label>
      <input
        id="workdir-input"
        v-model="draft"
        type="text"
        name="workDir"
        class="min-h-8 min-w-0 flex-1 rounded border border-[var(--control)] bg-[var(--surface)] px-2 font-mono text-xs text-[var(--text)]"
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
      <button
        type="submit"
        class="min-h-8 shrink-0 rounded-md border border-[var(--control)] bg-[var(--surface)] px-3 text-xs enabled:hover:bg-[var(--raised)] disabled:cursor-not-allowed disabled:opacity-50"
        :disabled="changing || !draft.trim()"
      >{{ changing ? '切换中…' : '切换工作目录' }}</button>
    </div>
    <p v-if="changing" class="sr-only" role="status">正在切换工作目录…</p>
    <p v-if="error" id="workdir-error" class="mt-1.5 text-xs text-[var(--negative)]" role="alert">{{ error }}</p>
  </form>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  workDir: { type: String, default: '' },
  changing: { type: Boolean, default: false },
  error: { type: String, default: '' },
})
const emit = defineEmits(['switch'])
const draft = ref(props.workDir)
const dirty = ref(false)

watch(() => props.workDir, value => {
  if (!dirty.value) draft.value = value
})
watch(() => props.changing, (changing, wasChanging) => {
  if (wasChanging && !changing && !props.error) {
    draft.value = props.workDir
    dirty.value = false
  }
})

function submit() {
  if (props.changing || !draft.value.trim()) return
  emit('switch', draft.value.trim())
}
</script>
