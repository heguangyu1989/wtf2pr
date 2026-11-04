import test from 'node:test'
import assert from 'node:assert/strict'
import { createSSRApp } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createServer } from 'vite'

test('file explorer renders tree controls and all files without pagination', async () => {
  const vite = await createServer({ server: { middlewareMode: true }, appType: 'custom', logLevel: 'silent' })
  try {
    const { default: FileExplorer } = await vite.ssrLoadModule('/src/components/FileExplorer.vue')
    const files = Array.from({ length: 25 }, (_, index) => ({
      oldFile: `src/file${index + 1}.js`,
      newFile: `src/file${index + 1}.js`,
      hunks: [],
    }))
    const html = await renderToString(createSSRApp(FileExplorer, {
      files,
      comments: [],
      selectedPath: 'src/file1.js',
    }))
    assert.match(html, /树形/)
    assert.match(html, /平铺/)
    assert.match(html, /文件排序规则/)
    assert.match(html, /反向排序/)
    assert.match(html, /aria-current="page"/)
    assert.match(html, /file25\.js/)
    assert.doesNotMatch(html, /上一页|下一页|pageSize|filePage/)
  } finally {
    await vite.close()
  }
})

test('file explorer filters files to the selected directory', async () => {
  const vite = await createServer({ server: { middlewareMode: true }, appType: 'custom', logLevel: 'silent' })
  try {
    const { default: FileExplorer } = await vite.ssrLoadModule('/src/components/FileExplorer.vue')
    const files = [
      { oldFile: 'docs/guide.md', newFile: 'docs/guide.md', hunks: [] },
      { oldFile: 'src/main.js', newFile: 'src/main.js', hunks: [] },
      { oldFile: 'src/nested/a.js', newFile: 'src/nested/a.js', hunks: [] },
      { oldFile: 'tests/main.js', newFile: 'tests/main.js', hunks: [] },
    ]
    const html = await renderToString(createSSRApp(FileExplorer, {
      files,
      comments: [],
      selectedPath: 'src/main.js',
      selectedDirectory: 'src',
    }))
    assert.match(html, /全部文件/)
    assert.match(html, /src\/nested\/a\.js/)
    assert.doesNotMatch(html, /docs\/guide\.md/)
    assert.doesNotMatch(html, /tests\/main\.js/)
  } finally {
    await vite.close()
  }
})
