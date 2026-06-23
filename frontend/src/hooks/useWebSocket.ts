import { useEffect, useRef, useCallback } from 'react'
import { useChatStore } from '../store/chatStore'
import { useAuthStore } from '../store/authStore'
import type {
  WSEvent,
  NewMessagePayload,
  MessageEditedPayload,
  MessageDeletedPayload,
  UserTypingPayload,
} from '../types'

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:5050/api/v1'

export function useWebSocket(roomId: string | null) {
  const ws = useRef<WebSocket | null>(null)
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | null>(null)
  const { accessToken } = useAuthStore()
  const {
    addMessage,
    updateMessage,
    deleteMessage,
    setTyping,
    clearTyping,
    activeRoomId,
    incrementUnread,
  } = useChatStore()

  const handleEvent = useCallback((event: WSEvent) => {
    switch (event.type) {
      case 'new_message': {
        const p = event.payload as NewMessagePayload
        addMessage(p.room_id, {
          id: p.id,
          room_id: p.room_id,
          user_id: p.user_id,
          username: p.username,
          reply_to_id: p.reply_to_id,
          content: p.content,
          is_encrypted: p.is_encrypted,
          created_at: p.created_at,
          updated_at: p.created_at,
        })
        if (p.room_id !== activeRoomId) {
          incrementUnread(p.room_id)
        }
        break
      }
      case 'message_edited': {
        const p = event.payload as MessageEditedPayload
        updateMessage(p.room_id, p.message_id, p.content, p.updated_at)
        break
      }
      case 'message_deleted': {
        const p = event.payload as MessageDeletedPayload
        deleteMessage(p.room_id, p.message_id)
        break
      }
      case 'user_typing': {
        const p = event.payload as UserTypingPayload
        setTyping(p.room_id, p.username)
        setTimeout(() => clearTyping(p.room_id, p.username), 3000)
        break
      }
    }
  }, [addMessage, updateMessage, deleteMessage, setTyping, clearTyping, activeRoomId, incrementUnread])

  const connect = useCallback(() => {
    if (!roomId || !accessToken) return
    if (ws.current?.readyState === WebSocket.OPEN) return

    // Токен передаём через query param — браузер не поддерживает
    // кастомные заголовки в WebSocket
    const socket = new WebSocket(
      `${WS_URL}/ws/rooms/${roomId}?token=${accessToken}`
    )
    ws.current = socket

    socket.onopen = () => {
      console.log(`[WS] connected to room ${roomId}`)
    }

    socket.onmessage = (event) => {
      try {
        const msg: WSEvent = JSON.parse(event.data)
        handleEvent(msg)
      } catch {
        console.error('[WS] failed to parse message', event.data)
      }
    }

    socket.onerror = (err) => {
      console.error('[WS] error', err)
    }

    socket.onclose = () => {
      console.log(`[WS] disconnected from room ${roomId}`)
      reconnectTimeout.current = setTimeout(connect, 3000)
    }
  }, [roomId, accessToken, handleEvent])

  const send = useCallback((type: string, payload: unknown) => {
    if (ws.current?.readyState !== WebSocket.OPEN) return
    ws.current.send(JSON.stringify({ type, payload }))
  }, [])

  const sendMessage = useCallback((content: string, replyToId?: string) => {
    send('send_message', { content, reply_to_id: replyToId })
  }, [send])

  const sendTyping = useCallback(() => {
    send('typing', {})
  }, [send])

  useEffect(() => {
    connect()
    return () => {
      if (reconnectTimeout.current) clearTimeout(reconnectTimeout.current)
      ws.current?.close()
      ws.current = null
    }
  }, [connect])

  return { sendMessage, sendTyping }
}