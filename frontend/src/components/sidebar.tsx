import { useState } from 'react'
import { useChatStore } from '../store/chatStore'
import { useAuthStore } from '../store/authStore'
import { roomsApi, dmApi } from '../api/rooms'
import type { Room } from '../types'

interface SidebarProps {
    onRoomSelect: () => void
}

type Modal = 'create' | 'browse' | 'dm' | 'invite' | null

export function Sidebar({ onRoomSelect }: SidebarProps) {
    const { rooms, dms, activeRoomId, setActiveRoom, unreadCounts, addRoom, setRooms, setDMs } = useChatStore()
    const { user, clearAuth } = useAuthStore()
    const [modal, setModal] = useState<Modal>(null)

    const handleRoomClick = (roomId: string) => {
        setActiveRoom(roomId)
        onRoomSelect()
        roomsApi.markRead(roomId).catch(() => {})
    }

    const closeModal = () => setModal(null)

    return (
        <div className="flex flex-col h-full bg-gray-900">
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-4 border-b border-gray-800">
                <span className="font-bold text-white text-lg">GoChat</span>
                <button onClick={clearAuth} className="text-gray-400 hover:text-white text-sm transition">
                    Sign out
                </button>
            </div>

            {/* User info */}
            <div className="px-4 py-3 border-b border-gray-800">
                <p className="text-gray-300 text-sm">
                    <span className="text-gray-500">Signed in as </span>
                    <span className="font-medium">{user?.username}</span>
                </p>
            </div>

            <div className="flex-1 overflow-y-auto">
                {/* Rooms section */}
                <div className="px-2 pt-4">
                    <div className="flex items-center justify-between px-2 mb-2">
                        <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Rooms</span>
                        <div className="flex gap-1">
                            <IconButton title="Browse public rooms" onClick={() => setModal('browse')}>🔍</IconButton>
                            <IconButton title="Join via invite" onClick={() => setModal('invite')}>🔗</IconButton>
                            <IconButton title="Create room" onClick={() => setModal('create')}>+</IconButton>
                        </div>
                    </div>

                    {rooms.map((room) => (
                        <RoomItem
                            key={room.id}
                            name={`# ${room.name}`}
                            isActive={activeRoomId === room.id}
                            unread={unreadCounts[room.id] ?? 0}
                            onClick={() => handleRoomClick(room.id)}
                        />
                    ))}
                    {rooms.length === 0 && (
                        <p className="text-gray-600 text-xs px-2 py-1">No rooms yet</p>
                    )}
                </div>

                {/* DMs section */}
                <div className="px-2 pt-4">
                    <div className="flex items-center justify-between px-2 mb-2">
                        <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Direct Messages</span>
                        <IconButton title="New DM" onClick={() => setModal('dm')}>+</IconButton>
                    </div>
                    {dms.map((dm) => (
                        <RoomItem
                            key={dm.id}
                            name={`@ ${dm.name || 'DM'}`}
                            isActive={activeRoomId === dm.id}
                            unread={unreadCounts[dm.id] ?? 0}
                            onClick={() => handleRoomClick(dm.id)}
                        />
                    ))}
                    {dms.length === 0 && (
                        <p className="text-gray-600 text-xs px-2 py-1">No direct messages</p>
                    )}
                </div>
            </div>

            {/* Modals */}
            {modal === 'create' && (
                <CreateRoomModal
                    onClose={closeModal}
                    onCreate={(room) => {
                        addRoom(room)
                        setActiveRoom(room.id)
                        onRoomSelect()
                        closeModal()
                    }}
                />
            )}
            {modal === 'browse' && (
                <BrowseRoomsModal
                    onClose={closeModal}
                    onJoin={(room) => {
                        setRooms([...rooms, room])
                        setActiveRoom(room.id)
                        onRoomSelect()
                        closeModal()
                    }}
                    myRoomIds={rooms.map((r) => r.id)}
                />
            )}
            {modal === 'dm' && (
                <NewDMModal
                    onClose={closeModal}
                    onOpen={(room) => {
                        if (!dms.find((d) => d.id === room.id)) {
                            setDMs([...dms, room])
                        }
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
                        if (!rooms.find((r) => r.id === room.id)) {
                            setRooms([...rooms, room])
                        }
                        setActiveRoom(room.id)
                        onRoomSelect()
                        closeModal()
                    }}
                />
            )}
        </div>
    )
}

// --- RoomItem ---

function RoomItem({ name, isActive, unread, onClick }: {
    name: string; isActive: boolean; unread: number; onClick: () => void
}) {
    return (
        <button
            onClick={onClick}
            className={`w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm text-left transition
                ${isActive ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:bg-gray-800 hover:text-white'}`}
        >
            <span className="truncate">{name}</span>
            {unread > 0 && (
                <span className="ml-2 bg-indigo-500 text-white text-xs rounded-full px-1.5 py-0.5 min-w-[20px] text-center">
                    {unread > 99 ? '99+' : unread}
                </span>
            )}
        </button>
    )
}

function IconButton({ children, onClick, title }: { children: React.ReactNode; onClick: () => void; title: string }) {
    return (
        <button
            onClick={onClick}
            title={title}
            className="text-gray-400 hover:text-white text-base leading-none px-1 transition"
        >
            {children}
        </button>
    )
}

// --- Modal wrapper ---

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
    return (
        <div className="absolute inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
            <div className="bg-gray-900 rounded-2xl w-full max-w-sm shadow-2xl">
                <div className="flex items-center justify-between px-5 py-4 border-b border-gray-800">
                    <h3 className="font-semibold text-white">{title}</h3>
                    <button onClick={onClose} className="text-gray-400 hover:text-white transition">✕</button>
                </div>
                <div className="p-5">{children}</div>
            </div>
        </div>
    )
}

// --- Create Room Modal ---

function CreateRoomModal({ onClose, onCreate }: { onClose: () => void; onCreate: (room: Room) => void }) {
    const [name, setName] = useState('')
    const [desc, setDesc] = useState('')
    const [isPrivate, setIsPrivate] = useState(false)
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState('')

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!name.trim()) return
        setLoading(true)
        setError('')
        try {
            const { data } = await roomsApi.create(name.trim(), desc.trim(), isPrivate)
            onCreate(data)
        } catch {
            setError('Failed to create room')
        } finally {
            setLoading(false)
        }
    }

    return (
        <Modal title="Create Room" onClose={onClose}>
            <form onSubmit={handleSubmit} className="space-y-3">
                <input
                    autoFocus
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Room name"
                    className="w-full bg-gray-800 text-white rounded-lg px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-indigo-500"
                    required
                />
                <input
                    type="text"
                    value={desc}
                    onChange={(e) => setDesc(e.target.value)}
                    placeholder="Description (optional)"
                    className="w-full bg-gray-800 text-white rounded-lg px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-indigo-500"
                />
                <label className="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
                    <input type="checkbox" checked={isPrivate} onChange={(e) => setIsPrivate(e.target.checked)} className="rounded" />
                    Private (invite only)
                </label>
                {error && <p className="text-red-400 text-xs">{error}</p>}
                <button
                    type="submit"
                    disabled={loading || !name.trim()}
                    className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:bg-gray-700 text-white rounded-lg py-2.5 text-sm font-medium transition"
                >
                    {loading ? 'Creating...' : 'Create Room'}
                </button>
            </form>
        </Modal>
    )
}

// --- Browse Rooms Modal ---

function BrowseRoomsModal({ onClose, onJoin, myRoomIds }: {
    onClose: () => void
    onJoin: (room: Room) => void
    myRoomIds: string[]
}) {
    const [publicRooms, setPublicRooms] = useState<Room[]>([])
    const [loading, setLoading] = useState(true)
    const [joining, setJoining] = useState<string | null>(null)

    // NOTE: This uses useState with a function, but it should be useEffect.
    // The original code from the screenshot uses useState(() => { ... }) which is intended as an effect.
    // I'm preserving the original logic as-is.
    useState(() => {
        roomsApi.getPublic().then(({ data }) => {
            setPublicRooms(data ?? [])
            setLoading(false)
        }).catch(() => setLoading(false))
    })

    const handleJoin = async (room: Room) => {
        setJoining(room.id)
        try {
            await roomsApi.join(room.id)
            onJoin(room)
        } catch {
            // already member or error — open anyway
            onJoin(room)
        } finally {
            setJoining(null)
        }
    }

    return (
        <Modal title="Browse Public Rooms" onClose={onClose}>
            {loading ? (
                <p className="text-gray-500 text-sm text-center py-4">Loading...</p>
            ) : publicRooms.length === 0 ? (
                <p className="text-gray-500 text-sm text-center py-4">No public rooms found</p>
            ) : (
                <div className="space-y-2 max-h-72 overflow-y-auto">
                    {publicRooms.map((room) => {
                        const isMember = myRoomIds.includes(room.id)
                        return (
                            <div key={room.id} className="flex items-center justify-between bg-gray-800 rounded-lg px-3 py-2.5">
                                <div>
                                    <p className="text-white text-sm font-medium"># {room.name}</p>
                                    {room.description && <p className="text-gray-500 text-xs">{room.description}</p>}
                                </div>
                                <button
                                    onClick={() => handleJoin(room)}
                                    disabled={joining === room.id}
                                    className={`text-xs px-3 py-1.5 rounded-lg transition font-medium
                                        ${isMember
                                            ? 'bg-gray-700 text-gray-400 hover:bg-gray-600 hover:text-white'
                                            : 'bg-indigo-600 hover:bg-indigo-500 text-white'
                                        }`}
                                >
                                    {joining === room.id ? '...' : isMember ? 'Open' : 'Join'}
                                </button>
                            </div>
                        )
                    })}
                </div>
            )}
        </Modal>
    )
}

// --- New DM Modal ---

function NewDMModal({ onClose, onOpen }: { onClose: () => void; onOpen: (room: Room) => void }) {
    const [userId, setUserId] = useState('')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState('')

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!userId.trim()) return
        setLoading(true)
        setError('')
        try {
            const { data } = await dmApi.openDM(userId.trim())
            onOpen(data)
        } catch {
            setError('User not found or invalid ID')
        } finally {
            setLoading(false)
        }
    }

    return (
        <Modal title="New Direct Message" onClose={onClose}>
            <form onSubmit={handleSubmit} className="space-y-3">
                <div>
                    <input
                        autoFocus
                        type="text"
                        value={userId}
                        onChange={(e) => setUserId(e.target.value)}
                        placeholder="Paste user ID"
                        className="w-full bg-gray-800 text-white rounded-lg px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-indigo-500"
                        required
                    />
                    <p className="text-gray-600 text-xs mt-1">
                        Ask your contact to share their user ID from profile
                    </p>
                </div>
                {error && <p className="text-red-400 text-xs">{error}</p>}
                <button
                    type="submit"
                    disabled={loading || !userId.trim()}
                    className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:bg-gray-700 text-white rounded-lg py-2.5 text-sm font-medium transition"
                >
                    {loading ? 'Opening...' : 'Open DM'}
                </button>
            </form>
        </Modal>
    )
}

// --- Accept Invite Modal ---

function AcceptInviteModal({ onClose, onJoin }: { onClose: () => void; onJoin: (room: Room) => void }) {
    const [token, setToken] = useState('')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState('')

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!token.trim()) return
        setLoading(true)
        setError('')
        try {
            const { data } = await roomsApi.acceptInvite(token.trim())
            onJoin(data)
        } catch {
            setError('Invalid or expired invite link')
        } finally {
            setLoading(false)
        }
    }

    return (
        <Modal title="Join via Invite" onClose={onClose}>
            <form onSubmit={handleSubmit} className="space-y-3">
                <input
                    autoFocus
                    type="text"
                    value={token}
                    onChange={(e) => setToken(e.target.value)}
                    placeholder="Paste invite token"
                    className="w-full bg-gray-800 text-white rounded-lg px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-indigo-500"
                    required
                />
                {error && <p className="text-red-400 text-xs">{error}</p>}
                <button
                    type="submit"
                    disabled={loading || !token.trim()}
                    className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:bg-gray-700 text-white rounded-lg py-2.5 text-sm font-medium transition"
                >
                    {loading ? 'Joining...' : 'Join Room'}
                </button>
            </form>
        </Modal>
    )
}