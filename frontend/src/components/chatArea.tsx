import { Fragment, useCallback, useContext, useEffect, useImperativeHandle, useMemo, useRef, useState, forwardRef, createContext } from 'react'
import type { FormEvent, ReactNode } from 'react'
import { useChatStore } from '../store/chatStore'
import { useAuthStore } from '../store/authStore'
import { useWebSocket } from '../hooks/useWebSocket'
import { useChatLoader } from '../hooks/useChatLoader'
import { useChatScroll } from '../hooks/useChatScroll'
import { useTyping } from '../hooks/useTyping'
import { roomsApi, dmApi } from '../api/rooms'
import type { Message, RoomInvite, MessageReaction, RoomMember } from '../types'

interface ChatAreaProps {
  onBack: () => void
  setSidebarOpen?: (open: boolean) => void
}

interface ComposerContextValue {
  setInput: (value: string) => void
  composerRef: React.RefObject<HTMLTextAreaElement>
  editingMessage: Message | null
  setEditingMessage: (msg: Message | null) => void
}

const ComposerContext = createContext<ComposerContextValue | null>(null)

function useChatArea() {
  const ctx = useContext(ComposerContext)
  if (!ctx) throw new Error('useChatArea must be used within ChatArea')
  return ctx
}

export function ChatArea({ onBack, setSidebarOpen }: ChatAreaProps) {
  const { activeRoomId, rooms, dms, messages, clearUnread, typingUsers, unreadCounts, getOnlineCount, isUserOnline, updateMessage: storeUpdateMessage, addRoom, setActiveRoom, isMemberMuted, getMutedUntil } = useChatStore()
  const { user } = useAuthStore()
  const [input, setInput] = useState('')
  const [composerError, setComposerError] = useState('')
  const [showInviteModal, setShowInviteModal] = useState(false)
  const [showMembersModal, setShowMembersModal] = useState(false)
  const [members, setMembers] = useState<RoomMember[]>([])
  const [membersLoading, setMembersLoading] = useState(false)
  const [editingMessage, setEditingMessage] = useState<Message | null>(null)
  const lastRenderedMessageId = useRef<string | null>(null)
  const composerRef = useRef<HTMLTextAreaElement>(null)

  // Context menu state
  const [contextMenu, setContextMenu] = useState<{
    x: number
    y: number
    member: RoomMember | null
  } | null>(null)

  // Mute time modal state
  const [muteModal, setMuteModal] = useState<{
    member: RoomMember | null
    isOpen: boolean
  }>({ member: null, isOpen: false })

  const closeContextMenu = () => setContextMenu(null)

  const muteOptions = [
    { label: '5 minutes', minutes: 5 },
    { label: '10 minutes', minutes: 10 },
    { label: '30 minutes', minutes: 30 },
    { label: '1 hour', minutes: 60 },
    { label: '6 hours', minutes: 360 },
    { label: '24 hours', minutes: 1440 },
    { label: '7 days', minutes: 10080 },
  ]

  const handleOpenDM = useCallback(async (targetUserId: string) => {
    try {
      const { data } = await dmApi.openDM(targetUserId)
      addRoom(data)
      setActiveRoom(data.id)
      setShowMembersModal(false)
      setSidebarOpen?.(false)
      closeContextMenu()
    } catch (error) {
      console.error('Failed to open DM:', error)
    }
  }, [addRoom, setActiveRoom, setSidebarOpen])

  const handleKick = useCallback(async (member: RoomMember) => {
    if (!activeRoomId || !confirm(`Kick ${member.username} from this room?`)) return
    try {
      await roomsApi.kickMember(activeRoomId, member.user_id)
      closeContextMenu()
      // Refresh members
      const { data } = await roomsApi.getMembers(activeRoomId)
      setMembers(data ?? [])
    } catch (error) {
      console.error('Failed to kick member:', error)
    }
  }, [activeRoomId])

  const handleMute = useCallback(async (minutes: number) => {
    if (!activeRoomId || !muteModal.member) return
    // eslint-disable-next-line react-hooks/purity
    const now = Date.now()
    const mutedUntil = new Date(now + minutes * 60 * 1000).toISOString()
    try {
      console.log('[DEBUG] Muting member:', muteModal.member.user_id, 'until:', mutedUntil)
      const response = await roomsApi.muteMember(activeRoomId, muteModal.member.user_id, mutedUntil)
      console.log('[DEBUG] Mute response status:', response.status)
      setMuteModal({ member: null, isOpen: false })
      closeContextMenu()
      // Refresh members
      const { data } = await roomsApi.getMembers(activeRoomId)
      console.log('[DEBUG] GetMembers raw response:', data)
      if (data) {
        data.forEach((m: RoomMember) => {
          console.log('[DEBUG] Member:', m.username, 'muted_until:', m.muted_until)
        })
      }
      setMembers(data ?? [])
    } catch (error) {
      console.error('Failed to mute member:', error)
      if (error instanceof Error) {
        alert('Failed to mute: ' + error.message)
      }
    }
  }, [activeRoomId, muteModal.member])

  const handleUnmute = useCallback(async (member: RoomMember) => {
    if (!activeRoomId) return
    try {
      console.log('[DEBUG] Unmuting member:', member.user_id)
      const response = await roomsApi.unmuteMember(activeRoomId, member.user_id)
      console.log('[DEBUG] Unmute response status:', response.status)
      closeContextMenu()
      // Refresh members
      const { data } = await roomsApi.getMembers(activeRoomId)
      console.log('[DEBUG] GetMembers raw response:', data)
      if (data) {
        data.forEach((m: RoomMember) => {
          console.log('[DEBUG] Member:', m.username, 'muted_until:', m.muted_until)
        })
      }
      setMembers(data ?? [])
    } catch (error) {
      console.error('Failed to unmute member:', error)
      if (error instanceof Error) {
        alert('Failed to unmute: ' + error.message)
      }
    }
  }, [activeRoomId])

  const openMuteModal = useCallback((member: RoomMember) => {
    setMuteModal({ member, isOpen: true })
    closeContextMenu()
  }, [])

  const handleContextMenu = useCallback((e: React.MouseEvent, member: RoomMember) => {
    e.preventDefault()
    if (member.user_id === user?.id) return
    setContextMenu({ x: e.clientX, y: e.clientY, member })
  }, [user?.id])

  const handleClickOutside = useCallback((e: MouseEvent) => {
    const target = e.target as Element
    if (contextMenu && !target.closest('[data-context-menu]')) {
      closeContextMenu()
    }
    if (muteModal.isOpen && !target.closest('[data-mute-modal]')) {
      setMuteModal({ member: null, isOpen: false })
    }
  }, [contextMenu, muteModal.isOpen])

  useEffect(() => {
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [handleClickOutside])

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

  const { sendMessage, sendTyping, connectionState, updateMessage: wsUpdateMessage } = useWebSocket(activeRoomId)
  const { onInputChange, cleanup } = useTyping(activeRoomId, sendTyping)
  const { loading, hasMore, error, loadMore } = useChatLoader(activeRoomId, onInitialLoad)

  useEffect(() => cleanup, [cleanup, activeRoomId])
  useEffect(() => {
    lastRenderedMessageId.current = null
  }, [activeRoomId])

  const isOwner = room?.owner_id === user?.id
  const canInvite = Boolean(room?.is_private && !room.is_dm && isOwner)
  const roomTitle = room?.is_dm ? room.name || 'Direct message' : room?.name || 'Chat'

  const currentMember = useMemo(
    () => members.find((m) => m.user_id === user?.id),
    [members, user?.id]
  )
  const currentUserRole = currentMember?.role ?? 'member'

  const canManage = useCallback((targetRole: string) => {
    const roleHierarchy = { owner: 3, admin: 2, member: 1 }
    return roleHierarchy[currentUserRole as keyof typeof roleHierarchy] > roleHierarchy[targetRole as keyof typeof roleHierarchy]
  }, [currentUserRole])

  // Use store's mute status for real-time updates
  const isCurrentUserMuted = activeRoomId && user?.id ? isMemberMuted(activeRoomId, user.id) : false
  const currentUserMutedUntil = activeRoomId && user?.id ? getMutedUntil(activeRoomId, user.id) : undefined

  console.log('[COMPONENT DEBUG] isCurrentUserMuted:', isCurrentUserMuted, 'currentUserMutedUntil:', currentUserMutedUntil, 'activeRoomId:', activeRoomId, 'userId:', user?.id)

  const isMuted = useCallback((member: RoomMember) => {
    if (!activeRoomId) return false
    return isMemberMuted(activeRoomId, member.user_id)
  }, [activeRoomId, isMemberMuted])
  const lastMessage = roomMessages[roomMessages.length - 1]
  const activeUnread = activeRoomId ? unreadCounts[activeRoomId] ?? 0 : 0
  const canSend = connectionState === 'connected'

  const onlineCount = activeRoomId ? getOnlineCount(activeRoomId, user?.id) : 0
  const otherUserId = room?.is_dm ? room.other_user_id : undefined
  const otherUserOnline = activeRoomId && room?.is_dm && otherUserId ? isUserOnline(activeRoomId, otherUserId) : false
  const otherUserLastSeen = undefined

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

  const openMembersModal = useCallback(async () => {
    if (!activeRoomId) return
    setMembersLoading(true)
    try {
      const { data } = await roomsApi.getMembers(activeRoomId)
      setMembers(data ?? [])
    } catch (error) {
      console.error('Failed to load members:', error)
      setMembers([])
    } finally {
      setMembersLoading(false)
      setShowMembersModal(true)
    }
  }, [activeRoomId])

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

    // Check if current user is muted in this room
    if (isCurrentUserMuted) {
      setComposerError('You are muted in this room')
      return
    }

    if (editingMessage) {
      // Optimistic update immediately
      const now = new Date().toISOString()
      storeUpdateMessage(activeRoomId, editingMessage.id, content, now)
      wsUpdateMessage?.(editingMessage.id, content)

      // Sync with server
      roomsApi.editMessage(activeRoomId, editingMessage.id, content)
        .catch((err) => {
          console.error('Failed to edit message:', err)
          // Could add rollback logic here if needed
        })

      setInput('')
      setEditingMessage(null)
      setComposerError('')
    } else {
      if (sendMessage(content)) {
        setInput('')
        setComposerError('')
      }
    }
    if (composerRef.current) {
      composerRef.current.style.height = '44px'
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
    handleSend(event as unknown as FormEvent)
  }

  if (!activeRoomId) return null

  const composerContextValue: ComposerContextValue = {
    setInput,
    composerRef,
    editingMessage,
    setEditingMessage,
  }

  return (
    <ComposerContext.Provider value={composerContextValue}>
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

          <div
            className="flex items-center gap-3 min-w-0 flex-1 p-1"
            onClick={!room?.is_dm ? openMembersModal : undefined}
            role={!room?.is_dm ? 'button' : undefined}
            tabIndex={!room?.is_dm ? 0 : undefined}
            onKeyDown={!room?.is_dm ? (e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openMembersModal(); }} : undefined}
            aria-label={!room?.is_dm ? 'View members' : undefined}
            style={!room?.is_dm ? { cursor: 'pointer' } : undefined}
          >
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
                  : room?.is_dm
                    ? otherUserOnline
                      ? 'online'
                      : otherUserLastSeen
                        ? `last seen ${formatActivity(otherUserLastSeen)}`
                        : 'offline'
                    : `${onlineCount} online`}
              </p>
            </div>
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
              aria-label={editingMessage ? 'Save changes' : 'Send message'}
              className="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-[#229ed9] text-white shadow-sm transition hover:bg-[#168ac0] disabled:cursor-not-allowed disabled:bg-slate-300"
            >
              <Icon name={editingMessage ? 'check' : 'send'} className="h-5 w-5" />
            </button>
          </div>
          {(composerError || !canSend) && (
            <p role="status" className="mx-auto mt-2 max-w-3xl px-2 text-xs text-slate-500">
              {composerError || !canSend ? 'Connecting...' : ''}
            </p>
          )}

          {isCurrentUserMuted && currentUserMutedUntil && (
            <p role="status" className="mx-auto mt-2 max-w-3xl px-2 text-xs text-orange-600">
              You are muted until {new Date(currentUserMutedUntil).toLocaleTimeString()}
            </p>
          )}
        </form>

        {showInviteModal && activeRoomId && (
          <InviteModal roomId={activeRoomId} onClose={() => setShowInviteModal(false)} />
        )}

        <MembersModal
          isOpen={showMembersModal}
          onClose={() => setShowMembersModal(false)}
          members={members}
          loading={membersLoading}
          isDM={Boolean(room?.is_dm)}
          currentUserId={user?.id ?? ''}
          roomId={activeRoomId ?? ''}
          onContextMenu={handleContextMenu}
        />

        {contextMenu && (
          <div
            data-context-menu
            className="fixed z-[100] rounded-xl border border-slate-200 bg-white shadow-lg min-w-[160px] py-1"
            style={{ left: contextMenu.x, top: contextMenu.y }}
          >
            {/* DM */}
            <button
              type="button"
              onClick={() => handleOpenDM(contextMenu.member!.user_id)}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
            >
              <Icon name="message" className="h-4 w-4" />
              Direct Message
            </button>

            {canManage(contextMenu.member!.role) && (
              <>
                <div className="border-t border-slate-100 my-1" />

                {/* Kick - only for private rooms */}
                {room?.is_private && !room?.is_dm && (
                  <button
                    type="button"
                    onClick={() => handleKick(contextMenu.member!)}
                    className="flex w-full items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50"
                  >
                    <Icon name="userMinus" className="h-4 w-4" />
                    Kick
                  </button>
                )}

                {/* Mute/Unmute */}
                {isMuted(contextMenu.member!.user_id) ? (
                  <button
                    type="button"
                    onClick={() => handleUnmute(contextMenu.member!)}
                    className="flex w-full items-center gap-2 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
                  >
                    <Icon name="bell" className="h-4 w-4" />
                    Unmute
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => openMuteModal(contextMenu.member!)}
                    className="flex w-full items-center gap-2 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
                  >
                    <Icon name="bellOff" className="h-4 w-4" />
                    Mute
                  </button>
                )}
              </>
            )}
          </div>
        )}

        {/* Mute Time Modal */}
        {muteModal.isOpen && muteModal.member && (
          <div
            data-mute-modal
            className="fixed inset-0 z-[100] flex items-center justify-center bg-slate-950/60 backdrop-blur-sm p-4"
          >
            <div className="w-full max-w-sm rounded-2xl bg-white shadow-2xl">
              <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4">
                <h3 className="text-sm font-semibold text-slate-950">Mute {muteModal.member.username}</h3>
                <button
                  type="button"
                  onClick={() => setMuteModal({ member: null, isOpen: false })}
                  aria-label="Close"
                  className="grid h-8 w-8 place-items-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-900"
                >
                  <Icon name="close" className="h-4 w-4" />
                </button>
              </div>
              <div className="p-5 space-y-2">
                {muteOptions.map((option) => (
                  <button
                    key={option.minutes}
                    type="button"
                    onClick={() => handleMute(option.minutes)}
                    className="flex w-full items-center justify-between px-3 py-2 text-sm text-slate-700 rounded-lg hover:bg-slate-50 transition"
                  >
                    <span>{option.label}</span>
                  </button>
                ))}
              </div>
            </div>
          </div>
        )}
      </section>
    </ComposerContext.Provider>
  )
}

function MessageBubble({ message, isMine, showUsername }: {
  message: Message
  isMine: boolean
  showUsername: boolean
}) {
  const edited = message.updated_at !== message.created_at
  const isDeleted = message.deleted_at != null
  const { user } = useAuthStore()
  const { addReaction, removeReaction, updateMessage: wsUpdateMessage, deleteMessage: wsDeleteMessage } = useWebSocket(message.room_id)
  const { updateMessage: storeUpdateMessage, deleteMessage: storeDeleteMessage } = useChatStore()
  const { setInput, composerRef, setEditingMessage } = useChatArea()
  const [showReactionPicker, setShowReactionPicker] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const pickerRef = useRef<HTMLDivElement>(null)

  const reactions = message.reactions ?? []

  const handleAddReaction = (emoji: string) => {
    if (isDeleted) return
    if (addReaction(message.id, emoji)) {
      setShowReactionPicker(false)
    }
  }

  const handleRemoveReaction = (emoji: string) => {
    if (isDeleted) return
    removeReaction(message.id, emoji)
    setShowReactionPicker(false)
  }

  const toggleReaction = (emoji: string, isReactedByMe: boolean) => {
    if (isReactedByMe) {
      handleRemoveReaction(emoji)
    } else {
      handleAddReaction(emoji)
    }
  }

  const handleEdit = () => {
    if (isDeleted) return
    setInput(message.content)
    setEditingMessage(message)
    composerRef.current?.focus()
  }

  const handleDelete = () => {
    if (isDeleted) return
    setShowDeleteConfirm(true)
  }

  const confirmDelete = async () => {
    try {
      await roomsApi.deleteMessage(message.room_id, message.id)
      storeDeleteMessage(message.room_id, message.id)
      wsDeleteMessage?.(message.id)
    } catch (error) {
      console.error('Failed to delete message:', error)
    }
    setShowDeleteConfirm(false)
  }

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (pickerRef.current && !pickerRef.current.contains(event.target as Node)) {
        setShowReactionPicker(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  if (isDeleted) {
    return (
      <div data-message-id={message.id} className={`flex ${isMine ? 'justify-end' : 'justify-start'}`}>
        <div className={`flex max-w-[86%] items-end gap-2 md:max-w-[72%] ${isMine ? 'flex-row-reverse' : ''}`}>
          {!isMine && showUsername ? (
            <ConversationAvatar name={message.username} compact />
          ) : (
            !isMine && <div className="h-8 w-8 shrink-0" />
          )}

          <div
            className={`relative rounded-2xl px-3.5 py-2 text-sm leading-5 shadow-sm ${
              isMine
                ? 'rounded-br-md bg-[#dff6d5] text-slate-950'
                : 'rounded-bl-md bg-white text-slate-950'
            }`}
          >
            {showUsername && !isMine && (
              <p className="mb-0.5 text-xs font-semibold text-[#229ed9]">{message.username}</p>
            )}
            <p className="whitespace-pre-wrap break-words text-slate-500 italic" style={{ wordBreak: 'break-word', overflowWrap: 'break-word' }}>
              This message was deleted
            </p>
            <div className="mt-1 flex justify-end gap-1 text-[11px] leading-none text-slate-400">
              <span>{formatTime(message.created_at)}</span>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div data-message-id={message.id} className={`flex ${isMine ? 'justify-end' : 'justify-start'}`}>
      <div className={`flex max-w-[86%] items-end gap-2 md:max-w-[72%] ${isMine ? 'flex-row-reverse' : ''}`}>
        {!isMine && showUsername ? (
          <ConversationAvatar name={message.username} compact />
        ) : (
          !isMine && <div className="h-8 w-8 shrink-0" />
        )}

        <div
          className={`relative rounded-2xl px-3.5 py-2 text-sm leading-5 shadow-sm ${
            isMine
              ? 'rounded-br-md bg-[#dff6d5] text-slate-950'
              : 'rounded-bl-md bg-white text-slate-950'
          }`}
        >
          {showUsername && !isMine && (
            <p className="mb-0.5 text-xs font-semibold text-[#229ed9]">{message.username}</p>
          )}
          <p className="whitespace-pre-wrap break-words" style={{ wordBreak: 'break-word', overflowWrap: 'break-word' }}>{message.content}</p>
          
          {!isDeleted && (
            <ReactionsRow
              reactions={reactions}
              currentUserId={user?.id}
              onToggle={toggleReaction}
              onPickerToggle={setShowReactionPicker}
              isOpen={showReactionPicker}
            />
          )}

          <div className="mt-1 flex justify-end gap-1 text-[11px] leading-none text-slate-400">
            {edited && <span>edited {formatTime(message.updated_at)}</span>}
            <span>{formatTime(message.created_at)}</span>
            {isMine && (
              <div className="flex items-center gap-0.5 ml-2">
                <button
                  type="button"
                  onClick={handleEdit}
                  aria-label="Edit message"
                  className="p-0.5 rounded hover:bg-slate-200 transition"
                >
                  <Icon name="edit" className="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  onClick={handleDelete}
                  aria-label="Delete message"
                  className="p-0.5 rounded hover:bg-slate-200 transition text-red-500"
                >
                  <Icon name="trash" className="h-3.5 w-3.5" />
                </button>
              </div>
            )}
          </div>

          {!isDeleted && showReactionPicker && (
            <ReactionPicker
              ref={pickerRef}
              onSelect={handleAddReaction}
              onClose={() => setShowReactionPicker(false)}
              isMine={isMine}
            />
          )}
        </div>
      </div>

      {showDeleteConfirm && (
        <DeleteConfirmModal onConfirm={confirmDelete} onCancel={() => setShowDeleteConfirm(false)} />
      )}
    </div>
  )
}

function ReactionsRow({ reactions, currentUserId, onToggle, onPickerToggle, isOpen }: {
  reactions: MessageReaction[]
  currentUserId: string | undefined
  onToggle: (emoji: string, isReactedByMe: boolean) => void
  onPickerToggle: (open: boolean) => void
  isOpen: boolean
}) {
  return (
    <div className="mt-1.5 flex flex-wrap items-center gap-1.5 max-w-[280px]">
      {reactions.map((reaction) => {
        const isReactedByMe =
          reaction.isReactedByMe || reaction.users.includes(currentUserId ?? '')

        return (
          <button
            key={reaction.emoji}
            type="button"
            onClick={() => onToggle(reaction.emoji, isReactedByMe)}
            onContextMenu={(e) => {
              e.preventDefault()
              onPickerToggle(true)
            }}
            className={`inline-flex h-6 items-center gap-1 rounded-full border px-2 text-xs transition shrink-0 ${
              isReactedByMe
                ? 'border-[#229ed9]/30 bg-[#229ed9]/10 text-[#229ed9]'
                : 'border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50 shadow-sm'
            }`}
            aria-label={`${reaction.emoji} ${reaction.count}`}
          >
            <span className="text-sm leading-none">{reaction.emoji}</span>

            {reaction.count > 1 && (
              <span className="font-medium leading-none text-slate-500">
                {reaction.count}
              </span>
            )}
          </button>
        )
      })}

      <button
        type="button"
        onClick={() => onPickerToggle(!isOpen)}
        onContextMenu={(e) => e.preventDefault()}
        className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-base leading-none text-slate-400 transition hover:bg-slate-100 hover:text-slate-700 border border-slate-200 bg-white shadow-sm"
        aria-label="Add reaction"
        title="Add reaction"
      >
        +
      </button>
    </div>
  )
}

const ReactionPicker = forwardRef<
  HTMLDivElement,
  {
    onSelect: (emoji: string) => void
    onClose: () => void
    isMine: boolean
  }
>(({ onSelect, onClose, isMine }, ref) => {
  const emojis = useMemo(
    () => ['👍', '👎', '❤️', '😂', '😮', '😢', '🔥', '🤡', '💩', '🎉', '👏', '🤔', '🙏', '💯'],
    [],
  )

  const containerRef = useRef<HTMLDivElement>(null)
  const buttonRefs = useRef<(HTMLButtonElement | null)[]>([])
  const [focusedIndex, setFocusedIndex] = useState(0)

  useImperativeHandle(ref, () => containerRef.current, [])

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const handleKeyDown = (e: KeyboardEvent) => {
      switch (e.key) {
        case 'ArrowRight':
          e.preventDefault()
          setFocusedIndex((i) => (i + 1) % emojis.length)
          break

        case 'ArrowLeft':
          e.preventDefault()
          setFocusedIndex((i) => (i - 1 + emojis.length) % emojis.length)
          break

        case 'ArrowDown':
          e.preventDefault()
          setFocusedIndex((i) => (i + 7) % emojis.length)
          break

        case 'ArrowUp':
          e.preventDefault()
          setFocusedIndex((i) => (i - 7 + emojis.length) % emojis.length)
          break

        case 'Enter':
        case ' ':
          e.preventDefault()
          onSelect(emojis[focusedIndex])
          onClose()
          break

        case 'Escape':
          e.preventDefault()
          onClose()
          break
      }
    }

    container.addEventListener('keydown', handleKeyDown)
    return () => container.removeEventListener('keydown', handleKeyDown)
  }, [emojis, focusedIndex, onSelect, onClose])

  useEffect(() => {
    buttonRefs.current[focusedIndex]?.focus()
  }, [focusedIndex])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 backdrop-blur-sm">
      <div
        ref={containerRef}
        className="grid grid-cols-7 gap-2 rounded-2xl border border-slate-200 bg-white p-4 shadow-2xl"
        role="dialog"
        aria-label="Choose reaction"
      >
        {emojis.map((emoji, index) => (
          <button
            key={emoji}
            ref={(el) => {
              buttonRefs.current[index] = el
            }}
            type="button"
            onClick={() => {
              onSelect(emoji)
              onClose()
            }}
            className={`grid h-12 w-12 place-items-center rounded-xl text-xl transition ${
              index === focusedIndex
                ? 'bg-slate-100 ring-2 ring-[#229ed9] scale-105'
                : 'hover:bg-slate-100 hover:scale-105'
            }`}
            aria-label={emoji}
            aria-selected={index === focusedIndex}
          >
            {emoji}
          </button>
        ))}
      </div>
    </div>
  )
})

ReactionPicker.displayName = 'ReactionPicker'

function DeleteConfirmModal({ onConfirm, onCancel }: { onConfirm: () => void; onCancel: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 backdrop-blur-sm">
      <div className="w-full max-w-sm rounded-2xl bg-white shadow-2xl">
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <h3 className="text-sm font-semibold text-slate-950">Delete message</h3>
          <button
            type="button"
            onClick={onCancel}
            aria-label="Close"
            className="grid h-8 w-8 place-items-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-900"
          >
            <Icon name="close" className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4 p-5">
          <p className="text-center text-sm text-slate-700">Are you sure you want to delete this message?</p>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={onCancel}
              className="flex-1 h-10 rounded-full border border-slate-300 text-sm font-medium text-slate-700 transition hover:bg-slate-50"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={onConfirm}
              className="flex-1 h-10 rounded-full bg-red-600 text-sm font-medium text-white transition hover:bg-red-700"
            >
              Delete
            </button>
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

function MembersModal({ isOpen, onClose, members, loading, isDM, currentUserId, roomId, onContextMenu }: {
  isOpen: boolean
  onClose: () => void
  members: RoomMember[]
  loading: boolean
  isDM: boolean
  currentUserId: string
  roomId: string
  onContextMenu?: (e: React.MouseEvent, member: RoomMember) => void
}) {
  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (event.key === 'Escape') onClose()
  }, [onClose])

  useEffect(() => {
    if (!isOpen) return
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, handleKeyDown])

  const getRoleLabel = (role: string) => {
    switch (role) {
      case 'owner': return 'Owner'
      case 'admin': return 'Admin'
      default: return 'Member'
    }
  }

  const getRoleColor = (role: string) => {
    switch (role) {
      case 'owner': return 'bg-amber-100 text-amber-700'
      case 'admin': return 'bg-blue-100 text-blue-700'
      default: return 'bg-slate-100 text-slate-700'
    }
  }

  // Use store's mute status
  const isMuted = useCallback((userId: string) => {
    if (!roomId) return false
    const store = useChatStore.getState()
    const result = store.isMemberMuted(roomId, userId)
    console.log('[MEMBERS MODAL DEBUG] isMuted:', userId, 'result:', result)
    return result
  }, [roomId])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 backdrop-blur-sm p-4">
      <div className="w-full max-w-md rounded-2xl bg-white shadow-2xl max-h-[80vh] flex flex-col">
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4 sticky top-0 bg-white z-10 rounded-t-2xl">
          <h3 className="text-sm font-semibold text-slate-950">{isDM ? 'Participants' : 'Members'}</h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="grid h-8 w-8 place-items-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-900"
          >
            <Icon name="close" className="h-4 w-4" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-5">
          {loading ? (
            <div className="py-10 text-center text-sm text-slate-500">Loading members...</div>
          ) : members.length === 0 ? (
            <div className="py-10 text-center text-sm text-slate-500">No members</div>
          ) : (
            <ul className="space-y-2" role="list">
              {members.map((member) => (
                onContextMenu && member.user_id !== currentUserId ? (
                  <li
                    key={member.user_id}
                    className="flex items-center gap-3 relative"
                    onContextMenu={(e) => onContextMenu(e, member)}
                  >
                    <ConversationAvatar name={member.username} isDM={isDM} compact />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="truncate text-sm font-semibold text-slate-950">{member.username}</p>
                        <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${getRoleColor(member.role)}`}>
                          {getRoleLabel(member.role)}
                        </span>
                        {isMuted(member.user_id) && (
                          <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium bg-orange-100 text-orange-700">
                            <Icon name="bell" className="h-3 w-3" />
                            Muted
                          </span>
                        )}
                      </div>
                    </div>
                  </li>
                ) : (
                  <li
                    key={member.user_id}
                    className="flex items-center gap-3 relative"
                  >
                    <ConversationAvatar name={member.username} isDM={isDM} compact />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="truncate text-sm font-semibold text-slate-950">{member.username}</p>
                        <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${getRoleColor(member.role)}`}>
                          {getRoleLabel(member.role)}
                        </span>
                        {isMuted(member.user_id) && (
                          <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium bg-orange-100 text-orange-700">
                            <Icon name="bell" className="h-3 w-3" />
                            Muted
                          </span>
                        )}
                      </div>
                    </div>
                  </li>
                )
              ))}
            </ul>
          )}
        </div>

        <div className="border-t border-slate-100 px-5 py-3 sticky bottom-0 bg-white z-10 rounded-b-2xl">
          <p className="text-center text-xs text-slate-500">{members.length} member{members.length !== 1 ? 's' : ''}</p>
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

type IconName = 'back' | 'send' | 'link' | 'lock' | 'close' | 'down' | 'smile' | 'edit' | 'trash' | 'check' | 'message' | 'userMinus' | 'bell' | 'bellOff'

function Icon({ name, className }: { name: IconName; className?: string }) {
  const paths: Record<IconName, ReactNode> = {
    back: <path d="M15 18l-6-6 6-6M9 12h12" />,
    send: <path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z" />,
    link: <path d="M10 13a5 5 0 0 0 7.1 0l2-2a5 5 0 0 0-7.1-7.1l-1.1 1.1M14 11a5 5 0 0 0-7.1 0l-2 2A5 5 0 0 0 12 20.1l1.1-1.1" />,
    lock: <path d="M7 11V8a5 5 0 0 1 10 0v3M6 11h12v10H6V11z" />,
    close: <path d="M18 6L6 18M6 6l12 12" />,
    down: <path d="M12 5v14M19 12l-7 7-7-7" />,
    smile: <path d="M22 11c0 6.1-4.9 11-11 11S0 17.1 0 11s4.9-11 11-11 11 4.9 11 11zM8 9a2 2 0 1 1-4 0 2 2 0 0 1 4 0zM16 9a2 2 0 1 1-4 0 2 2 0 0 1 4 0zM21 11a10 10 0 0 0-6.3-9.3.9.9 0 0 0-1.4 0A10 10 0 0 0 3 11c0 2.5 1.2 4.7 3 6M11 17a4 4 0 0 1 0-8" />,
    edit: <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />,
    trash: <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />,
    check: <path d="M20 6 9 17l-5-5" />,
    message: <path d="M21 15a4 4 0 0 1-4 4H8l-5 3V7a4 4 0 0 1 4-4h10a4 4 0 0 1 4 4v8zM17 9h-1v4h1v-4zm-4 0H9v4h4V9zm4 6H9v4h8v-4z" />,
    userMinus: <path d="M16 21v-2a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM4 7h16" />,
    bell: <path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" />,
    bellOff: <path d="M8.7 3.3a20.2 20.2 0 0 0 0 21.4M22.5 8.5a20.2 20.2 0 0 1-20.2 10.2M18 8a6 6 0 0 0-9.3 3.3m-4.7 5.7A6.1 6.1 0 0 0 14 18c0 7-3 7-3 9M10 21h4M1 1l22 22" />,
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

function formatActivity(value: string) {
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function sameDay(left: string, right: string) {
  const a = new Date(left)
  const b = new Date(right)
  return a.getFullYear() === b.getFullYear()
    && a.getMonth() === b.getMonth()
    && a.getDate() === b.getDate()
}