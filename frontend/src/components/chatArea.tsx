import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent, KeyboardEvent, ReactNode } from 'react'
import { useChatStore } from '../store/chatStore'
import { useAuthStore } from '../store/authStore'
import { useWebSocket } from '../hooks/useWebSocket'
import { useChatLoader } from '../hooks/useChatLoader'
import { useChatScroll } from '../hooks/useChatScroll'
import { useTyping } from '../hooks/useTyping'
import { roomsApi } from '../api/rooms'
import type { Message, RoomInvite } from '../types'

interface ChatAreaProps {
  onBack: () => void
}

export function ChatArea({ onBack }: ChatAreaProps) {
  const { activeRoomId, rooms, dms, messages, clearUnread, typingUsers, unreadCounts } = useChatStore()
  const { user } = useAuthStore()
  const [input, setInput] = useState('')
  const [composerError, setComposerError] = useState('')
  const [showInviteModal, setShowInviteModal] = useState(false)
  const lastRenderedMessageId = useRef<string | null>(null)
  const composerRef = useRef<HTMLTextAreaElement>(null)

  const room = useMemo(
    () => [...rooms, ...dms].find((item) => item.id === activeRoomId),
    [rooms, dms, activeRoomId]
  )
  const roomMessages = activeRoomId ? (messages[activeRoomId] ?? []) : []
  const typing = activeRoomId ? (typingUsers[activeRoomId] ?? []) : []
  const otherTyping = typing.filter((username) => username !== user?.username)

  const {
    containerRef,
    bottomRef,
    isAtBottom,
    scrollToBottom,
    shouldAutoScrollForNewMessage,
    saveScrollPosition,
    restoreScrollPosition,
    trackScroll,
    onInitialLoad,
  } = useChatScroll()

  const { sendMessage, sendTyping, connectionState } = useWebSocket(activeRoomId)
  const { onInputChange, cleanup } = useTyping(activeRoomId, sendTyping)
  const { loading, hasMore, error, loadMore } = useChatLoader(activeRoomId, onInitialLoad)

  useEffect(() => cleanup, [cleanup, activeRoomId])
  useEffect(() => {
    lastRenderedMessageId.current = null
  }, [activeRoomId])

  const isOwner = room?.owner_id === user?.id
  const canInvite = Boolean(room?.is_private && !room.is_dm && isOwner)
  const roomTitle = room?.is_dm ? room.name || 'Direct message' : room?.name || 'Chat'
  const lastMessage = roomMessages[roomMessages.length - 1]
  const activeUnread = activeRoomId ? unreadCounts[activeRoomId] ?? 0 : 0
  const canSend = connectionState === 'connected'

  const handleLoadMore = useCallback(() => {
    void loadMore(saveScrollPosition, restoreScrollPosition)
  }, [loadMore, saveScrollPosition, restoreScrollPosition])

  const markActiveRoomRead = useCallback(() => {
    if (!activeRoomId || activeUnread === 0) return
    clearUnread(activeRoomId)
    roomsApi.markRead(activeRoomId).catch(() => {})
  }, [activeRoomId, activeUnread, clearUnread])

  const handleScroll = useCallback(() => {
    if (trackScroll()) markActiveRoomRead()
  }, [trackScroll, markActiveRoomRead])

  useEffect(() => {
    if (!lastMessage) return

    const previousLastId = lastRenderedMessageId.current
    lastRenderedMessageId.current = lastMessage.id

    if (!previousLastId || previousLastId === lastMessage.id) return

    const isMine = lastMessage.user_id === user?.id
    if (!isMine && !shouldAutoScrollForNewMessage()) return

    requestAnimationFrame(() => {
      scrollToBottom('smooth')
      markActiveRoomRead()
    })
  }, [
    lastMessage,
    markActiveRoomRead,
    scrollToBottom,
    shouldAutoScrollForNewMessage,
    user?.id,
  ])

  const handleSend = (event: FormEvent) => {
    event.preventDefault()
    const content = input.trim()
    if (!content || !activeRoomId) return
    if (!canSend) {
      setComposerError('Waiting for connection')
      return
    }

    if (sendMessage(content)) {
      setInput('')
      setComposerError('')
      if (composerRef.current) {
        composerRef.current.style.height = '44px'
      }
    }
  }

  const handleComposerChange = (value: string) => {
    setInput(value)
    setComposerError('')
    onInputChange()

    const el = composerRef.current
    if (!el) return
    el.style.height = '44px'
    el.style.height = `${Math.min(el.scrollHeight, 144)}px`
  }

  const handleComposerKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key !== 'Enter' || event.shiftKey) return
    event.preventDefault()
    event.currentTarget.form?.requestSubmit()
  }

  if (!activeRoomId) return null

  return (
    <section className="relative flex h-full min-w-0 flex-col bg-[#eef3f8] text-slate-950">
      <header className="flex min-h-[64px] items-center gap-3 border-b border-slate-200 bg-white px-3 shadow-sm md:px-5">
        <button
          type="button"
          onClick={onBack}
          aria-label="Back to chats"
          className="grid h-10 w-10 place-items-center rounded-full text-slate-500 transition hover:bg-slate-100 hover:text-slate-900 md:hidden"
        >
          <Icon name="back" className="h-5 w-5" />
        </button>

        <ConversationAvatar name={roomTitle} isDM={Boolean(room?.is_dm)} />

        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h2 className="truncate text-[15px] font-semibold leading-5 text-slate-950">
              {room?.is_dm ? roomTitle : `#${roomTitle}`}
            </h2>
            {room?.is_private && !room.is_dm && (
              <Icon name="lock" className="h-3.5 w-3.5 shrink-0 text-slate-400" />
            )}
          </div>
          <p className="truncate text-xs text-slate-500">
            {otherTyping.length > 0
              ? `${otherTyping.join(', ')} typing`
              : connectionLabel(connectionState)}
          </p>
        </div>

        {canInvite && (
          <button
            type="button"
            onClick={() => setShowInviteModal(true)}
            aria-label="Create invite"
            title="Create invite"
            className="grid h-10 w-10 place-items-center rounded-full text-slate-500 transition hover:bg-slate-100 hover:text-[#229ed9]"
          >
            <Icon name="link" className="h-5 w-5" />
          </button>
        )}
      </header>

      <div
        ref={containerRef}
        onScroll={handleScroll}
        role="log"
        aria-label="Messages"
        aria-live="polite"
        aria-relevant="additions text"
        aria-busy={loading}
        className="min-h-0 flex-1 overflow-y-auto px-3 py-4 md:px-8"
      >
        <div className="mx-auto flex w-full max-w-3xl flex-col gap-2">
          {hasMore && (
            <button
              type="button"
              onClick={handleLoadMore}
              disabled={loading}
              className="mx-auto rounded-full bg-white px-4 py-2 text-xs font-medium text-[#229ed9] shadow-sm transition hover:bg-slate-50 disabled:text-slate-400"
            >
              {loading ? 'Loading...' : 'Load earlier messages'}
            </button>
          )}

          {error && (
            <div className="mx-auto rounded-full bg-red-50 px-4 py-2 text-xs font-medium text-red-600">
              {error}
            </div>
          )}

          {loading && roomMessages.length === 0 && (
            <div className="py-10 text-center text-sm text-slate-500">Loading messages...</div>
          )}

          {!loading && roomMessages.length === 0 && (
            <div className="py-10 text-center text-sm text-slate-500">No messages yet</div>
          )}

          {roomMessages.map((message, index) => {
            const previous = roomMessages[index - 1]
            const showDate = !previous || !sameDay(previous.created_at, message.created_at)
            const showUsername = !previous || previous.user_id !== message.user_id || showDate

            return (
              <Fragment key={message.id}>
                {showDate && <DateDivider value={message.created_at} />}
                <MessageBubble
                  message={message}
                  isMine={message.user_id === user?.id}
                  showUsername={showUsername}
                />
              </Fragment>
            )
          })}

          <div ref={bottomRef} />
        </div>
      </div>

      {!isAtBottom && (
        <button
          type="button"
          onClick={() => {
            scrollToBottom('smooth')
            markActiveRoomRead()
          }}
          aria-label="Scroll to latest messages"
          className="absolute bottom-20 right-4 grid h-11 w-11 place-items-center rounded-full bg-white text-[#229ed9] shadow-lg shadow-slate-300/60 transition hover:bg-slate-50 md:right-8"
        >
          <Icon name="down" className="h-5 w-5" />
        </button>
      )}

      <form
        onSubmit={handleSend}
        className="border-t border-slate-200 bg-white px-3 py-3 md:px-5"
      >
        <div className="mx-auto flex max-w-3xl items-end gap-2">
          <label className="sr-only" htmlFor="message-composer">Message</label>
          <textarea
            id="message-composer"
            ref={composerRef}
            value={input}
            onChange={(event) => {
              handleComposerChange(event.target.value)
            }}
            onKeyDown={handleComposerKeyDown}
            placeholder={`Message ${room?.is_dm ? roomTitle : `#${roomTitle}`}`}
            rows={1}
            className="min-h-11 max-h-36 min-w-0 flex-1 resize-none rounded-3xl border border-transparent bg-slate-100 px-4 py-3 text-sm leading-5 text-slate-950 outline-none transition placeholder:text-slate-400 focus:border-[#229ed9] focus:bg-white"
          />
          <button
            type="submit"
            disabled={!input.trim() || !canSend}
            aria-label="Send message"
            className="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-[#229ed9] text-white shadow-sm transition hover:bg-[#168ac0] disabled:cursor-not-allowed disabled:bg-slate-300"
          >
            <Icon name="send" className="h-5 w-5" />
          </button>
        </div>
        {(composerError || !canSend) && (
          <p role="status" className="mx-auto mt-2 max-w-3xl px-2 text-xs text-slate-500">
            {composerError || connectionLabel(connectionState)}
          </p>
        )}
      </form>

      {showInviteModal && activeRoomId && (
        <InviteModal roomId={activeRoomId} onClose={() => setShowInviteModal(false)} />
      )}
    </section>
  )
}

function MessageBubble({ message, isMine, showUsername }: {
  message: Message
  isMine: boolean
  showUsername: boolean
}) {
  const edited = message.updated_at !== message.created_at

  return (
    <div data-message-id={message.id} className={`flex ${isMine ? 'justify-end' : 'justify-start'}`}>
      <div className={`flex max-w-[86%] items-end gap-2 md:max-w-[72%] ${isMine ? 'flex-row-reverse' : ''}`}>
        {!isMine && showUsername ? (
          <ConversationAvatar name={message.username} compact />
        ) : (
          !isMine && <div className="h-8 w-8 shrink-0" />
        )}

        <div
          className={`rounded-2xl px-3.5 py-2 text-sm leading-5 shadow-sm ${
            isMine
              ? 'rounded-br-md bg-[#dff6d5] text-slate-950'
              : 'rounded-bl-md bg-white text-slate-950'
          }`}
        >
          {showUsername && !isMine && (
            <p className="mb-0.5 text-xs font-semibold text-[#229ed9]">{message.username}</p>
          )}
          <p className="whitespace-pre-wrap break-words" style={{ wordBreak: 'break-word', overflowWrap: 'break-word' }}>{message.content}</p>
          <div className="mt-1 flex justify-end gap-1 text-[11px] leading-none text-slate-400">
            {edited && <span>edited</span>}
            <span>{formatTime(message.created_at)}</span>
          </div>
        </div>
      </div>
    </div>
  )
}

function InviteModal({ roomId, onClose }: { roomId: string; onClose: () => void }) {
  const [invite, setInvite] = useState<RoomInvite | null>(null)
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let alive = true

    roomsApi.createInvite(roomId, 10, 168)
      .then(({ data }) => {
        if (alive) setInvite(data)
      })
      .finally(() => {
        if (alive) setLoading(false)
      })

    return () => {
      alive = false
    }
  }, [roomId])

  const handleCopy = async () => {
    if (!invite) return
    await navigator.clipboard.writeText(invite.token)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="absolute inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4 backdrop-blur-sm">
      <div className="w-full max-w-sm rounded-2xl bg-white shadow-2xl">
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <h3 className="text-sm font-semibold text-slate-950">Invite link</h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="grid h-8 w-8 place-items-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-900"
          >
            <Icon name="close" className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4 p-5">
          {loading ? (
            <p className="text-center text-sm text-slate-500">Generating invite...</p>
          ) : invite ? (
            <>
              <div className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5">
                <code className="break-all text-sm font-medium text-slate-900">{invite.token}</code>
              </div>
              <div className="text-xs text-slate-500">
                <p>Max uses: {invite.max_uses}</p>
                {invite.expires_at && <p>Expires: {new Date(invite.expires_at).toLocaleDateString()}</p>}
              </div>
              <button
                type="button"
                onClick={handleCopy}
                className="h-10 w-full rounded-full bg-[#229ed9] text-sm font-semibold text-white transition hover:bg-[#168ac0]"
              >
                {copied ? 'Copied' : 'Copy token'}
              </button>
            </>
          ) : (
            <p className="text-center text-sm text-red-600">Could not create invite</p>
          )}
        </div>
      </div>
    </div>
  )
}

function DateDivider({ value }: { value: string }) {
  return (
    <div className="my-3 flex justify-center">
      <span className="rounded-full bg-slate-400/70 px-3 py-1 text-[11px] font-medium text-white shadow-sm">
        {formatDay(value)}
      </span>
    </div>
  )
}

function ConversationAvatar({ name, isDM = false, compact = false }: {
  name: string
  isDM?: boolean
  compact?: boolean
}) {
  const initial = (name.trim()[0] || '#').toUpperCase()

  return (
    <div
      className={`${compact ? 'h-8 w-8 text-xs' : 'h-10 w-10 text-sm'} grid shrink-0 place-items-center rounded-full ${
        isDM ? 'bg-gradient-to-br from-[#35b779] to-[#229ed9]' : 'bg-gradient-to-br from-[#f59f00] to-[#e8590c]'
      } font-semibold text-white`}
    >
      {initial}
    </div>
  )
}

type IconName = 'back' | 'send' | 'link' | 'lock' | 'close' | 'down'

function Icon({ name, className }: { name: IconName; className?: string }) {
  const paths: Record<IconName, ReactNode> = {
    back: <path d="M15 18l-6-6 6-6M9 12h12" />,
    send: <path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z" />,
    link: <path d="M10 13a5 5 0 0 0 7.1 0l2-2a5 5 0 0 0-7.1-7.1l-1.1 1.1M14 11a5 5 0 0 0-7.1 0l-2 2A5 5 0 0 0 12 20.1l1.1-1.1" />,
    lock: <path d="M7 11V8a5 5 0 0 1 10 0v3M6 11h12v10H6V11z" />,
    close: <path d="M18 6L6 18M6 6l12 12" />,
    down: <path d="M12 5v14M19 12l-7 7-7-7" />,
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

function connectionLabel(state: string) {
  switch (state) {
    case 'connected':
      return 'online'
    case 'connecting':
      return 'connecting...'
    case 'reconnecting':
      return 'reconnecting...'
    default:
      return 'offline'
  }
}

function formatTime(value: string) {
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatDay(value: string) {
  return new Date(value).toLocaleDateString([], { day: 'numeric', month: 'long' })
}

function sameDay(left: string, right: string) {
  const a = new Date(left)
  const b = new Date(right)
  return a.getFullYear() === b.getFullYear()
    && a.getMonth() === b.getMonth()
    && a.getDate() === b.getDate()
}
