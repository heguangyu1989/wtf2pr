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

function splitTopLevelPatterns(value) {
  const parts = ['']
  let braceDepth = 0
  let inCharacterClass = false
  for (const char of value) {
    if (char === '[' && !inCharacterClass) inCharacterClass = true
    else if (char === ']' && inCharacterClass) inCharacterClass = false
    else if (!inCharacterClass && char === '{') braceDepth++
    else if (!inCharacterClass && char === '}' && braceDepth > 0) braceDepth--

    if (char === ',' && !inCharacterClass && braceDepth === 0) parts.push('')
    else parts[parts.length - 1] += char
  }
  return parts
}

export function parseGlobPatterns(value = '') {
  if (typeof value !== 'string') return []
  return splitTopLevelPatterns(value).map(part => part.trim()).filter(Boolean)
}

function expandBracePatterns(pattern) {
  let start = -1
  let depth = 0
  let inCharacterClass = false
  for (let index = 0; index < pattern.length; index += 1) {
    const char = pattern[index]
    if (char === '[' && !inCharacterClass) inCharacterClass = true
    else if (char === ']' && inCharacterClass) inCharacterClass = false
    else if (!inCharacterClass && char === '{') {
      if (depth === 0) start = index
      depth++
    } else if (!inCharacterClass && char === '}' && depth > 0) {
      depth--
      if (depth === 0) {
        const prefix = pattern.slice(0, start)
        const suffix = pattern.slice(index + 1)
        return splitTopLevelPatterns(pattern.slice(start + 1, index)).flatMap(part =>
          expandBracePatterns(prefix + part + suffix))
      }
    }
  }
  return [pattern]
}

const regexSpecialCharacters = new Set(['\\', '^', '$', '.', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|'])

function escapeRegexCharacter(char) {
  return regexSpecialCharacters.has(char) ? `\\${char}` : char
}

function compileCharacterClass(pattern, start) {
  let index = start + 1
  let content = ''
  while (index < pattern.length && pattern[index] !== ']') {
    const char = pattern[index]
    content += char === '\\' ? '\\\\' : char === '[' ? '\\[' : char
    index++
  }
  if (index === pattern.length || !content) return null

  const negated = content[0] === '!' || content[0] === '^'
  if (negated) content = content.slice(1)
  if (!content) return null
  return { source: `(?!/)[${negated ? '^' : ''}${content}]`, index: index + 1 }
}

function compileFragment(pattern, start = 0, inBraces = false) {
  const alternatives = ['']
  let index = start
  let closed = false

  function append(source) {
    alternatives[alternatives.length - 1] += source
  }

  while (index < pattern.length) {
    const char = pattern[index]
    if (inBraces && char === '}') {
      closed = true
      index++
      break
    }
    if (inBraces && char === ',') {
      alternatives.push('')
      index++
      continue
    }
    if (char === '{') {
      const group = compileFragment(pattern, index + 1, true)
      if (group.closed) {
        append(`(?:${group.source})`)
        index = group.index
        continue
      }
    }
    if (char === '*') {
      let end = index
      while (pattern[end] === '*') end++
      const globstar = end - index > 1
      if (globstar && pattern[end] === '/') {
        append('(?:[^/]+/)*')
        index = end + 1
      } else {
        append(globstar ? '.*' : '[^/]*')
        index = end
      }
      continue
    }
    if (char === '?') {
      append('[^/]')
      index++
      continue
    }
    if (char === '[') {
      const characterClass = compileCharacterClass(pattern, index)
      if (characterClass) {
        append(characterClass.source)
        index = characterClass.index
        continue
      }
    }
    append(escapeRegexCharacter(char))
    index++
  }

  return { source: alternatives.join('|'), index, closed }
}

function compileGlob(pattern) {
  const matchers = expandBracePatterns(pattern).map(compileSingleGlob)
  return filePath => matchers.some(matches => matches(filePath))
}

function compileSingleGlob(rawPattern) {
  let pattern = rawPattern
  let rootOnly = false
  if (pattern.startsWith('./')) {
    rootOnly = true
    pattern = pattern.slice(2)
  } else if (pattern.startsWith('/')) {
    rootOnly = true
    pattern = pattern.replace(/^\/+/, '')
  }

  const matchPath = rootOnly || pattern.includes('/')
  pattern = pattern.replace(/\/+$/, '')
  if (!pattern) return () => false

  const { source } = compileFragment(pattern)
  let regex
  try {
    regex = new RegExp(`^(?:${source})$`)
  } catch {
    return () => false
  }
  if (!matchPath) {
    return filePath => filePath.split('/').some(segment => regex.test(segment))
  }

  return filePath => {
    if (regex.test(filePath)) return true
    let separator = filePath.indexOf('/')
    while (separator >= 0) {
      if (regex.test(filePath.slice(0, separator))) return true
      separator = filePath.indexOf('/', separator + 1)
    }
    return false
  }
}

export function filterFiles(files, include = '', exclude = '') {
  const includes = parseGlobPatterns(include).map(compileGlob)
  const excludes = parseGlobPatterns(exclude).map(compileGlob)
  return files.filter(file => {
    const filePath = getFilePath(file)
    if (includes.length && !includes.some(matches => matches(filePath))) return false
    return !excludes.some(matches => matches(filePath))
  })
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
