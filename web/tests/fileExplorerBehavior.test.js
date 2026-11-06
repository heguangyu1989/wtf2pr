import test from 'node:test'
import assert from 'node:assert/strict'
import { createClientRenderer } from './clientRenderer.js'

const fixturePaths = [
  'main.go',
  'main_test.go',
  'README.md',
  'docs/guide.md',
  'src/main.go',
  'src/nested/util.go',
  'src/nested/util_test.go',
  'web/app.test.ts',
  'web/app.js',
]

async function fileExplorerScenario(t, { preferences = {}, failingStorage = false } = {}) {
  const h = await createClientRenderer(t)
  const originalStorage = globalThis.localStorage
  const values = new Map(Object.entries(preferences))
  globalThis.localStorage = failingStorage ? {
    getItem() { throw new Error('storage blocked') },
    setItem() { throw new Error('storage blocked') },
  } : {
    getItem(key) { return values.has(key) ? values.get(key) : null },
    setItem(key, value) { values.set(key, String(value)) },
  }
  t.after(() => {
    if (originalStorage === undefined) delete globalThis.localStorage
    else globalThis.localStorage = originalStorage
  })

  const { default: FileExplorer } = await h.runner.import('/src/components/FileExplorer.vue')
  const files = fixturePaths.map(path => ({ oldFile: path, newFile: path, hunks: [] }))
  const selectedPaths = []
  const wrapper = {
    render: () => h.Vue.h(FileExplorer, {
      files,
      comments: [],
      selectedPath: 'main.go',
      onSelect: path => selectedPaths.push(path),
    }),
  }
  const mounted = h.mount(wrapper)
  await h.flush()
  return { ...h, ...mounted, files, selectedPaths, values, wrapper }
}

function visibleFilePaths(s) {
  const paths = new Set(fixturePaths)
  return s.walk(s.root, node => node.type === 'button' && paths.has(node.props?.title))
    .map(node => node.props.title)
}

function assertVisiblePaths(s, expected) {
  assert.deepEqual(visibleFilePaths(s).sort(), [...expected].sort())
}

function buttonWithText(s, text) {
  return s.walk(s.root, node => node.type === 'button' && s.textOf(node).includes(text))[0]
}

async function click(s, node) {
  assert.ok(node, 'expected button to exist')
  const handlers = Array.isArray(node.props.onClick) ? node.props.onClick : [node.props.onClick]
  for (const handler of handlers) handler?.({ target: node, preventDefault() {} })
  await s.flush()
}

async function choosePreset(s, label, value) {
  const select = s.walk(s.root, node => node.type === 'select' && node.props?.['aria-label'] === label)[0]
  assert.ok(select, `expected ${label} select`)
  select.value = value
  const handlers = Array.isArray(select.props.onChange) ? select.props.onChange : [select.props.onChange]
  for (const handler of handlers) handler?.({ target: select })
  await s.flush()
}

test('include and exclude inputs filter files, persist and restore through clear', async t => {
  const s = await fileExplorerScenario(t)
  assert.equal(visibleFilePaths(s).length, fixturePaths.length)
  await click(s, buttonWithText(s, '包含 / 排除'))

  await s.type(s.root, '**/*_test.go, *.md', 'file-include')
  assertVisiblePaths(s, ['main_test.go', 'README.md', 'docs/guide.md', 'src/nested/util_test.go'])
  await s.type(s.root, 'docs/**, **/*_test.go', 'file-exclude')
  assertVisiblePaths(s, ['README.md'])
  assert.match(s.textOf(s.root), /1 \/ 9 个文件/)
  await s.flush()
  assert.equal(s.values.get('wtf2pr.fileInclude'), '**/*_test.go, *.md')
  assert.equal(s.values.get('wtf2pr.fileExclude'), 'docs/**, **/*_test.go')

  s.app.unmount()
  const remounted = s.mount(s.wrapper)
  s.root = remounted.root
  await s.flush()
  assertVisiblePaths(s, ['README.md'])
  assert.equal(s.inputOf(s.root, 'file-include').value, '**/*_test.go, *.md')

  await click(s, buttonWithText(s, '清空'))
  assert.equal(visibleFilePaths(s).length, fixturePaths.length)
  assert.equal(s.values.get('wtf2pr.fileInclude'), '')
  assert.equal(s.values.get('wtf2pr.fileExclude'), '')

  await s.type(s.root, '*.does-not-exist', 'file-include')
  assertVisiblePaths(s, [])
  assert.match(s.textOf(s.root), /没有匹配的文件/)
  assert.match(s.textOf(s.root), /0 \/ 9 个文件/)
  await click(s, buttonWithText(s, '清空'))
  assert.equal(visibleFilePaths(s).length, fixturePaths.length)
  s.app.unmount()
})

test('presets merge without duplicates and compose with tree and flat views', async t => {
  const s = await fileExplorerScenario(t)
  await click(s, buttonWithText(s, '包含 / 排除'))
  await choosePreset(s, '向包含规则添加内置规则', '**/*_test.go')
  await choosePreset(s, '向包含规则添加内置规则', '**/*.md')
  await choosePreset(s, '向包含规则添加内置规则', '**/*.{test,spec}.{js,jsx,ts,tsx}')
  await choosePreset(s, '向包含规则添加内置规则', '**/*_test.go')
  assert.equal(
    s.inputOf(s.root, 'file-include').value,
    '**/*_test.go, **/*.md, **/*.{test,spec}.{js,jsx,ts,tsx}',
  )
  await choosePreset(s, '向排除规则添加内置规则', '**/*.md')
  const expected = ['main_test.go', 'src/nested/util_test.go', 'web/app.test.ts']
  assertVisiblePaths(s, expected)

  await click(s, buttonWithText(s, '平铺'))
  assertVisiblePaths(s, expected)
  await click(s, buttonWithText(s, '树形'))
  assertVisiblePaths(s, expected)

  await click(s, buttonWithText(s, '清空'))
  assertVisiblePaths(s, fixturePaths)
  s.app.unmount()
})

test('folder clicks only collapse and expand without selecting a file', async t => {
  const s = await fileExplorerScenario(t)
  const folderButton = label => s.walk(s.root, node => node.type === 'button' && node.props?.['aria-label'] === label)[0]

  await click(s, folderButton('折叠目录 src'))
  assertVisiblePaths(s, fixturePaths.filter(path => !path.startsWith('src/')))
  assert.deepEqual(s.selectedPaths, [])

  await click(s, folderButton('展开目录 src'))
  assertVisiblePaths(s, fixturePaths)
  assert.deepEqual(s.selectedPaths, [])
  s.app.unmount()
})

test('storage failures do not prevent filtering', async t => {
  const s = await fileExplorerScenario(t, { failingStorage: true })
  await click(s, buttonWithText(s, '包含 / 排除'))
  await s.type(s.root, '*.md', 'file-include')
  assertVisiblePaths(s, ['README.md', 'docs/guide.md'])
  s.app.unmount()
})
