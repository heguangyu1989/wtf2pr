import { createServer } from 'vite'
import { ModuleRunner } from 'vite/module-runner'

export function deferred() {
  let resolve, reject
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

export function apiOk(data) {
  return { json: async () => ({ code: 0, message: 'ok', data }) }
}

export async function createClientRenderer(t) {
  const originalFetch = globalThis.fetch
  const originalDocument = globalThis.Document
  const originalShadowRoot = globalThis.ShadowRoot
  globalThis.__VUE_OPTIONS_API__ = true
  globalThis.__VUE_PROD_DEVTOOLS__ = false
  globalThis.__VUE_PROD_HYDRATION_MISMATCH_DETAILS__ = false
  globalThis.Document = class Document {}
  globalThis.ShadowRoot = class ShadowRoot {}

  const vite = await createServer({
    server: { middlewareMode: true, hmr: false },
    appType: 'custom',
    logLevel: 'silent',
    environments: { client: { dev: { moduleRunnerTransform: true } } },
  })
  const stub = id => ({
    id, url: id, file: null, invalidate: false,
    code: id === '/@vite/client'
      ? "const createHotContext = () => ({ accept(){}, dispose(){}, prune(){}, invalidate(){}, on(){}, off(){}, send(){} }); __vite_ssr_exportName__('createHotContext', () => createHotContext);"
      : '',
  })
  const runner = new ModuleRunner({
    hmr: false,
    transport: {
      invoke: async ({ data: { name, data } }) => {
        if (name === 'getBuiltins') return { result: [] }
        const [id] = data
        if (id === '/@vite/client' || id.includes('type=style') || id.endsWith('.css')) return { result: stub(id) }
        return { result: await vite.environments.client.fetchModule(...data) }
      },
    },
  })
  t.after(async () => {
    globalThis.fetch = originalFetch
    if (originalDocument === undefined) delete globalThis.Document
    else globalThis.Document = originalDocument
    if (originalShadowRoot === undefined) delete globalThis.ShadowRoot
    else globalThis.ShadowRoot = originalShadowRoot
    await runner.close()
    await vite.close()
  })

  const Vue = await runner.import('vue')
  const insert = (child, parent, anchor) => {
    if (child.parent) {
      const siblings = child.parent.children || []
      const index = siblings.indexOf(child)
      if (index >= 0) siblings.splice(index, 1)
    }
    child.parent = parent
    parent.children ||= []
    const index = anchor ? parent.children.indexOf(anchor) : -1
    if (index < 0) parent.children.push(child)
    else parent.children.splice(index, 0, child)
  }
  const makeNode = (type, text = '') => ({
    type, text, props: {}, children: [], parent: null, listeners: {}, options: [],
    addEventListener(name, fn) { (this.listeners[name] ||= []).push(fn) },
    removeEventListener(name, fn) { this.listeners[name] = (this.listeners[name] || []).filter(item => item !== fn) },
    getRootNode() { return {} },
  })
  const renderer = Vue.createRenderer({
    patchProp(el, key, prev, next) {
      if (next == null) delete el.props[key]
      else el.props[key] = next
    },
    insert,
    remove(child) {
      if (!child.parent) return
      const siblings = child.parent.children || []
      const index = siblings.indexOf(child)
      if (index >= 0) siblings.splice(index, 1)
      child.parent = null
    },
    createElement: makeNode,
    createText: text => makeNode('#text', text),
    createComment: text => makeNode('#comment', text),
    setText(node, text) { node.text = text },
    setElementText(el, text) { el.text = text; el.children = [] },
    parentNode: node => node.parent,
    nextSibling(node) {
      if (!node.parent) return null
      const siblings = node.parent.children
      return siblings[siblings.indexOf(node) + 1] || null
    },
    querySelector: () => null,
    setScopeId(el, id) { el.props[id] = '' },
    cloneNode: node => ({ ...node }),
    insertStaticContent(content, parent, anchor) {
      const node = makeNode('#static', content)
      insert(node, parent, anchor)
      return [node, node]
    },
  })

  function walk(node, predicate, found = []) {
    if (!node) return found
    if (predicate(node)) found.push(node)
    for (const child of node.children || []) walk(child, predicate, found)
    return found
  }
  function textOf(node) {
    if (!node) return ''
    return `${node.text || ''}${(node.children || []).map(textOf).join('')}`
  }
  function inputOf(root) {
    return walk(root, node => node.props?.id === 'workdir-input')[0]
  }
  function formOf(root) {
    return walk(root, node => node.type === 'form' && node.props?.onSubmit)[0]
  }
  async function flush() {
    for (let i = 0; i < 5; i++) {
      await new Promise(resolve => setTimeout(resolve, 0))
      await Vue.nextTick()
    }
  }
  async function type(root, value) {
    const input = inputOf(root)
    input.value = value
    const event = { target: input, preventDefault() {} }
    for (const listener of input.listeners.input || []) listener(event)
    const handlers = Array.isArray(input.props.onInput) ? input.props.onInput : [input.props.onInput]
    for (const handler of handlers) handler?.(event)
    await Vue.nextTick()
  }
  async function submit(root) {
    formOf(root).props.onSubmit({ preventDefault() {} })
    await Vue.nextTick()
  }
  function mount(component) {
    const root = makeNode('root')
    const app = renderer.createApp(component)
    app.mount(root)
    return { app, root }
  }

  return { Vue, runner, walk, textOf, inputOf, flush, type, submit, mount }
}
