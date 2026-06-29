import { useEffect, useMemo, useState } from 'react'
import type { FormEvent, ReactNode } from 'react'
import { useChatStore } from '../store/chatStore'
import { useAuthStore } from '../store/authStore'
import { roomsApi, dmApi, usersApi } from '../api/rooms'
import type { Message, Room, User } from '../types'

interface SidebarProps {
  onRoomSelect: () => void
}

type ModalKind = 'create' | 'browse' | 'dm' | 'invite' | null
type ChatFilter = 'all' | 'unread' | 'dm' | 'room'

export function Sidebar({ onRoomSelect }: SidebarProps) {
  const {
    rooms,
    dms,
    activeRoomId,
    lastMessages,
    roomActivity,
    unreadCounts,
    setActiveRoom,
    addRoom,
    setRooms,
    setDMs,
    clearUnread,
  } = useChatStore()
  const { user, clearAuth } = useAuthStore()
  const [modal, setModal] = useState<ModalKind>(null)
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState<ChatFilter>('all')
  const [notificationPermission, setNotificationPermission] = useState(() => getNotificationPermission())

  const normalizedQuery = query.trim().toLowerCase()
  const unreadTotal = useMemo(
    () => Object.values(unreadCounts).reduce((sum, count) => sum + count, 0),
    [unreadCounts]
  )
  const conversations = useMemo(
    () => [...dms, ...rooms]
      .filter((room) => matchesQuery(room, normalizedQuery, lastMessages[room.id]))
      .filter((room) => matchesFilter(room, filter, unreadCounts[room.id] ?? 0))
      .sort((a, b) => conversationTime(b, lastMessages, roomActivity) - conversationTime(a, lastMessages, roomActivity)),
    [dms, rooms, normalizedQuery, filter, unreadCounts, lastMessages, roomActivity]
  )
  const hasResults = conversations.length > 0

  const handleRoomClick = (roomId: string) => {
    setActiveRoom(roomId)
    clearUnread(roomId)
    onRoomSelect()
    roomsApi.markRead(roomId).catch(() => {})
  }

  const closeModal = () => setModal(null)

  const requestNotifications = async () => {
    if (typeof Notification === 'undefined') return
    const permission = await Notification.requestPermission()
    setNotificationPermission(permission)
  }

  return (
    <aside className="relative flex h-full flex-col border-r border-slate-200 bg-white text-slate-950">
      <header className="border-b border-slate-100 px-4 py-3">
        <div className="mb-3 flex items-center gap-3">
          <Avatar name={user?.username ?? 'User'} />
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-semibold text-slate-950">{user?.username ?? 'GoChat'}</p>
            <p className="truncate text-xs text-slate-500">{user?.email ?? 'Messenger'}</p>
          </div>
          <IconButton title="Sign out" onClick={clearAuth}>
            <Icon name="logout" className="h-5 w-5" />
          </IconButton>
        </div>

        <div className="flex items-center gap-2 rounded-full bg-slate-100 px-3 py-2">
          <Icon name="search" className="h-4 w-4 shrink-0 text-slate-400" />
          <input
            aria-label="Search chats"
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search"
            className="min-w-0 flex-1 bg-transparent text-sm text-slate-900 outline-none placeholder:text-slate-400"
          />
        </div>

        <div className="mt-3 grid grid-cols-4 gap-1 rounded-full bg-slate-100 p-1">
          <FilterButton active={filter === 'all'} onClick={() => setFilter('all')}>
            All
          </FilterButton>
          <FilterButton active={filter === 'unread'} onClick={() => setFilter('unread')}>
            Unread{unreadTotal > 0 ? ` ${unreadTotal > 99 ? '99+' : unreadTotal}` : ''}
          </FilterButton>
          <FilterButton active={filter === 'dm'} onClick={() => setFilter('dm')}>
            People
          </FilterButton>
          <FilterButton active={filter === 'room'} onClick={() => setFilter('room')}>
            Rooms
          </FilterButton>
        </div>
      </header>

      <div className="flex items-center gap-1 border-b border-slate-100 px-3 py-2">
        <IconButton title="New direct message" onClick={() => setModal('dm')}>
          <Icon name="userPlus" className="h-5 w-5" />
        </IconButton>
        <IconButton title="Create room" onClick={() => setModal('create')}>
          <Icon name="plus" className="h-5 w-5" />
        </IconButton>
        <IconButton title="Browse public rooms" onClick={() => setModal('browse')}>
          <Icon name="compass" className="h-5 w-5" />
        </IconButton>
        <IconButton title="Join by invite" onClick={() => setModal('invite')}>
          <Icon name="link" className="h-5 w-5" />
        </IconButton>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto py-2">
        {!hasResults && (
          <div className="px-5 py-10 text-center text-sm text-slate-500">
            {emptyStateText(normalizedQuery, filter)}
          </div>
        )}

        {hasResults && (
          <ConversationSection title="Chats">
            {conversations.map((room) => (
              <ConversationItem
                key={room.id}
                room={room}
                lastMessage={lastMessages[room.id]}
                activityAt={roomActivity[room.id]}
                currentUserId={user?.id}
                isActive={activeRoomId === room.id}
                unread={unreadCounts[room.id] ?? 0}
                onClick={() => handleRoomClick(room.id)}
              />
            ))}
          </ConversationSection>
        )}
      </div>

      {notificationPermission === 'default' && (
        <div className="border-t border-slate-100 p-3">
          <button
            type="button"
            onClick={requestNotifications}
            className="flex w-full items-center gap-3 rounded-xl bg-slate-50 px-3 py-2.5 text-left text-sm font-medium text-slate-700 transition hover:bg-slate-100"
          >
            <Icon name="bell" className="h-5 w-5 text-[#229ed9]" />
            <span className="min-w-0 flex-1 truncate">Turn on desktop alerts</span>
          </button>
        </div>
      )}

      {modal === 'create' && (
        <CreateRoomModal onClose={closeModal} onCreate={(room) => {
          addRoom(room)
          setActiveRoom(room.id)
          onRoomSelect()
          closeModal()
        }} />
      )}

      {modal === 'browse' && (
        <BrowseRoomsModal
          onClose={closeModal}
          myRoomIds={rooms.map((room) => room.id)}
          onJoin={(room) => {
            if (!rooms.find((item) => item.id === room.id)) setRooms([room, ...rooms])
            setActiveRoom(room.id)
            onRoomSelect()
            closeModal()
          }}
        />
      )}

      {modal === 'dm' && (
        <NewDMModal
          onClose={closeModal}
          currentUserId={user?.id ?? ''}
          onOpen={(room) => {
            if (!dms.find((item) => item.id === room.id)) setDMs([room, ...dms])
            setActiveRoom(room.id)
            onRoomSelect()
            closeModal()
          }}
        />
      )}

      {modal === 'invite' && (
        <AcceptInviteModal
          onClose={closeModal}
          onJoin={(room) => {
            if (!rooms.find((item) => item.id === room.id)) setRooms([room, ...rooms])
            setActiveRoom(room.id)
            onRoomSelect()
            closeModal()
          }}
        />
      )}
    </aside>
  )
}

function ConversationSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="mb-3">
      <h3 className="px-4 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wide text-slate-400">
        {title}
      </h3>
      <div className="space-y-0.5 px-2">{children}</div>
    </section>
  )
}

function FilterButton({ active, onClick, children }: {
  active: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={`h-8 truncate rounded-full px-2 text-xs font-semibold transition ${
        active ? 'bg-white text-[#229ed9] shadow-sm' : 'text-slate-500 hover:text-slate-900'
      }`}
    >
      {children}
    </button>
  )
}

function ConversationItem({ room, lastMessage, activityAt, currentUserId, isActive, unread, onClick }: {
  room: Room
  lastMessage?: Message
  activityAt?: string
  currentUserId?: string
  isActive: boolean
  unread: number
  onClick: () => void
}) {
  const title = room.is_dm ? room.name || 'Direct message' : room.name
  const subtitle = lastMessagePreview(room, lastMessage, currentUserId)
  const activityLabel = formatActivity(lastMessage?.created_at ?? activityAt ?? room.last_message_at ?? room.created_at)

  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left transition ${
        isActive ? 'bg-[#229ed9] text-white' : 'text-slate-950 hover:bg-slate-100'
      }`}
    >
      <Avatar name={title} isDM={room.is_dm} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="truncate text-sm font-semibold">{room.is_dm ? title : `#${title}`}</p>
          {room.is_private && !room.is_dm && (
            <Icon name="lock" className={`h-3.5 w-3.5 shrink-0 ${isActive ? 'text-white/70' : 'text-slate-400'}`} />
          )}
        </div>
        <p className={`truncate text-xs ${isActive ? 'text-white/75' : 'text-slate-500'}`}>{subtitle}</p>
      </div>
      <div className="flex shrink-0 flex-col items-end gap-1">
        <span className={`text-[11px] ${isActive ? 'text-white/70' : 'text-slate-400'}`}>{activityLabel}</span>
      {unread > 0 && (
        <span className={`grid min-w-6 place-items-center rounded-full px-1.5 py-0.5 text-[11px] font-semibold ${
          isActive ? 'bg-white text-[#229ed9]' : 'bg-[#35b779] text-white'
        }`}>
          {unread > 99 ? '99+' : unread}
        </span>
      )}
      </div>
    </button>
  )
}

function Dialog({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div className="absolute inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4 backdrop-blur-sm">
      <div className="w-full max-w-sm rounded-2xl bg-white shadow-2xl">
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <h3 className="text-sm font-semibold text-slate-950">{title}</h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="grid h-8 w-8 place-items-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-900"
          >
            <Icon name="close" className="h-4 w-4" />
          </button>
        </div>
        <div className="p-5">{children}</div>
      </div>
    </div>
  )
}

function CreateRoomModal({ onClose, onCreate }: { onClose: () => void; onCreate: (room: Room) => void }) {
  const [name, setName] = useState('')
  const [desc, setDesc] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setLoading(true)
    setError('')

    try {
      const { data } = await roomsApi.create(name.trim(), desc.trim(), isPrivate)
      onCreate(data)
    } catch {
      setError('Could not create room')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog title="Create room" onClose={onClose}>
      <form onSubmit={handleSubmit} className="space-y-3">
        <TextField autoFocus value={name} onChange={setName} placeholder="Room name" required />
        <TextField value={desc} onChange={setDesc} placeholder="Description" />
        <label className="flex cursor-pointer items-center gap-2 rounded-xl bg-slate-50 px-3 py-2.5 text-sm text-slate-700">
          <input
            type="checkbox"
            checked={isPrivate}
            onChange={(event) => setIsPrivate(event.target.checked)}
            className="h-4 w-4 rounded border-slate-300 text-[#229ed9]"
          />
          Private room
        </label>
        {error && <p className="text-xs font-medium text-red-600">{error}</p>}
        <PrimaryButton disabled={loading || !name.trim()}>
          {loading ? 'Creating...' : 'Create'}
        </PrimaryButton>
      </form>
    </Dialog>
  )
}

function BrowseRoomsModal({ onClose, onJoin, myRoomIds }: {
  onClose: () => void
  onJoin: (room: Room) => void
  myRoomIds: string[]
}) {
  const [publicRooms, setPublicRooms] = useState<Room[]>([])
  const [loading, setLoading] = useState(true)
  const [joining, setJoining] = useState<string | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let alive = true

    roomsApi.getPublic()
      .then(({ data }) => {
        if (alive) setPublicRooms(data ?? [])
      })
      .catch(() => {
        if (alive) setError('Could not load public rooms')
      })
      .finally(() => {
        if (alive) setLoading(false)
      })

    return () => {
      alive = false
    }
  }, [])

  const handleJoin = async (room: Room) => {
    setJoining(room.id)
    setError('')

    try {
      if (!myRoomIds.includes(room.id)) await roomsApi.join(room.id)
      onJoin(room)
    } catch {
      setError('Could not join room')
    } finally {
      setJoining(null)
    }
  }

  return (
    <Dialog title="Public rooms" onClose={onClose}>
      {loading ? (
        <p className="py-4 text-center text-sm text-slate-500">Loading...</p>
      ) : error ? (
        <p className="py-4 text-center text-sm text-red-600">{error}</p>
      ) : publicRooms.length === 0 ? (
        <p className="py-4 text-center text-sm text-slate-500">No public rooms</p>
      ) : (
        <div className="max-h-72 space-y-1 overflow-y-auto">
          {publicRooms.map((room) => {
            const isMember = myRoomIds.includes(room.id)
            return (
              <button
                key={room.id}
                type="button"
                onClick={() => handleJoin(room)}
                disabled={joining === room.id}
                className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left transition hover:bg-slate-50 disabled:opacity-70"
              >
                <Avatar name={room.name} />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-semibold text-slate-950">#{room.name}</p>
                  <p className="truncate text-xs text-slate-500">{room.description || 'Public room'}</p>
                </div>
                <span className="text-xs font-semibold text-[#229ed9]">
                  {joining === room.id ? '...' : isMember ? 'Open' : 'Join'}
                </span>
              </button>
            )
          })}
        </div>
      )}
    </Dialog>
  )
}

function NewDMModal({ onClose, onOpen, currentUserId }: {
  onClose: () => void
  onOpen: (room: Room) => void
  currentUserId: string
}) {
  const [search, setSearch] = useState('')
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [opening, setOpening] = useState<string | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let alive = true

    usersApi.getAll(100)
      .then(({ data }) => {
        if (alive) setUsers(data ?? [])
      })
      .catch(() => {
        if (alive) setError('Could not load users')
      })
      .finally(() => {
        if (alive) setLoading(false)
      })

    return () => {
      alive = false
    }
  }, [])

  const filtered = users.filter((candidate) =>
    candidate.id !== currentUserId &&
    (search.trim() === '' || candidate.username.toLowerCase().includes(search.trim().toLowerCase()))
  )

  const handleOpen = async (target: User) => {
    setOpening(target.id)
    setError('')

    try {
      const { data } = await dmApi.openDM(target.id)
      onOpen({ ...data, name: target.username })
    } catch {
      setError('Could not open conversation')
    } finally {
      setOpening(null)
    }
  }

  return (
    <Dialog title="New message" onClose={onClose}>
      <div className="space-y-3">
        <TextField autoFocus value={search} onChange={setSearch} placeholder="Search people" />
        {loading ? (
          <p className="py-4 text-center text-sm text-slate-500">Loading...</p>
        ) : error ? (
          <p className="py-4 text-center text-sm text-red-600">{error}</p>
        ) : filtered.length === 0 ? (
          <p className="py-4 text-center text-sm text-slate-500">No users found</p>
        ) : (
          <div className="max-h-64 space-y-1 overflow-y-auto">
            {filtered.map((candidate) => (
              <button
                key={candidate.id}
                type="button"
                onClick={() => handleOpen(candidate)}
                disabled={opening === candidate.id}
                className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left transition hover:bg-slate-50 disabled:opacity-70"
              >
                <Avatar name={candidate.username} isDM />
                <span className="min-w-0 flex-1 truncate text-sm font-semibold text-slate-950">
                  {candidate.username}
                </span>
                {opening === candidate.id && <span className="text-xs font-semibold text-[#229ed9]">...</span>}
              </button>
            ))}
          </div>
        )}
      </div>
    </Dialog>
  )
}

function AcceptInviteModal({ onClose, onJoin }: { onClose: () => void; onJoin: (room: Room) => void }) {
  const [token, setToken] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setLoading(true)
    setError('')

    try {
      const { data } = await roomsApi.acceptInvite(token.trim())
      onJoin(data)
    } catch {
      setError('Invalid or expired invite')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog title="Join by invite" onClose={onClose}>
      <form onSubmit={handleSubmit} className="space-y-3">
        <TextField autoFocus value={token} onChange={setToken} placeholder="Invite token" required />
        {error && <p className="text-xs font-medium text-red-600">{error}</p>}
        <PrimaryButton disabled={loading || !token.trim()}>
          {loading ? 'Joining...' : 'Join'}
        </PrimaryButton>
      </form>
    </Dialog>
  )
}

function TextField({ value, onChange, placeholder, required = false, autoFocus = false }: {
  value: string
  onChange: (value: string) => void
  placeholder: string
  required?: boolean
  autoFocus?: boolean
}) {
  return (
    <input
      autoFocus={autoFocus}
      type="text"
      value={value}
      onChange={(event) => onChange(event.target.value)}
      placeholder={placeholder}
      required={required}
      className="h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm text-slate-950 outline-none transition placeholder:text-slate-400 focus:border-[#229ed9]"
    />
  )
}

function PrimaryButton({ disabled, children }: { disabled?: boolean; children: ReactNode }) {
  return (
    <button
      type="submit"
      disabled={disabled}
      className="h-10 w-full rounded-full bg-[#229ed9] text-sm font-semibold text-white transition hover:bg-[#168ac0] disabled:cursor-not-allowed disabled:bg-slate-300"
    >
      {children}
    </button>
  )
}

function IconButton({ children, onClick, title }: { children: ReactNode; onClick: () => void; title: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      aria-label={title}
      className="grid h-10 w-10 place-items-center rounded-full text-slate-500 transition hover:bg-slate-100 hover:text-[#229ed9]"
    >
      {children}
    </button>
  )
}

function Avatar({ name, isDM = false }: { name: string; isDM?: boolean }) {
  const initial = (name.trim()[0] || '#').toUpperCase()

  return (
    <div className={`grid h-11 w-11 shrink-0 place-items-center rounded-full ${
      isDM ? 'bg-gradient-to-br from-[#35b779] to-[#229ed9]' : 'bg-gradient-to-br from-[#f59f00] to-[#e8590c]'
    } text-sm font-semibold text-white`}>
      {initial}
    </div>
  )
}

type IconName =
  | 'search'
  | 'logout'
  | 'bell'
  | 'userPlus'
  | 'plus'
  | 'compass'
  | 'link'
  | 'lock'
  | 'close'

function Icon({ name, className }: { name: IconName; className?: string }) {
  const paths: Record<IconName, ReactNode> = {
    search: <path d="m21 21-4.3-4.3M10.8 18a7.2 7.2 0 1 1 0-14.4 7.2 7.2 0 0 1 0 14.4z" />,
    logout: <path d="M10 17l5-5-5-5M15 12H3M21 19V5a2 2 0 0 0-2-2h-5M14 21h5a2 2 0 0 0 2-2" />,
    bell: <path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" />,
    userPlus: <path d="M16 21v-2a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v2M9.5 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM19 8v6M22 11h-6" />,
    plus: <path d="M12 5v14M5 12h14" />,
    compass: <path d="M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zM16 8l-2.2 5.8L8 16l2.2-5.8L16 8z" />,
    link: <path d="M10 13a5 5 0 0 0 7.1 0l2-2a5 5 0 0 0-7.1-7.1l-1.1 1.1M14 11a5 5 0 0 0-7.1 0l-2 2A5 5 0 0 0 12 20.1l1.1-1.1" />,
    lock: <path d="M7 11V8a5 5 0 0 1 10 0v3M6 11h12v10H6V11z" />,
    close: <path d="M18 6L6 18M6 6l12 12" />,
  }

  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      {paths[name]}
    </svg>
  )
}

function matchesQuery(room: Room, query: string, lastMessage?: Message) {
  if (!query) return true
  return `${room.name} ${room.description ?? ''} ${lastMessage?.content ?? ''} ${lastMessage?.username ?? ''}`
    .toLowerCase()
    .includes(query)
}

function matchesFilter(room: Room, filter: ChatFilter, unread: number) {
  switch (filter) {
    case 'unread':
      return unread > 0
    case 'dm':
      return room.is_dm
    case 'room':
      return !room.is_dm
    default:
      return true
  }
}

function emptyStateText(query: string, filter: ChatFilter) {
  if (query) return 'Nothing found'
  if (filter === 'unread') return 'No unread chats'
  if (filter === 'dm') return 'No direct messages'
  if (filter === 'room') return 'No rooms'
  return 'No chats yet'
}

function conversationTime(
  room: Room,
  lastMessages: Record<string, Message>,
  roomActivity: Record<string, string>
) {
  return new Date(lastMessages[room.id]?.created_at ?? roomActivity[room.id] ?? room.last_message_at ?? room.created_at).getTime()
}

function lastMessagePreview(room: Room, lastMessage?: Message, currentUserId?: string) {
  if (!lastMessage) {
    if (room.is_dm) return 'Direct message'
    return room.description || (room.is_private ? 'Private room' : 'Public room')
  }

  const prefix = lastMessage.user_id === currentUserId ? 'You: ' : room.is_dm ? '' : `${lastMessage.username}: `
  return `${prefix}${lastMessage.content}`
}

function formatActivity(value: string) {
  const date = new Date(value)
  const now = new Date()
  const sameDay = date.getFullYear() === now.getFullYear()
    && date.getMonth() === now.getMonth()
    && date.getDate() === now.getDate()

  if (sameDay) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  return date.toLocaleDateString([], { day: '2-digit', month: '2-digit' })
}

function getNotificationPermission() {
  if (typeof Notification === 'undefined') return 'unsupported'
  return Notification.permission
}
