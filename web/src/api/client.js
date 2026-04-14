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

export function getReview() {
  return request('/review')
}

export function saveReview(comments) {
  return request('/review', {
    method: 'POST',
    body: JSON.stringify({ comments }),
  })
}

export function exportReview(format, type, commit) {
  return request('/export', {
    method: 'POST',
    body: JSON.stringify({ format, type, commit }),
  })
}
