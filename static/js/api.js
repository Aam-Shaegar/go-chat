let token = localStorage.getItem('gochat_token')

async function refreshToken() {
  try {
    const res  = await fetch('/api/auth/refresh', { method: 'POST', credentials: 'include' })
    const data = await res.json()
    if (data.access_token) {
      token = data.access_token
      localStorage.setItem('gochat_token', token)
      return true
    }
  } catch {}
  return false
}

async function apiFetch(path, opts = {}) {
  opts.headers = { ...(opts.headers || {}), 'Authorization': 'Bearer ' + token }
  let res = await fetch(path, opts)
  if (res.status === 401) {
    const ok = await refreshToken()
    if (ok) {
      opts.headers['Authorization'] = 'Bearer ' + token
      res = await fetch(path, opts)
    } else {
      localStorage.clear()
      location.reload()
      return null
    }
  }
  return res
}

async function apiGet(path) {
  const res = await apiFetch(path)
  return res ? res.json() : null
}

async function apiPost(path, body) {
  const res = await apiFetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
  return res ? res.json() : null
}

async function apiDel(path) {
  const res = await apiFetch(path, { method: 'DELETE' })
  return res ? res.json() : null
}