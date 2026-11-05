const BASE = '/api'

async function request(url, options = {}) {
  const res = await fetch(BASE + url, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  })
  const data = await res.json()
  if (data.code !== 0) {
    throw new Error(data.message || 'API error')
  }
  return data.data
}

export function getDiff(type = 'working', commit = '', base = '', options = {}) {
  const qs = new URLSearchParams({ type })
  if (commit) qs.set('commit', commit)
  if (base) qs.set('base', base)
  return request(`/diff?${qs.toString()}`, options)
}

export function getBranches() {
  return request('/branches')
}

export function getFileContent({ type, commit = '', base = '', path, side }, options = {}) {
  const qs = new URLSearchParams({ type, path, side })
  if (commit) qs.set('commit', commit)
  if (base) qs.set('base', base)
  return request(`/file/content?${qs.toString()}`, options)
}

export function getCommits(page = 1, pageSize = 10) {
  const qs = new URLSearchParams()
  qs.set('page', String(page))
  qs.set('page_size', String(pageSize))
  return request(`/commits?${qs.toString()}`)
}

export function getConfig() {
  return request('/config')
}

export async function switchWorkdir(workDir) {
  const data = await request('/config/workdir', {
    method: 'POST',
    body: JSON.stringify({ workDir }),
  })
  if (typeof data?.workDir !== 'string' || !data.workDir.trim()) {
    throw new Error('invalid workdir response')
  }
  return data
}

export function getReview() {
  return request('/review')
}

export function saveReview(comments, type, commit) {
  return request('/review', {
    method: 'POST',
    body: JSON.stringify({ comments, type, commit }),
  })
}

export function newReview() {
  return request('/review/new', { method: 'POST' })
}

export function getReviews() {
  return request('/reviews')
}

export function switchReview(reviewID) {
  return request('/review/switch', {
    method: 'POST',
    body: JSON.stringify({ reviewID }),
  })
}

export function getReviewDetail(id) {
  return request(`/review/detail?id=${encodeURIComponent(id)}`)
}

export function exportReview(format, type, commit, reviewID = '', base = '', comments = null) {
  return request('/export', {
    method: 'POST',
    body: JSON.stringify({ format, type, commit, reviewID, base, comments }),
  })
}

export function getTemplate() {
  return request('/template')
}

export function saveTemplate(content) {
  return request('/template', {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
}
