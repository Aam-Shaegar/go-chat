import { useCallback } from 'react'
import type { ConnectionState } from '../services/websocketManager'
import { roomSocketHub } from '../services/roomSocketHub.ts'
import { useChatStore } from '../store/chatStore'

export function useWebSocket(roomId: string | null) {
  const connectionState = useChatStore((state) =>
    roomId ? state.connectionStates[roomId] ?? roomSocketHub.state(roomId) : 'disconnected'
  ) as ConnectionState

  const sendMessage = useCallback((content: string, replyToId?: string) => {
    if (!roomId) return false
    return roomSocketHub.send(roomId, 'send_message', { content, reply_to_id: replyToId })
  }, [roomId])

  const sendTyping = useCallback(() => {
    if (!roomId) return false
    return roomSocketHub.send(roomId, 'typing', {})
  }, [roomId])

  return { sendMessage, sendTyping, connectionState }
}
