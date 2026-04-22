const LIMIT = 100

async function openRoom(room) {
  if (chatMode === 'room' && currentRoom?.id === room.id) return

  await apiPost(`/api/rooms/${room.id}/join`, {})

  chatMode    = 'room'
  currentRoom = room
  currentDM   = null
  msgOffset   = 0
  typingUsers = {}
  unreadCounts[room.id] = 0

  highlightSidebar('room', room.id)
  showChatArea()

  document.getElementById('chat-icon').textContent      = '#'
  document.getElementById('chat-room-name').textContent = room.name
  document.getElementById('chat-room-desc').textContent = room.description || ''

  const isOwner = room.owner_id === currentUser.id
  setVisible('btn-members',     true)
  setVisible('btn-invite',      isOwner)
  setVisible('btn-leave',       !isOwner)
  setVisible('btn-delete-room', isOwner)

  document.getElementById('online-badge').style.display = 'none'
  document.getElementById('members-panel').classList.remove('open')
  document.getElementById('messages').innerHTML = ''
  document.getElementById('typing-bar').textContent = ''
  hideBanner()

  const readData = await apiGet(`/api/rooms/${room.id}/read`)
  lastReadIds[room.id] = readData?.last_read_message_id || ''

  await loadMessages(false)
  connectWS(room.id)
  loadSidebar()
}

async function loadMessages(prepend) {
  if (chatMode !== 'room' || !currentRoom) return
  const data = await apiGet(`/api/rooms/${currentRoom.id}/messages?limit=${LIMIT}&offset=${msgOffset}`)
  if (!Array.isArray(data)) return

  const lastRead  = lastReadIds[currentRoom.id]
  const container = document.getElementById('messages')

  if (prepend) {
    const prevH = container.scrollHeight
    data.forEach(msg => prependMessageEl(msg))
    container.scrollTop = container.scrollHeight - prevH
    if (data.length < LIMIT) container.querySelector('.load-more-wrap')?.remove()
  } else {
    let firstUnreadIdx = -1
    if (lastRead) {
      const lastReadIdx = data.findIndex(m => m.id === lastRead)
      if (lastReadIdx !== -1 && lastReadIdx < data.length - 1) {
        firstUnreadIdx = lastReadIdx + 1
      }
    }

    data.forEach((msg, i) => {
      if (i === firstUnreadIdx) {
        insertUnreadDivider(container, null)
      }
      appendMessageEl(container, msg)
    })

    if (data.length === LIMIT) addLoadMoreBtn()

    const divider = container.querySelector('.unread-divider')
    if (divider) {
      setTimeout(() => divider.scrollIntoView({ block: 'start', behavior: 'smooth' }), 50)
    } else {
      scrollBottom()
    }
  }

  msgOffset += data.length
}

async function deleteMessage(messageId) {
  if (!currentRoom) return
  await apiDel(`/api/rooms/${currentRoom.id}/messages/${messageId}`)
}

function initRoomButtons() {
  document.getElementById('btn-new-room').onclick = () => {
    document.getElementById('modal-create').classList.add('open')
    setTimeout(() => document.getElementById('new-room-name').focus(), 50)
  }

  document.getElementById('create-cancel').onclick = () =>
    document.getElementById('modal-create').classList.remove('open')

  document.getElementById('modal-create').onclick = e => {
    if (e.target === e.currentTarget) document.getElementById('modal-create').classList.remove('open')
  }

  document.getElementById('new-room-name').onkeydown = e => {
    if (e.key === 'Enter') document.getElementById('create-confirm').click()
  }

  document.getElementById('create-confirm').onclick = async () => {
    const name        = document.getElementById('new-room-name').value.trim()
    const description = document.getElementById('new-room-desc').value.trim()
    const isPrivate   = document.getElementById('new-room-private').checked
    if (!name) return
    const data = await apiPost('/api/rooms', { name, description, is_private: isPrivate })
    if (!data || data.error) { alert(data?.error || 'Ошибка'); return }
    document.getElementById('modal-create').classList.remove('open')
    document.getElementById('new-room-name').value = ''
    document.getElementById('new-room-desc').value = ''
    document.getElementById('new-room-private').checked = false
    await loadSidebar()
    openRoom(data)
  }

  document.getElementById('btn-members').onclick = async () => {
    const panel = document.getElementById('members-panel')
    panel.classList.toggle('open')
    if (panel.classList.contains('open') && currentRoom) loadMembers()
  }

  document.getElementById('btn-invite').onclick = async () => {
    const data = await apiGet(`/api/rooms/${currentRoom.id}/invites`)
    renderInviteModal(Array.isArray(data) ? data : [])
    document.getElementById('modal-invite').classList.add('open')
  }

  document.getElementById('invite-new').onclick = async () => {
    const inv = await apiPost(`/api/rooms/${currentRoom.id}/invites`, {})
    if (inv?.id) {
      copyLink(`${location.origin}/invite/${inv.id}`)
      const data = await apiGet(`/api/rooms/${currentRoom.id}/invites`)
      renderInviteModal(Array.isArray(data) ? data : [])
    }
  }

  document.getElementById('invite-close').onclick = () =>
    document.getElementById('modal-invite').classList.remove('open')

  document.getElementById('btn-leave').onclick = () => {
    document.getElementById('leave-text').textContent = `Покинуть комнату "${currentRoom?.name}"?`
    document.getElementById('modal-leave').classList.add('open')
  }

  document.getElementById('leave-cancel').onclick = () =>
    document.getElementById('modal-leave').classList.remove('open')

  document.getElementById('leave-confirm').onclick = async () => {
    if (!currentRoom) return
    document.getElementById('modal-leave').classList.remove('open')
    await apiPost(`/api/rooms/${currentRoom.id}/leave`, {})
    resetChat()
    loadSidebar()
  }

  document.getElementById('btn-delete-room').onclick = () => {
    document.getElementById('delete-room-text').textContent =
      `Удалить "${currentRoom?.name}"? Все сообщения будут удалены.`
    document.getElementById('modal-delete-room').classList.add('open')
  }

  document.getElementById('del-room-cancel').onclick = () =>
    document.getElementById('modal-delete-room').classList.remove('open')

  document.getElementById('del-room-confirm').onclick = async () => {
    if (!currentRoom) return
    document.getElementById('modal-delete-room').classList.remove('open')
    await apiDel(`/api/rooms/${currentRoom.id}`)
    if (ws) { ws.onclose = null; ws.close(); ws = null }
    resetChat()
    loadSidebar()
  }
}

async function loadMembers() {
  const data = await apiGet(`/api/rooms/${currentRoom.id}/members`)
  const list = document.getElementById('members-list')
  list.innerHTML = ''
  if (!Array.isArray(data)) return
  const isOwner = currentRoom.owner_id === currentUser.id
  data.forEach(m => {
    const item = document.createElement('div')
    item.className = 'member-item'
    const isRoomOwner = m.user_id === currentRoom.owner_id
    item.innerHTML = `
      <div class="member-av">${m.username[0].toUpperCase()}</div>
      <span class="member-name ${isRoomOwner ? 'owner' : ''}">${esc(m.username)}${isRoomOwner ? ' 👑' : ''}</span>
      ${isOwner && !isRoomOwner ? `<button class="btn-kick" title="Выгнать">✕</button>` : ''}
    `
    if (isOwner && !isRoomOwner) {
      item.querySelector('.btn-kick').onclick = () => kickMember(m.user_id, m.username)
    }
    list.appendChild(item)
  })
}

async function kickMember(userId, username) {
  if (!confirm(`Выгнать ${username}?`)) return
  await apiDel(`/api/rooms/${currentRoom.id}/members/${userId}`)
  loadMembers()
}

function renderInviteModal(invites) {
  const content = document.getElementById('invite-content')
  if (!invites.length) {
    content.innerHTML = '<div style="font-size:14px;color:var(--muted);margin-bottom:12px;">Нет активных ссылок</div>'
    return
  }
  content.innerHTML = invites.map(inv => {
    const link = `${location.origin}/invite/${inv.id}`
    const used = !!inv.used_by
    return `
      <div class="invite-list-item">
        <div class="invite-id">${inv.id.slice(0, 18)}...</div>
        <span class="invite-status ${used ? 'used' : 'active'}">${used ? 'использована' : 'активна'}</span>
        ${!used ? `<button class="btn-copy" onclick="copyLink('${link}')">Копировать</button>` : ''}
      </div>
    `
  }).join('')
}

function copyLink(link) {
  navigator.clipboard.writeText(link).then(() => alert('Ссылка скопирована:\n' + link))
}