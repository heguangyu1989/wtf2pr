<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-100 via-gray-50 to-blue-50 dark:from-gray-950 dark:via-gray-900 dark:to-slate-900 px-4">
    <div class="w-full max-w-sm">
      <div class="bg-white/80 dark:bg-gray-900/80 backdrop-blur rounded-2xl shadow-xl border border-gray-200/60 dark:border-gray-700/60 p-8">
        <div class="flex items-center gap-3 mb-2">
          <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center text-white text-lg font-bold shadow-md">
            w2
          </div>
          <div>
            <h1 class="text-lg font-semibold text-gray-900 dark:text-gray-100">wtf2pr</h1>
            <p class="text-xs text-gray-500 dark:text-gray-400">code review made simple</p>
          </div>
        </div>

        <p class="text-sm text-gray-600 dark:text-gray-400 mt-4 mb-4">
          请输入服务启动时输出的访问令牌（Access Token）以继续。
        </p>

        <form @submit.prevent="onSubmit">
          <input
            v-model="token"
            type="password"
            autocomplete="off"
            placeholder="Access Token"
            class="w-full px-3 py-2.5 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 transition font-mono"
            autofocus
          />
          <p v-if="error" class="text-xs text-red-500 mt-2">{{ error }}</p>
          <button
            type="submit"
            :disabled="loading || !token.trim()"
            class="w-full mt-4 px-4 py-2.5 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800 disabled:opacity-50 disabled:cursor-not-allowed transition shadow-sm"
          >
            {{ loading ? '验证中…' : '进入' }}
          </button>
        </form>
      </div>
      <p class="text-center text-xs text-gray-400 dark:text-gray-500 mt-4">
        令牌可在服务启动日志中找到，也可通过启动参数 --token 指定
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { setToken, checkAuth } from '../api/client.js'

const emit = defineEmits(['authenticated'])

const token = ref('')
const error = ref('')
const loading = ref(false)

async function onSubmit() {
  const value = token.value.trim()
  if (!value) return
  loading.value = true
  error.value = ''
  try {
    setToken(value)
    await checkAuth()
    emit('authenticated')
  } catch (e) {
    setToken('')
    error.value = e.status === 401 ? '令牌无效，请检查后重试' : (e.message || '验证失败')
  } finally {
    loading.value = false
  }
}
</script>
