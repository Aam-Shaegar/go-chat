function avatarEl(letter) {
  const d = document.createElement('div')
  d.className = 'msg-avatar'
  d.textContent = letter.toUpperCase()
  return d
}

function makeRoomMsgEl(msg) {
  const isOwn   = msg.user_id === currentUser.id;
  const isOwner = currentRoom?.owner_id === currentUser.id;
  const canDel  = isOwn || isOwner;
  const displayName = isOwn ? currentUser.username : (msg.username || getUserName(msg.user_id));
  const letter = displayName[0];

  const el = document.createElement('div');
  el.className = 'msg-group ' + (isOwn ? 'own' : 'other');
  el.dataset.msgId = msg.id;

  const av   = avatarEl(letter);
  const body = document.createElement('div');
  body.className = 'msg-body';

  if (!isOwn) {
    const name = document.createElement('div');
    name.className = 'msg-name';
    name.textContent = displayName;
    body.appendChild(name);
  }

  const bubble = document.createElement('div');
  bubble.className = 'msg-bubble';
  bubble.textContent = msg.content;

  if (canDel) {
    const actions = document.createElement('div');
    actions.className = 'msg-actions';
    const btn = document.createElement('button');
    btn.className = 'btn-del-msg';
    btn.textContent = '✕';
    btn.onclick = e => { e.stopPropagation(); deleteMessage(msg.id); };
    actions.appendChild(btn);
    bubble.appendChild(actions);
  }

  const timeEl = document.createElement('div');
  timeEl.className = 'msg-time';
  timeEl.textContent = fmt(msg.created_at);

  body.appendChild(bubble);
  body.appendChild(timeEl);

  if (isOwn) { el.appendChild(body); el.appendChild(av); }
  else       { el.appendChild(av);   el.appendChild(body); }

  return el;
}

function makeDMMsgEl(msg) {
  const isOwn = msg.from_user_id === currentUser.id;
  let otherName = currentDM?.username;
  if (!isOwn && !otherName) {
    otherName = getUserName(msg.from_user_id);
  }
  const letter = isOwn ? currentUser.username[0] : (otherName || '?')[0];

  const el = document.createElement('div');
  el.className = 'msg-group ' + (isOwn ? 'dm-own' : 'dm-other');
  el.dataset.dmId = msg.id;

  const av   = avatarEl(letter);
  const body = document.createElement('div');
  body.className = 'msg-body';

  const bubble = document.createElement('div');
  bubble.className = 'msg-bubble';
  bubble.textContent = msg.content;

  const timeEl = document.createElement('div');
  timeEl.className = 'msg-time';
  timeEl.textContent = fmt(msg.created_at);

  body.appendChild(bubble);
  body.appendChild(timeEl);

  if (isOwn) { el.appendChild(body); el.appendChild(av); }
  else       { el.appendChild(av);   el.appendChild(body); }

  return el;
}

function appendMessageEl(container, msg) { container.appendChild(makeRoomMsgEl(msg)) }

function prependMessageEl(msg) {
  const container = document.getElementById('messages')
  const ref = container.querySelector('.load-more-wrap')
  const el  = makeRoomMsgEl(msg)
  if (ref) ref.after(el)
  else container.prepend(el)
}

function appendDMEl(container, msg) { container.appendChild(makeDMMsgEl(msg)) }

function prependDMEl(msg) {
  const container = document.getElementById('messages')
  const ref = container.querySelector('.load-more-wrap')
  const el  = makeDMMsgEl(msg)
  if (ref) ref.after(el)
  else container.prepend(el)
}

function insertUnreadDivider(container, before) {
  const div = document.createElement('div')
  div.className = 'unread-divider'
  div.innerHTML = '<div class="unread-divider-line"></div><div class="unread-divider-label">непрочитанные</div><div class="unread-divider-line"></div>'
  if (before) container.insertBefore(div, before)
  else container.appendChild(div)
}

function renderSystem(text) {
  const el = document.createElement('div')
  el.className = 'system-msg'
  el.textContent = text
  document.getElementById('messages').appendChild(el)
  scrollBottom()
}

function removeMessageEl(messageId) {
  const el = document.querySelector(`[data-msg-id="${messageId}"]`)
  if (!el) return
  const bubble = el.querySelector('.msg-bubble')
  if (bubble) bubble.innerHTML = '<span class="msg-deleted">сообщение удалено</span>'
}

function addLoadMoreBtn() {
  const container = document.getElementById('messages')
  if (container.querySelector('.load-more-wrap')) return
  const wrap = document.createElement('div')
  wrap.className = 'load-more-wrap'
  wrap.innerHTML = '<button class="btn-load-more">Загрузить ещё</button>'
  wrap.querySelector('button').onclick = () => chatMode === 'room' ? loadMessages(true) : loadMoreDM()
  container.prepend(wrap)
}

function scrollBottom() {
  const el = document.getElementById('messages');
  el.scrollTop = el.scrollHeight;
  hideBanner();
}

function isAtBottom() {
  const el = document.getElementById('messages')
  return el.scrollHeight - el.scrollTop - el.clientHeight < 60
}

function initBanner() {
  const banner = document.getElementById('new-msgs-banner')
  banner.onclick = () => { scrollBottom(); banner.classList.remove('visible') }
}

function showBanner() { document.getElementById('new-msgs-banner').classList.add('visible') }
function hideBanner() { document.getElementById('new-msgs-banner').classList.remove('visible') }

function initScrollHandler() {
  document.getElementById('messages').onscroll = () => {
    if (!isAtBottom()) return;
    hideBanner();
    if (chatMode !== 'room' || !currentRoom) return;
    const el   = document.getElementById('messages');
    const last = el.querySelector('[data-msg-id]:last-of-type');
    if (!last) return;
    const msgId = last.dataset.msgId;
    apiPost(`/api/rooms/${currentRoom.id}/read`, { message_id: msgId }).then(() => {
      lastReadIds[currentRoom.id] = msgId;
      unreadCounts[currentRoom.id] = 0;
      updateSidebarBadge(currentRoom.id);
    });
  };
}

function getUserName(userId) {
  const user = allUsers.find(u => u.id === userId);
  return user ? user.username : userId.slice(0, 8);
}