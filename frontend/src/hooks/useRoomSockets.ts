import { useCallback, useEffect, useMemo, useRef } from 'react'
import { roomSocketHub } from '../services/roomSocketHub.ts'
import { useAuthStore } from '../store/authStore'
import { useChatStore } from '../store/chatStore'
import { roomsApi } from '../api/rooms'
import type {
  MessageDeletedPayload,
  MessageEditedPayload,
  NewMessagePayload,
  Room,
  UserTypingPayload,
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
