import test from 'node:test'
import assert from 'node:assert/strict'
import { createSSRApp } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createServer } from 'vite'

async function withVite(run) {
  const vite = await createServer({ server: { middlewareMode: true }, appType: 'custom', logLevel: 'silent' })
  try {
    await run(vite)
  } finally {
    await vite.close()
  }
}

test('source selector exposes all three modes and an honest empty branch state', async () => {
  await withVite(async vite => {
    const { default: DiffSourceSelector } = await vite.ssrLoadModule('/src/components/DiffSourceSelector.vue')
    const html = await renderToString(createSSRApp(DiffSourceSelector, {
      target: { type: 'branch', commit: '', base: '' },
      branches: { current: 'feature', default: '', list: [{ name: 'feature', current: true }] },
    }))
    assert.match(html, /未提交/)
    assert.match(html, /Commit/)
    assert.match(html, /分支对比/)
    assert.match(html, /aria-pressed="true"/)
    assert.match(html, /没有可对比的本地分支/)
  })
})

test('review workspace and app use the unified review navigation', async () => {
  await withVite(async vite => {
    const { default: ReviewWorkspace } = await vite.ssrLoadModule('/src/components/ReviewWorkspace.vue')
    const workspaceHtml = await renderToString(createSSRApp(ReviewWorkspace, {
      target: { type: 'working', commit: '', base: '' },
      comments: [],
    }))
    assert.match(workspaceHtml, /未提交/)
    assert.match(workspaceHtml, /分支对比/)
    assert.match(workspaceHtml, /<h1[^>]*>代码审查<\/h1>/)
    assert.match(workspaceHtml, /本地 Git 审查工作区/)

    const { default: App } = await vite.ssrLoadModule('/src/App.vue')
    const appHtml = await renderToString(createSSRApp(App))
    assert.match(appHtml, /代码审查/)
    assert.match(appHtml, /历史 Review/)
    assert.match(appHtml, /导出模板/)
    assert.doesNotMatch(appHtml, /Working tree/)
  })
})
