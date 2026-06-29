import { useEffect, useState } from 'react'
import { Sidebar } from '../components/sidebar'
import { ChatArea } from '../components/chatArea'
import { useChatStore } from '../store/chatStore'
import { useAuthStore } from '../store/authStore'
import { useRoomSockets } from '../hooks/useRoomSockets'
import { roomsApi, dmApi, readsApi } from '../api/rooms'
import type { Room } from '../types'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

export function ChatPage() {
  const { rooms, dms, unreadCounts, setRooms, setDMs, setUnreadCounts, activeRoomId } = useChatStore()
  const { user } = useAuthStore()
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [loadState, setLoadState] = useState<LoadState>('idle')

  useRoomSockets(rooms, dms)

  useEffect(() => {
    if (!user) return
    let alive = true

    void Promise.resolve().then(async () => {
      setLoadState('loading')

      try {
        const [myRooms, dmsRes, unread] = await Promise.all([
          roomsApi.getMy(),
          dmApi.getAll(),
          readsApi.getAllUnread(),
        ])
        if (!alive) return

        setRooms(myRooms.data ?? [])

        const dmList = dmsRes.data ?? []
        const enrichedDms = await Promise.all(
          dmList.map(async (dm) => {
            try {
              const { data: members } = await roomsApi.getMembers(dm.id)
              const other = members.find((member) => member.user_id !== user.id)
              return { ...dm, name: other?.username ?? dm.name ?? 'Direct message' } as Room
            } catch {
              return { ...dm, name: dm.name || 'Direct message' } as Room
            }
          })
        )
        if (!alive) return

        setDMs(enrichedDms)
        setUnreadCounts(unread.data ?? {})
        setLoadState('ready')
      } catch {
        if (alive) setLoadState('error')
      }
    })

    return () => {
      alive = false
    }
  }, [user, setRooms, setDMs, setUnreadCounts])

  useEffect(() => {
    const unreadTotal = Object.values(unreadCounts).reduce((sum, count) => sum + count, 0)
    document.title = unreadTotal > 0 ? `(${unreadTotal}) GoChat` : 'GoChat'
  }, [unreadCounts])

  return (
    <div className="flex h-screen overflow-hidden bg-white text-slate-950">
      <div className={`
        ${activeRoomId && !sidebarOpen ? 'hidden' : 'flex'}
        md:flex flex-col
        w-full md:w-[360px]
        flex-shrink-0
      `}>
        <Sidebar onRoomSelect={() => setSidebarOpen(false)} />
      </div>

      <div className={`
        ${activeRoomId && !sidebarOpen ? 'flex' : 'hidden'}
        md:flex flex-col flex-1 min-w-0
      `}>
        {activeRoomId ? (
          <ChatArea onBack={() => setSidebarOpen(true)} />
        ) : (
          <div className="flex flex-1 items-center justify-center bg-[#eef3f8] px-6 text-slate-500">
            <div className="text-center">
              <div className="mx-auto mb-4 grid h-16 w-16 place-items-center rounded-full bg-white text-[#229ed9] shadow-sm">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-8 w-8" aria-hidden="true">
                  <path d="M21 15a4 4 0 0 1-4 4H8l-5 3V7a4 4 0 0 1 4-4h10a4 4 0 0 1 4 4v8z" />
                </svg>
              </div>
              <p className="text-base font-semibold text-slate-700">
                {loadState === 'loading' ? 'Loading chats...' : 'Select a chat'}
              </p>
              <p className="mt-1 text-sm">
                {loadState === 'error' ? 'Could not load conversations' : 'Messages will appear here'}
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
