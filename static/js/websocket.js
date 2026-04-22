function connectWS(roomID) {
  if (ws) { ws.onclose = null; ws.close() }
  setWSDot(false)
  const url = `ws://${location.host}/ws/rooms/${roomID}?token=${token}&username=${encodeURIComponent(currentUser.username)}`
  ws = new WebSocket(url)
  ws.onopen  = () => setWSDot(true)
  ws.onmessage = e => handleWS(JSON.parse(e.data))
  ws.onclose = () => {
    setWSDot(false)
    setTimeout(() => { if (chatMode === 'room' && currentRoom?.id === roomID) connectWS(roomID) }, 2000)
  }
  ws.onerror = () => setWSDot(false)
}

function handleWS(msg) {
  switch (msg.type) {
    case 'new_message': {
        const m = msg.payload.message;
        m.username = msg.payload.username;
        if (chatMode === 'room' && currentRoom?.id === m.room_id) {
            const container = document.getElementById('messages');
            appendMessageEl(container, m);
            msgOffset++;
            const shouldScroll = (m.user_id === currentUser.id) || isAtBottom();
            if (shouldScroll) {
            scrollBottom();
            } else {
            showBanner();
            }
            if (isAtBottom()) {
            apiPost(`/api/rooms/${currentRoom.id}/read`, { message_id: m.id }).then(() => {
                lastReadIds[currentRoom.id] = m.id;
                unreadCounts[currentRoom.id] = 0;
                updateSidebarBadge(currentRoom.id);
            });
            }
        } else {
            unreadCounts[m.room_id] = (unreadCounts[m.room_id] || 0) + 1;
            updateSidebarBadge(m.room_id);
        }
        break;
    }
    case 'new_dm': {
        const dm      = msg.payload;
        const otherId = dm.from_user_id === currentUser.id ? dm.to_user_id : dm.from_user_id;
        if (chatMode === 'dm' && currentDM?.userId === otherId) {
            const container = document.getElementById('messages');
            appendDMEl(container, dm);
            const shouldScroll = (dm.from_user_id === currentUser.id) || isAtBottom();
            if (shouldScroll) {
            scrollBottom();
            } else {
            showBanner();
            }
        } else {
            unreadCounts['dm_' + otherId] = (unreadCounts['dm_' + otherId] || 0) + 1;
            updateDMBadge(otherId);
        }
        break;
    }
    case 'user_typing':
      if (msg.payload.user_id !== currentUser.id) showTyping(msg.payload.username)
      break
    case 'user_joined':
      renderSystem(msg.payload.username + ' вошёл в комнату')
      break
    case 'user_left':
      renderSystem(msg.payload.username + ' покинул комнату')
      break
    case 'room_stats':
      updateOnlineCount(msg.payload.online_count)
      break
    case 'message_deleted':
      removeMessageEl(msg.payload.message_id)
      break
  }
}

function setWSDot(on) {
  document.getElementById('ws-dot').className = 'ws-dot' + (on ? ' connected' : '')
}

function updateOnlineCount(n) {
  const badge = document.getElementById('online-badge')
  document.getElementById('online-count').textContent = n + ' онлайн'
  badge.style.display = n > 0 ? 'flex' : 'none'
}