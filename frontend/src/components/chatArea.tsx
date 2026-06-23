import { useEffect, useRef, useState, useCallback } from 'react'
import { useChatStore } from '../store/chatStore'
import { useAuthStore } from '../store/authStore'
import { useWebSocket } from '../hooks/useWebSocket'
import { roomsApi } from '../api/rooms'
import type { Message } from '../types'

interface ChatAreaProps {
  onBack: () => void
}

export function ChatArea({ onBack }: ChatAreaProps) {
  const { activeRoomId, rooms, dms, messages, setMessages, prependMessages, clearUnread, typingUsers } = useChatStore()
  const { user } = useAuthStore()
  const [input, setInput] = useState('')
  const [loadingHistory, setLoadingHistory] = useState(false)
  const [hasMore, setHasMore] = useState(false)
  const [nextCursor, setNextCursor] = useState<number | undefined>()
  const bottomRef = useRef<HTMLDivElement>(null)
  const topRef = useRef<HTMLDivElement>(null)

  const { sendMessage, sendTyping } = useWebSocket(activeRoomId)

  const room = [...rooms, ...dms].find((r) => r.id === activeRoomId)
  const roomMessages = activeRoomId ? (messages[activeRoomId] ?? []) : []
  const typing = activeRoomId ? (typingUsers[activeRoomId] ?? []) : []

  // Загрузка истории при смене комнаты
  useEffect(() => {
    if (!activeRoomId) return
    setNextCursor(undefined)
    setHasMore(false)

    const load = async () => {
      setLoadingHistory(true)
      try {
        const { data } = await roomsApi.getMessages(activeRoomId, undefined, 50)
        setMessages(activeRoomId, data.messages ?? [])
        setHasMore(data.has_more)
        setNextCursor(data.next_cursor)
        clearUnread(activeRoomId)
        roomsApi.markRead(activeRoomId).catch(() => {})
      } finally {
        setLoadingHistory(false)
      }
    }
    load()
  }, [activeRoomId])

  // Скролл вниз при новых сообщениях
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [roomMessages.length])

  const loadMore = async () => {
    if (!activeRoomId || !hasMore || loadingHistory) return
    setLoadingHistory(true)
    try {
      const { data } = await roomsApi.getMessages(activeRoomId, nextCursor, 50)
      prependMessages(activeRoomId, data.messages ?? [])
      setHasMore(data.has_more)
      setNextCursor(data.next_cursor)
    } finally {
      setLoadingHistory(false)
    }
  }

  const handleSend = (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim()) return
    sendMessage(input.trim())
    setInput('')
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInput(e.target.value)
    sendTyping()
  }

  if (!activeRoomId) return null

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-4 border-b border-gray-800 bg-gray-900">
        {/* Back button — только на мобильном */}
        <button
          onClick={onBack}
          className="md:hidden text-gray-400 hover:text-white mr-1"
        >
          ←
        </button>
        <div>
          <h2 className="font-semibold text-white">
            {room?.is_dm ? `@ ${room.name || 'DM'}` : `# ${room?.name}`}
          </h2>
          {room?.description && (
            <p className="text-xs text-gray-500">{room.description}</p>
          )}
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-1">
        {/* Load more */}
        {hasMore && (
          <div className="text-center mb-4">
            <button
              onClick={loadMore}
              disabled={loadingHistory}
              className="text-sm text-indigo-400 hover:text-indigo-300 transition"
            >
              {loadingHistory ? 'Loading...' : 'Load earlier messages'}
            </button>
          </div>
        )}

        {loadingHistory && roomMessages.length === 0 && (
          <div className="text-center text-gray-500 text-sm py-8">Loading...</div>
        )}

        {!loadingHistory && roomMessages.length === 0 && (
          <div className="text-center text-gray-600 text-sm py-8">
            No messages yet. Say hi! 👋
          </div>
        )}

        {roomMessages.map((msg, i) => (
          <MessageBubble
            key={msg.id}
            message={msg}
            isMine={msg.user_id === user?.id}
            showUsername={i === 0 || roomMessages[i - 1].user_id !== msg.user_id}
          />
        ))}

        {/* Typing indicator */}
        {typing.filter((u) => u !== user?.username).length > 0 && (
          <div className="text-xs text-gray-500 italic px-2">
            {typing.filter((u) => u !== user?.username).join(', ')} is typing...
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <form
        onSubmit={handleSend}
        className="flex items-center gap-3 px-4 py-4 border-t border-gray-800 bg-gray-900"
      >
        <input
          type="text"
          value={input}
          onChange={handleInputChange}
          placeholder={`Message ${room?.is_dm ? '' : '#'}${room?.name ?? ''}...`}
          className="flex-1 bg-gray-800 text-white rounded-lg px-4 py-3 outline-none focus:ring-2 focus:ring-indigo-500 transition text-sm"
        />
        <button
          type="submit"
          disabled={!input.trim()}
          className="bg-indigo-600 hover:bg-indigo-500 disabled:bg-gray-700 disabled:cursor-not-allowed text-white rounded-lg px-4 py-3 transition text-sm font-medium"
        >
          Send
        </button>
      </form>
    </div>
  )
}

interface MessageBubbleProps {
  message: Message
  isMine: boolean
  showUsername: boolean
}

function MessageBubble({ message, isMine, showUsername }: MessageBubbleProps) {
  const time = new Date(message.created_at).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
  })
  const edited = message.updated_at !== message.created_at

  return (
    <div className={`flex flex-col ${isMine ? 'items-end' : 'items-start'} group`}>
      {showUsername && !isMine && (
        <span className="text-xs text-gray-500 mb-1 px-2">{message.username}</span>
      )}
      <div className={`
        max-w-[70%] px-4 py-2 rounded-2xl text-sm
        ${isMine
          ? 'bg-indigo-600 text-white rounded-tr-sm'
          : 'bg-gray-800 text-gray-100 rounded-tl-sm'
        }
      `}>
        <p className="break-words">{message.content}</p>
        <div className={`flex items-center gap-1 mt-1 text-xs ${isMine ? 'text-indigo-300' : 'text-gray-500'}`}>
          <span>{time}</span>
          {edited && <span>· edited</span>}
        </div>
      </div>
    </div>
  )
}