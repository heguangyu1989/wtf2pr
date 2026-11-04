const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })
const supportedSortKeys = ['path', 'type', 'test', 'document']

function normalizeSortKey(sortKey) {
  return supportedSortKeys.includes(sortKey) ? sortKey : 'path'
}

export function getFilePath(file) {
  return file?.newFile || file?.oldFile || ''
}

export function getFileSidePath(file, side) {
  if (side === 'old') return file?.oldFile || getFilePath(file)
  return file?.newFile || getFilePath(file)
}

export function getNodeKey(node) {
  return `${node.type}:${node.path}`
}

function getFileCategory(filePath) {
  const normalized = filePath.toLowerCase()
  const name = normalized.split('/').at(-1)
  if (/(^|\/)(__tests__|tests?)(\/|$)/.test(normalized) || /\.(test|spec)\.[^.]+$/.test(name) || /_test\.go$/.test(name)) {
    return 'test'
  }
  if (/(^|\/)docs\//.test(normalized) || /\.(md|mdx|rst|adoc|asciidoc)$/.test(name) || /^(readme|changelog|license|contributing|code_of_conduct)(\.|$)/.test(name)) {
    return 'document'
  }
  return 'other'
}

function compareText(a, b) {
  const result = collator.compare(a, b)
  if (result !== 0) return result
  if (a === b) return 0
  return a < b ? -1 : 1
}

function extension(filePath) {
  const name = filePath.split('/').at(-1)
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(dot + 1).toLowerCase() : ''
}

function compareFiles(a, b, sortKey) {
  const aPath = getFilePath(a)
  const bPath = getFilePath(b)
  if (sortKey === 'test' || sortKey === 'document') {
    const aRank = getFileCategory(aPath) === sortKey ? 0 : 1
    const bRank = getFileCategory(bPath) === sortKey ? 0 : 1
    if (aRank !== bRank) return aRank - bRank
  }
  if (sortKey === 'type') {
    const byExtension = compareText(extension(aPath), extension(bPath))
    if (byExtension !== 0) return byExtension
  }
  return compareText(aPath, bPath)
}

function directionFactor(direction) {
  return direction === 'desc' ? -1 : 1
}

export function sortFiles(files, sortKey = 'path', direction = 'asc') {
  const key = normalizeSortKey(sortKey)
  const factor = directionFactor(direction)
  return [...files].sort((a, b) => factor * compareFiles(a, b, key))
}

function sortNodes(nodes, sortKey, direction) {
  const factor = directionFactor(direction)
  nodes.sort((a, b) => {
    if (a.type !== b.type) return a.type === 'directory' ? -1 : 1
    if (a.type === 'file') return factor * compareFiles(a.file, b.file, sortKey)
    return factor * compareText(a.name, b.name)
  })
  for (const node of nodes) {
    if (node.type === 'directory') sortNodes(node.children, sortKey, direction)
  }
}

export function getDefaultFilePath(files) {
  const sorted = sortFiles(files)
  const preferred = sorted.find(file => !file.isBinary && file.hunks?.some(hunk => hunk.lines?.length))
  return getFilePath(preferred || sorted[0])
}

export function buildFileTree(files, sortKey = 'path', direction = 'asc') {
  const key = normalizeSortKey(sortKey)
  const roots = []
  const directories = new Map()

  for (const file of files) {
    const filePath = getFilePath(file)
    const segments = filePath.split('/').filter(Boolean)
    if (!segments.length) continue
    let siblings = roots
    let directoryPath = ''
    for (const name of segments.slice(0, -1)) {
      directoryPath = directoryPath ? `${directoryPath}/${name}` : name
      let directory = directories.get(directoryPath)
      if (!directory) {
        directory = { name, path: directoryPath, type: 'directory', children: [] }
        directories.set(directoryPath, directory)
        siblings.push(directory)
      }
      siblings = directory.children
    }
    siblings.push({
      name: segments.at(-1),
      path: filePath,
      type: 'file',
      file,
      children: [],
    })
  }

  sortNodes(roots, key, direction)
  return roots
}
