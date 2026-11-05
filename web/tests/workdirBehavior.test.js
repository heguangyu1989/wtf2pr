import test from 'node:test'
import assert from 'node:assert/strict'
import { createClientRenderer, deferred, apiOk } from './clientRenderer.js'

async function appScenario(t, review) {
  const h = await createClientRenderer(t)
  const { default: App } = await h.runner.import('/src/App.vue')
  const persistedReview = {
    reviewID: 'test-review',
    commitExists: true,
    createdAt: 1,
    updatedAt: 1,
    diff: null,
    ...review,
  }
  const initialReview = deferred()
  const config = deferred()
  const switching = deferred()
  const exporting = deferred()
  const calls = []
  let reviewCalls = 0
  globalThis.fetch = async url => {
    calls.push(url)
    if (url === '/api/review') return ++reviewCalls === 1 ? initialReview.promise : apiOk(persistedReview)
    if (url === '/api/config') return config.promise
    if (url === '/api/config/workdir') return switching.promise
    if (url === '/api/export') return exporting.promise
    if (url === '/api/reviews') return apiOk([])
    if (url === '/api/branches') return apiOk({ current: 'main', default: 'main', list: [] })
    if (url.startsWith('/api/commits')) return apiOk({ list: [], total: 0, page: 1, pageSize: 10, totalPages: 1 })
    if (url.startsWith('/api/diff')) return apiOk({ type: 'working', files: [] })
    throw new Error(`unexpected request ${url}`)
  }
  const mounted = h.mount(App)
  return { ...h, ...mounted, initialReview, config, switching, exporting, calls, review: persistedReview }
}

async function initialize(s) {
  s.initialReview.resolve(apiOk(s.review))
  await s.flush()
}

async function workspaceScenario(t, target, fetcher) {
  const h = await createClientRenderer(t)
  const { default: ReviewWorkspace } = await h.runner.import('/src/components/ReviewWorkspace.vue')
  globalThis.fetch = fetcher
  const state = h.Vue.reactive({ workDir: '/repo/a', target })
  const mounted = h.mount({
    render: () => h.Vue.h(ReviewWorkspace, {
      target: state.target,
      comments: [],
      reviewHydrated: true,
      workDir: state.workDir,
    }),
  })
  await h.flush()
  return { ...h, ...mounted, state }
}

test('initial commit and slow config cannot overwrite a successful canonical switch', async t => {
  const s = await appScenario(t, { type: 'commit', commit: 'abc123', comments: [] })
  await initialize(s)
  await s.type(s.root, '/repo/b/subdirectory')
  await s.submit(s.root)
  s.switching.resolve(apiOk({ workDir: '/canonical/b' }))
  await s.flush()
  s.config.resolve(apiOk({ workDir: '/repo/a', reviewID: 'old-review' }))
  await s.flush()
  assert.equal(s.inputOf(s.root).value, '/canonical/b')
  s.app.unmount()
})

test('normal config still applies after restoring an initial commit', async t => {
  const s = await appScenario(t, { type: 'commit', commit: 'abc123', comments: [] })
  await initialize(s)
  s.config.resolve(apiOk({ workDir: '/repo/a', reviewFile: 'review.json' }))
  await s.flush()
  assert.equal(s.inputOf(s.root).value, '/repo/a')
  assert.match(s.textOf(s.root), /Review: default/)
  s.app.unmount()
})

test('initial config preserves an unsubmitted workdir draft', async t => {
  const s = await appScenario(t, { type: 'working', commit: '', comments: [] })
  await initialize(s)
  await s.type(s.root, '/typed/b')
  s.config.resolve(apiOk({ workDir: '/repo/a', reviewFile: 'review.json' }))
  await s.flush()
  assert.equal(s.inputOf(s.root).value, '/typed/b')
  s.app.unmount()
})

test('failed switch preserves the draft, error and current repository requests', async t => {
  const s = await appScenario(t, { type: 'working', commit: '', comments: [] })
  await initialize(s)
  s.config.resolve(apiOk({ workDir: '/repo/a', reviewFile: 'review.json' }))
  await s.flush()
  const diffCalls = s.calls.filter(url => url.startsWith('/api/diff')).length
  await s.type(s.root, '/typed/b')
  await s.submit(s.root)
  s.switching.resolve({ json: async () => ({ code: 400, message: 'bad repository', data: null }) })
  await s.flush()
  assert.equal(s.inputOf(s.root).value, '/typed/b')
  assert.match(s.textOf(s.root), /bad repository/)
  assert.equal(s.calls.filter(url => url.startsWith('/api/diff')).length, diffCalls)
  s.app.unmount()
})

test('stale export cannot reopen a result after switching repository', async t => {
  const s = await appScenario(t, { type: 'working', commit: '', comments: [] })
  await initialize(s)
  s.config.resolve(apiOk({ workDir: '/repo/a', reviewFile: 'review.json' }))
  await s.flush()
  const exportButton = s.walk(s.root, node => node.type === 'button' && s.textOf(node) === '导出')[0]
  assert.ok(exportButton)
  exportButton.props.onClick()
  await s.flush()
  assert.ok(s.calls.includes('/api/export'))
  await s.type(s.root, '/repo/b')
  await s.submit(s.root)
  s.switching.resolve(apiOk({ workDir: '/canonical/b' }))
  await s.flush()
  s.exporting.resolve(apiOk({ format: 'markdown', content: 'OLD_REPO_EXPORT' }))
  await s.flush()
  assert.doesNotMatch(s.textOf(s.root), /OLD_REPO_EXPORT/)
  s.app.unmount()
})

test('branch loading clears old data and ignores stale responses', async t => {
  const oldBranch = deferred()
  const newBranch = deferred()
  let branchCalls = 0
  const s = await workspaceScenario(t, { type: 'branch', commit: '', base: '' }, async url => {
    if (url === '/api/branches') {
      branchCalls++
      if (branchCalls === 1) return apiOk({ current: 'a-main', default: 'a-main', list: [{ name: 'a-main', current: true }] })
      return branchCalls === 2 ? oldBranch.promise : newBranch.promise
    }
    if (url.startsWith('/api/commits')) return apiOk({ list: [], total: 0, page: 1, pageSize: 10, totalPages: 1 })
    if (url.startsWith('/api/diff')) return apiOk({ type: 'branch', files: [] })
    throw new Error(`unexpected request ${url}`)
  })
  assert.match(s.textOf(s.root), /a-main/)
  s.state.workDir = '/repo/a-slow'
  await s.flush()
  assert.doesNotMatch(s.textOf(s.root), /a-main/)
  s.state.workDir = '/repo/b'
  await s.flush()
  newBranch.resolve(apiOk({ current: 'b-main', default: 'b-main', list: [{ name: 'b-main', current: true }] }))
  await s.flush()
  oldBranch.resolve(apiOk({ current: 'a-slow-main', default: 'a-slow-main', list: [{ name: 'a-slow-main', current: true }] }))
  await s.flush()
  assert.match(s.textOf(s.root), /b-main/)
  assert.doesNotMatch(s.textOf(s.root), /a-main|a-slow-main/)
  s.app.unmount()
})

test('stale diff response cannot replace the new repository files', async t => {
  const oldDiff = deferred()
  const newDiff = deferred()
  let diffCalls = 0
  const s = await workspaceScenario(t, { type: 'working', commit: '', base: '' }, async url => {
    if (url === '/api/branches') return apiOk({ current: 'main', default: 'main', list: [] })
    if (url.startsWith('/api/commits')) return apiOk({ list: [], total: 0, page: 1, pageSize: 10, totalPages: 1 })
    if (url.startsWith('/api/diff')) return ++diffCalls === 1 ? oldDiff.promise : newDiff.promise
    throw new Error(`unexpected request ${url}`)
  })
  s.state.workDir = '/repo/b'
  await s.flush()
  newDiff.resolve(apiOk({ type: 'working', files: [{ oldFile: 'only-b.txt', newFile: 'only-b.txt', hunks: [] }] }))
  await s.flush()
  oldDiff.resolve(apiOk({ type: 'working', files: [{ oldFile: 'only-a.txt', newFile: 'only-a.txt', hunks: [] }] }))
  await s.flush()
  assert.match(s.textOf(s.root), /only-b.txt/)
  assert.doesNotMatch(s.textOf(s.root), /only-a.txt/)
  s.app.unmount()
})
