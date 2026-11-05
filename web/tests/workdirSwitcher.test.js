import test from 'node:test'
import assert from 'node:assert/strict'
import { createSSRApp } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createServer } from 'vite'

test('workdir switcher renders path, action and error state', async () => {
  const vite = await createServer({ server: { middlewareMode: true }, appType: 'custom', logLevel: 'silent' })
  try {
    const { default: WorkdirSwitcher } = await vite.ssrLoadModule('/src/components/WorkdirSwitcher.vue')
    const html = await renderToString(createSSRApp(WorkdirSwitcher, {
      workDir: '/repo/a',
      changing: false,
      error: '不是 Git 仓库',
    }))
    assert.match(html, /工作目录/)
    assert.match(html, /value="\/repo\/a"/)
    assert.match(html, /切换工作目录/)
    assert.match(html, /不是 Git 仓库/)
    assert.match(html, /role="alert"/)
  } finally {
    await vite.close()
  }
})

test('workdir switcher disables input and action while changing', async () => {
  const vite = await createServer({ server: { middlewareMode: true }, appType: 'custom', logLevel: 'silent' })
  try {
    const { default: WorkdirSwitcher } = await vite.ssrLoadModule('/src/components/WorkdirSwitcher.vue')
    const html = await renderToString(createSSRApp(WorkdirSwitcher, {
      workDir: '/repo/a',
      changing: true,
      error: '',
    }))
    assert.match(html, /切换中/)
    assert.match(html, /<input[^>]*\sdisabled[\s>]/)
    assert.match(html, /<button[^>]*\sdisabled[\s>]/)
    assert.match(html, /aria-busy="true"/)
    assert.match(html, /role="status"/)
  } finally {
    await vite.close()
  }
})
