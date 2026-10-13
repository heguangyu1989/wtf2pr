export function lineClass(type) {
  if (type === 'addition') return 'bg-green-50 dark:bg-green-900/10'
  if (type === 'deletion') return 'bg-red-50 dark:bg-red-900/10'
  return ''
}

export function linePrefix(type) {
  if (type === 'addition') return '+'
  if (type === 'deletion') return '−'
  return ' '
}

export function fileDisplayPath(file) {
  return file.newFile || file.oldFile || 'unknown'
}
