import test from 'node:test'
import assert from 'node:assert/strict'
import { getBranches, getDiff, getFileContent } from '../src/api/client.js'

test('getDiff sends branch base and abort signal at the HTTP boundary', async t => {
  const originalFetch = globalThis.fetch
  const calls = []
  globalThis.fetch = async (url, options) => {
    calls.push({ url, options })
    return { json: async () => ({ code: 0, message: 'ok', data: {} }) }
  }
  t.after(() => { globalThis.fetch = originalFetch })

  const controller = new AbortController()
  await getDiff('branch', '', 'main', { signal: controller.signal })
  assert.equal(calls[0].url, '/api/diff?type=branch&base=main')
  assert.equal(calls[0].options.signal, controller.signal)

  await getBranches()
  assert.equal(calls[1].url, '/api/branches')
})

test('getFileContent sends identity, version, comparison and abort signal', async t => {
  const originalFetch = globalThis.fetch
  const calls = []
  globalThis.fetch = async (url, options) => {
    calls.push({ url, options })
    return { json: async () => ({ code: 0, message: 'ok', data: {} }) }
  }
  t.after(() => { globalThis.fetch = originalFetch })

  const controller = new AbortController()
  await getFileContent({ type: 'branch', base: 'main', path: 'src/a file.js', side: 'old' }, { signal: controller.signal })
  assert.equal(calls[0].url, '/api/file/content?type=branch&path=src%2Fa+file.js&side=old&base=main')
  assert.equal(calls[0].options.signal, controller.signal)
})

