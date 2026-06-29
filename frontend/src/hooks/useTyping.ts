import { useRef, useCallback } from 'react'

const TYPING_THROTTLE_MS = 1200

export function useTyping(roomId: string | null, sendTypingWS: () => void) {
  const lastSentAt = useRef(0)

  const onInputChange = useCallback(() => {
    if (!roomId) return
    const now = Date.now()
    if (now - lastSentAt.current < TYPING_THROTTLE_MS) return
    lastSentAt.current = now
    sendTypingWS()
  }, [roomId, sendTypingWS])

  const cleanup = useCallback(() => {
    lastSentAt.current = 0
  }, [])

  return { onInputChange, cleanup }
}
