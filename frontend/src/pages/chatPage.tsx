import { useEffect, useState } from 'react'
import { Sidebar } from '../components/sidebar'
import { ChatArea } from '../components/chatArea'
import { useChatStore } from '../store/chatStore'
import { roomsApi, dmApi, readsApi } from '../api/rooms'

export function ChatPage() {
  const { setRooms, setDMs, setUnreadCounts, activeRoomId } = useChatStore()
  const [sidebarOpen, setSidebarOpen] = useState(true)

  useEffect(() => {
    const load = async () => {
      const [myRooms, dms, unread] = await Promise.all([
        roomsApi.getMy(),
        dmApi.getAll(),
        readsApi.getAllUnread(),
      ])
      setRooms(myRooms.data ?? [])
      setDMs(dms.data ?? [])
      setUnreadCounts(unread.data ?? {})
    }
    load()
  }, [setRooms, setDMs, setUnreadCounts])

  return (
    <div className="flex h-screen bg-gray-950 text-white overflow-hidden">
      {/* Sidebar */}
      <div className={`
        ${activeRoomId && !sidebarOpen ? 'hidden' : 'flex'}
        md:flex flex-col
        w-full md:w-72 lg:w-80
        flex-shrink-0
        border-r border-gray-800
      `}>
        <Sidebar onRoomSelect={() => setSidebarOpen(false)} />
      </div>

      {/* Chat area */}
      <div className={`
        ${activeRoomId && !sidebarOpen ? 'flex' : 'hidden'}
        md:flex flex-col flex-1 min-w-0
      `}>
        {activeRoomId ? (
          <ChatArea onBack={() => setSidebarOpen(true)} />
        ) : (
          <div className="flex-1 flex items-center justify-center text-gray-500">
            <div className="text-center">
              <p className="text-xl mb-2">👈 Select a room to start chatting</p>
              <p className="text-sm">Or create a new one</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}