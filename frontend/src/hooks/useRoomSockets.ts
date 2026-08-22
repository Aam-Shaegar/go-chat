import { useCallback, useEffect, useMemo, useRef } from 'react'
import { roomSocketHub } from '../services/roomSocketHub.ts'
import { useAuthStore } from '../store/authStore'
import { useChatStore } from '../store/chatStore'
import { roomsApi } from '../api/rooms'
import type {
  MessageDeletedPayload,
  MessageEditedPayload,
  NewMessagePayload,
  ReactionPayload,
  Room,
  UserJoinedPayload,
  UserLeftPayload,
  UserMutedPayload,
  UserTypingPayload,
  UserUnmutedPayload,
  WSEvent,
} from '../types'

const TYPING_CLEAR_MS = 3000

export function useRoomSockets(rooms: Room[], dms: Room[]) {
  const { accessToken } = useAuthStore()
  const typingTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map())

  const roomIds = useMemo(
    () => [...new Set([...rooms, ...dms].map((room) => room.id))].sort(),
    [rooms, dms]
  )
  const roomIdsKey = roomIds.join('|')

  const handleEvent = useCallback((roomId: string, event: WSEvent) => {
    const chat = useChatStore.getState()
    const auth = useAuthStore.getState()

    switch (event.type) {
      case 'new_message': {
        const p = event.payload as NewMessagePayload
        const targetRoomId = p.room_id || roomId
        const message = { ...toMessage(p), room_id: targetRoomId }
        const added = chat.addMessage(targetRoomId, message)
        if (!added) break

        const isMine = p.user_id === auth.user?.id
        const isActive = targetRoomId === chat.activeRoomId

	        if (isMine || isActive) {
	          chat.clearUnread(targetRoomId)
	          if (isActive && !isMine) roomsApi.markRead(targetRoomId).catch(() => {})
	        } else {
	          chat.incrementUnread(targetRoomId)
	          notifyIncomingMessage(p)
        }
        break
      }
      case 'message_edited': {
        const p = event.payload as MessageEditedPayload
        chat.updateMessage(p.room_id, p.message_id, p.content, p.updated_at)
        break
      }
      case 'message_deleted': {
        const p = event.payload as MessageDeletedPayload
        chat.deleteMessage(p.room_id, p.message_id)
        break
      }
      case 'user_typing': {
        const p = event.payload as UserTypingPayload
        if (p.user_id === auth.user?.id) break

        chat.setTyping(p.room_id, p.username)

        const key = `${p.room_id}:${p.username}`
        const existing = typingTimers.current.get(key)
        if (existing) clearTimeout(existing)

        const timer = setTimeout(() => {
          useChatStore.getState().clearTyping(p.room_id, p.username)
          typingTimers.current.delete(key)
        }, TYPING_CLEAR_MS)

        typingTimers.current.set(key, timer)
        break
      }
      case 'reaction_added': {
        const p = event.payload as ReactionPayload
        chat.addReaction(p.room_id, p.message_id, p.emoji, p.user_id, p.is_reacted_by_me)
        break
      }
      case 'reaction_removed': {
        const p = event.payload as ReactionPayload
        chat.removeReaction(p.room_id, p.message_id, p.emoji, p.user_id, p.is_reacted_by_me)
        break
      }
      case 'user_joined': {
        const p = event.payload as UserJoinedPayload
        if (p.user_id === auth.user?.id) break
        chat.setUserOnline(p.room_id, p.user_id)
        break
      }
      case 'user_left': {
        const p = event.payload as UserLeftPayload
        if (p.user_id === auth.user?.id) break
        chat.setUserOffline(p.room_id, p.user_id)
        break
      }
      case 'user_muted': {
        const p = event.payload as UserMutedPayload
        chat.setMemberMuted(p.room_id, p.user_id, p.muted_until)
        break
      }
      case 'user_unmuted': {
        const p = event.payload as UserUnmutedPayload
        chat.setMemberUnmuted(p.room_id, p.user_id)
        break
      }
      case 'error': {
        const p = event.payload as { message: string }
        // Rollback optimistic reaction updates on error
        // The error payload doesn't have enough info to know which reaction failed,
        // so we rely on the server not sending reaction_added/removed for failed operations
        console.warn('[WS] Server error:', p.message)

        // Handle mute error - when muted user tries to send message
        if (p.message.toLowerCase().includes('muted')) {
          const activeRoomId = chat.activeRoomId
          if (activeRoomId && auth.user?.id) {
            // Mute until we get the actual user_muted event with the correct time
            // Use a far future date as temporary mute
            chat.setMemberMuted(activeRoomId, auth.user.id, new Date(Date.now() + 86400000).toISOString())
          }
        }
        break
      }
      default:
        break
    }
  }, [])

  useEffect(() => {
    const timers = typingTimers.current

	    roomSocketHub.setCallbacks({
	      onMessage: handleEvent,
	      onStateChange: (_roomId, state) => {
	        const chat = useChatStore.getState()
	        for (const room of [...chat.rooms, ...chat.dms]) {
	          chat.setConnectionState(room.id, state)
	        }
	      },
	    })

    return () => {
      roomSocketHub.setCallbacks(null)
      timers.forEach((timer) => clearTimeout(timer))
      timers.clear()
    }
  }, [handleEvent])

  useEffect(() => {
	    if (!accessToken) {
	      roomSocketHub.disconnectAll()
	      return
	    }

    roomSocketHub.sync(roomIds, accessToken)

    return () => {
      roomSocketHub.releaseSoon()
    }
  }, [accessToken, roomIds, roomIdsKey])
}

function toMessage(payload: NewMessagePayload) {
  return {
    id: payload.id,
    room_id: payload.room_id,
    user_id: payload.user_id,
    username: payload.username,
    reply_to_id: payload.reply_to_id,
    content: payload.content,
    is_encrypted: payload.is_encrypted,
    created_at: payload.created_at,
    updated_at: payload.created_at,
  }
}

function notifyIncomingMessage(payload: NewMessagePayload) {
  if (typeof document !== 'undefined') {
    document.dispatchEvent(new CustomEvent('gochat:message', { detail: payload }))
  }

  if (
    typeof Notification === 'undefined' ||
    Notification.permission !== 'granted' ||
    document.visibilityState === 'visible'
  ) {
    return
  }

  const title = payload.username || 'New message'
  const body = payload.content.length > 120
    ? `${payload.content.slice(0, 117)}...`
    : payload.content

  new Notification(title, { body })
}
