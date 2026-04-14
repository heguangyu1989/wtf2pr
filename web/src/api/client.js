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

export function getDiff(type = 'working', commit = '') {
  const qs = new URLSearchParams()
  qs.set('type', type)
  if (commit) qs.set('commit', commit)
  return request(`/diff?${qs.toString()}`)
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

export function exportReview(format, type, commit, reviewID = '') {
  return request('/export', {
    method: 'POST',
    body: JSON.stringify({ format, type, commit, reviewID }),
  })
}
