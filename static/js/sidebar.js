async function loadSidebar() {
  const [myRooms, pubRooms, convs, roomUnread, dmUnread] = await Promise.all([
    apiGet('/api/rooms/my'),
    apiGet('/api/rooms'),
    apiGet('/api/dm'),
    apiGet('/api/reads/unread'),
    apiGet('/api/dm/unread')
  ])

  if (Array.isArray(roomUnread)) {
    roomUnread.forEach(({ room_id, unread }) => {
      unreadCounts[room_id] = Math.max(unreadCounts[room_id] || 0, unread)
    })
  }

  if (Array.isArray(dmUnread)) {
    dmUnread.forEach(({ from_user_id, unread }) => {
      const key = 'dm_' + from_user_id
      unreadCounts[key] = Math.max(unreadCounts[key] || 0, unread)
    })
  }

  const myIds = new Set((myRooms || []).map(r => r.id))
  renderRoomSection('sec-my', myRooms || [])
  renderRoomSection('sec-pub', (pubRooms || []).filter(r => !myIds.has(r.id)))
  renderDMSection(convs || [])
}

function renderRoomSection(containerId, rooms) {
  const el = document.getElementById(containerId)
  el.innerHTML = ''

  if (!rooms.length) {
    const empty = document.createElement('div')
    empty.style.cssText = 'padding:4px 12px;font-size:12px;color:var(--muted)'
    empty.textContent = 'Нет комнат'
    el.appendChild(empty)
    return
  }

  rooms.forEach(room => {
    const item = document.createElement('div')
    item.className = 'room-item' + (currentRoom?.id === room.id ? ' active' : '')
    item.dataset.id = room.id
    const unread = unreadCounts[room.id] || 0
    item.innerHTML = `
      <span class="room-hash">#</span>
      <div class="room-info">
        <div class="room-item-name">${esc(room.name)}</div>
        ${room.description ? `<div class="room-item-desc">${esc(room.description)}</div>` : ''}
      </div>
      ${unread ? `<span class="unread-badge">${formatBadgeCount(unread)}</span>` : ''}
    `
    item.onclick = () => openRoom(room)
    el.appendChild(item)
  })
}

function renderDMSection(convs) {
  const el = document.getElementById('sec-dm')
  el.querySelectorAll('.dm-item').forEach(e => e.remove())

  convs.forEach(msg => {
    const otherId   = msg.from_user_id === currentUser.id ? msg.to_user_id : msg.from_user_id
    const otherName = allUsers.find(u => u.id === otherId)?.username || otherId.slice(0, 8)
    const item = document.createElement('div')
    item.className = 'dm-item' + (currentDM?.userId === otherId ? ' active' : '')
    item.dataset.userId = otherId
    const unread = unreadCounts['dm_' + otherId] || 0
    item.innerHTML = `
      <div class="dm-avatar">${otherName[0].toUpperCase()}</div>
      <span class="dm-name">${esc(otherName)}</span>
      ${unread ? `<span class="unread-badge">${formatBadgeCount(unread)}</span>` : ''}
    `
    item.onclick = () => openDM(otherId, otherName)
    el.appendChild(item)
  })
}

function formatBadgeCount(count){
    return count > 99 ? '99+' : count;
}

function updateSidebarBadge(roomId) {
  const item = document.querySelector(`.room-item[data-id="${roomId}"]`);
  if (!item) return;
  let badge = item.querySelector('.unread-badge');
  const count = unreadCounts[roomId] || 0;
  if (count === 0) {
    badge?.remove();
    return;
  }
  if (!badge) {
    badge = document.createElement('span');
    badge.className = 'unread-badge';
    item.appendChild(badge);
  }
  badge.textContent = formatBadgeCount(count);
}

function updateDMBadge(userId) {
  const item = document.querySelector(`.dm-item[data-user-id="${userId}"]`);
  if (!item) return;
  let badge = item.querySelector('.unread-badge');
  const count = unreadCounts['dm_' + userId] || 0;
  if (count === 0) {
    badge?.remove();
    return;
  }
  if (!badge) {
    badge = document.createElement('span');
    badge.className = 'unread-badge';
    item.appendChild(badge);
  }
  badge.textContent = formatBadgeCount(count);
}

function highlightSidebar(type, id) {
  document.querySelectorAll('.room-item, .dm-item').forEach(el => {
    if (type === 'room') el.classList.toggle('active', el.dataset.id === id)
    else el.classList.toggle('active', el.dataset.userId === id)
  })
}

function initSections() {
  function toggle(hdrId, bodyId, chevId) {
    document.getElementById(hdrId).onclick = () => {
      const body = document.getElementById(bodyId)
      const chev = document.getElementById(chevId)
      const open = !body.classList.contains('hidden')
      body.classList.toggle('hidden', open)
      chev.classList.toggle('open', !open)
    }
  }
  toggle('sec-my-hdr',  'sec-my',  'chev-my')
  toggle('sec-pub-hdr', 'sec-pub', 'chev-pub')
  toggle('sec-dm-hdr',  'sec-dm',  'chev-dm')
}