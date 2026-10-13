const BASE = '/api'
const TOKEN_KEY = 'wtf2pr_token'

function getToken() {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setToken(token) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
}

// 启动时支持从 URL ?token=xxx 读取并持久化（服务端启动日志会输出带 token 的链接）
(function initTokenFromURL() {
  try {
    const url = new URL(window.location.href)
    const token = url.searchParams.get('token')
    if (token) {
      setToken(token)
      url.searchParams.delete('token')
      window.history.replaceState({}, '', url.toString())
    }
  } catch {
    // ignore
  }
})()

async function request(url, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }
  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const res = await fetch(BASE + url, {
    ...options,
    headers,
  })
  let data
  try {
    data = await res.json()
  } catch {
    const err = new Error(`请求失败 (HTTP ${res.status})`)
    err.status = res.status
    throw err
  }
  if (res.status === 401) {
    const err = new Error(data.message || '未授权，请输入访问令牌')
    err.status = 401
    throw err
  }
  if (data.code !== 0) {
    throw new Error(data.message || 'API error')
  }
  return data.data
}

export function checkAuth() {
  return request('/auth/check')
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

export function getTemplate() {
  return request('/template')
}

export function saveTemplate(content) {
  return request('/template', {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
}

export function getWorkdir() {
  return request('/workdir')
}

export function switchWorkdir(path) {
  return request('/workdir', {
    method: 'POST',
    body: JSON.stringify({ path }),
  })
}

export function browseDirs(path = '') {
  const qs = new URLSearchParams()
  if (path) qs.set('path', path)
  return request(`/fs/browse?${qs.toString()}`)
}
