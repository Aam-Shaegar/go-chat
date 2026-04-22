let currentUser  = JSON.parse(localStorage.getItem('gochat_user') || 'null')
let currentRoom  = null
let currentDM    = null
let chatMode     = null
let ws           = null
let typingTimer  = null
let typingUsers  = {}
let msgOffset    = 0
let unreadCounts = {}
let lastReadIds  = {}
let allUsers     = []

function esc(str) {
  if (!str) return ''
  return str.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
}

function fmt(ts) {
  return new Date(ts).toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })
}

function showChatArea() {
  document.getElementById('chat-empty').style.display = 'none'
  document.getElementById('chat-area').style.display  = 'flex'
}

function resetChat() {
  currentRoom = null
  currentDM   = null
  chatMode    = null
  document.getElementById('chat-area').style.display  = 'none'
  document.getElementById('chat-empty').style.display = 'flex'
  document.querySelectorAll('.room-item, .dm-item').forEach(el => el.classList.remove('active'))
}

function setVisible(id, visible) {
  document.getElementById(id).classList.toggle('hidden', !visible)
}

function showTyping(username) {
  typingUsers[username] = true
  renderTyping()
  clearTimeout(typingTimer)
  typingTimer = setTimeout(() => { delete typingUsers[username]; renderTyping() }, 3000)
}

function renderTyping() {
  const names = Object.keys(typingUsers)
  document.getElementById('typing-bar').textContent = names.length ? names.join(', ') + ' печатает...' : ''
}

async function showApp() {
  document.getElementById('auth-screen').style.display = 'none'
  document.getElementById('app-screen').style.display  = 'block'
  document.getElementById('user-name').textContent     = currentUser.username
  document.getElementById('user-avatar').textContent   = currentUser.username[0].toUpperCase()
  await loadAllUsers()
  await loadSidebar()
}

async function loadAllUsers() {
  const data = await apiGet('/api/users')
  allUsers = Array.isArray(data) ? data : []
}

// ── ТЕМА ──────────────────────────────────────────────────

function initTheme() {
  const saved = localStorage.getItem('gochat_theme') || 'dark'
  applyTheme(saved)
}

function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme === 'light' ? 'light' : '')
  localStorage.setItem('gochat_theme', theme)
  document.getElementById('btn-theme').textContent = theme === 'light' ? '🌙' : '☀️'
}

function toggleTheme() {
  const current = localStorage.getItem('gochat_theme') || 'dark'
  applyTheme(current === 'dark' ? 'light' : 'dark')
}

// ── SEND ──────────────────────────────────────────────────

function initSendButton() {
  document.getElementById('send-btn').onclick = sendMessage

  document.getElementById('message-input').onkeydown = e => {
    if (e.key === 'Enter' && !e.shiftKey) { sendMessage(); return }
    if (chatMode === 'room' && ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'typing' }))
    }
  }
}

async function sendMessage() {
  const input   = document.getElementById('message-input')
  const content = input.value.trim()
  if (!content) return
  input.value = ''
  input.focus()

  if (chatMode === 'room') {
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({ type: 'send_message', content }))
    setTimeout(() => scrollBottom(), 50);
  } else if (chatMode === 'dm') {
    const data = await apiPost(`/api/dm/${currentDM.userId}/messages`, { content })
    if (data && !data.error) {
      appendDMEl(document.getElementById('messages'), data)
      scrollBottom()
      msgOffset++
    }
  }
}

// ── LOGOUT ────────────────────────────────────────────────

function initLogout() {
  document.getElementById('btn-logout').onclick = () => {
    if (ws) ws.close()
    localStorage.clear()
    location.reload()
  }
}

// ── INVITE ROUTE ──────────────────────────────────────────

async function handleInviteRoute() {
  const match = location.pathname.match(/^\/invite\/([^/]+)$/)
  if (!match) return
  const inviteId = match[1]
  if (!token || !currentUser) {
    localStorage.setItem('pending_invite', inviteId)
    return
  }
  const room = await apiPost(`/api/invites/${inviteId}/accept`, {})
  if (room?.id) {
    history.replaceState(null, '', '/')
    await loadSidebar()
    openRoom(room)
  } else {
    alert('Ссылка недействительна или уже использована')
    history.replaceState(null, '', '/')
  }
}

// ── INIT ──────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', () => {
  initTheme()
  initAuth()
  initSections()
  initBanner()
  initScrollHandler()
  initRoomButtons()
  initDMButtons()
  initSendButton()
  initLogout()

  document.getElementById('btn-theme').onclick = toggleTheme

  if (token && currentUser) {
    showApp()
    handleInviteRoute()
    const pending = localStorage.getItem('pending_invite')
    if (pending) {
      localStorage.removeItem('pending_invite')
      apiPost(`/api/invites/${pending}/accept`, {}).then(room => {
        if (room?.id) loadSidebar().then(() => openRoom(room))
      })
    }
  }
})