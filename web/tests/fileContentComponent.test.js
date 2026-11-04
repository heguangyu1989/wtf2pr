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

const file = {
  oldFile: 'src/example.js',
  newFile: 'src/example.js',
  isNew: false,
  isDeleted: false,
  isBinary: false,
  hunks: [],
}
const target = { type: 'working', commit: '', base: '' }

test('full file viewer is a native dialog with old and new versions', async () => {
  await withVite(async vite => {
    const { default: FileContentModal } = await vite.ssrLoadModule('/src/components/FileContentModal.vue')
    const html = await renderToString(createSSRApp(FileContentModal, { target, file }))
    assert.match(html, /<dialog/)
    assert.match(html, /closedby="any"/)
    assert.match(html, /aria-labelledby="file-content-title"/)
    assert.doesNotMatch(html, /role="dialog"|aria-modal="true"/)
    assert.match(html, /变更前/)
    assert.match(html, /变更后/)
    assert.match(html, /src\/example\.js/)
  })
})

test('editable and readonly diff headers both expose full file viewing', async () => {
  await withVite(async vite => {
    for (const name of ['DiffFile', 'ReadonlyDiffFile']) {
      const { default: component } = await vite.ssrLoadModule(`/src/components/${name}.vue`)
      const html = await renderToString(createSSRApp(component, { file, comments: [] }))
      assert.match(html, /查看完整文件/)
    }
  })
})
