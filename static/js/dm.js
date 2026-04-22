async function openDM(userId, username) {
  if (chatMode === 'dm' && currentDM?.userId === userId) return

  chatMode    = 'dm'
  currentDM   = { userId, username }
  currentRoom = null
  msgOffset   = 0
  typingUsers = {}

  highlightSidebar('dm', userId)
  showChatArea()

  document.getElementById('chat-icon').textContent      = '✉'
  document.getElementById('chat-room-name').textContent = username
  document.getElementById('chat-room-desc').textContent = 'личные сообщения'

  setVisible('btn-members',     false)
  setVisible('btn-invite',      false)
  setVisible('btn-leave',       false)
  setVisible('btn-delete-room', false)

  document.getElementById('online-badge').style.display = 'none'
  document.getElementById('members-panel').classList.remove('open')
  document.getElementById('messages').innerHTML = ''
  document.getElementById('typing-bar').textContent = ''
  hideBanner()

  if (ws) { ws.onclose = null; ws.close(); ws = null }
  setWSDot(false)

  await loadDMHistory()

  unreadCounts['dm_' + userId] = 0
  updateDMBadge(userId)
  apiPost(`/api/dm/${userId}/read`, {})
}

async function loadDMHistory() {
  const [msgs, lastReadData] = await Promise.all([
    apiGet(`/api/dm/${currentDM.userId}/messages`),
    apiGet(`/api/dm/${currentDM.userId}/read`)
  ])

  if (!Array.isArray(msgs)) return

  const container  = document.getElementById('messages')
  const lastReadAt = lastReadData?.last_read_at ? new Date(lastReadData.last_read_at) : null

  let firstUnreadIdx = -1
  if (lastReadAt) {
    firstUnreadIdx = msgs.findIndex(m =>
      m.from_user_id !== currentUser.id &&
      new Date(m.created_at) > lastReadAt
    )
  }

  msgs.forEach((msg, i) => {
    if (i === firstUnreadIdx) {
      insertUnreadDivider(container, null)
    }
    appendDMEl(container, msg)
  })

  if (msgs.length === LIMIT) addLoadMoreBtn()

  const divider = container.querySelector('.unread-divider')
  if (divider) {
    setTimeout(() => divider.scrollIntoView({ block: 'start', behavior: 'smooth' }), 50)
  } else {
    scrollBottom()
  }

  msgOffset += msgs.length
}

async function loadMoreDM() {
  const data = await apiGet(`/api/dm/${currentDM.userId}/messages?offset=${msgOffset}`)
  if (!Array.isArray(data)) return
  const container = document.getElementById('messages')
  const prevH = container.scrollHeight
  data.forEach(msg => prependDMEl(msg))
  container.scrollTop = container.scrollHeight - prevH
  if (data.length < LIMIT) container.querySelector('.load-more-wrap')?.remove()
  msgOffset += data.length
}

function initDMButtons() {
  document.getElementById('btn-new-dm').onclick = () => {
    renderDMUserList(allUsers)
    document.getElementById('modal-new-dm').classList.add('open')
    document.getElementById('dm-search').value = ''
  }

  document.getElementById('dm-cancel').onclick = () =>
    document.getElementById('modal-new-dm').classList.remove('open')

  document.getElementById('dm-search').oninput = e => {
    const q = e.target.value.toLowerCase()
    renderDMUserList(allUsers.filter(u => u.username.toLowerCase().includes(q)))
  }
}

function renderDMUserList(users) {
  const list = document.getElementById('dm-user-list')
  list.innerHTML = ''
  if (!users.length) {
    list.innerHTML = '<div style="padding:12px;font-size:13px;color:var(--muted)">Нет пользователей</div>'
    return
  }
  users.forEach(u => {
    const item = document.createElement('div')
    item.className = 'new-dm-item'
    item.innerHTML = `<div class="new-dm-av">${u.username[0].toUpperCase()}</div><span class="new-dm-name">${esc(u.username)}</span>`
    item.onclick = () => {
      document.getElementById('modal-new-dm').classList.remove('open')
      openDM(u.id, u.username)
    }
    list.appendChild(item)
  })
}