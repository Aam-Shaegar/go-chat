import { useState, useCallback, useRef, useEffect } from 'react'
import { useChatStore } from '../store/chatStore'
import { roomsApi } from '../api/rooms'

interface LoaderState {
  loading: boolean
  hasMore: boolean
  nextCursor?: string
  error: string | null
}

export function useChatLoader(roomId: string | null, onInitialLoad?: () => void) {
  const { setMessages, prependMessages, clearUnread } = useChatStore()
  const [state, setState] = useState<LoaderState>({
    loading: false,
    hasMore: false,
    nextCursor: undefined,
    error: null,
  })

  const requestId = useRef(0)

  useEffect(() => {
    const currentId = ++requestId.current
    if (!roomId) return

    void Promise.resolve().then(async () => {
      setState({
        loading: true,
        hasMore: false,
        nextCursor: undefined,
        error: null,
      })

      try {
        const { data } = await roomsApi.getMessages(roomId, undefined, 50)
        if (currentId !== requestId.current) return

        setMessages(roomId, data.messages ?? [])
        setState({
          loading: false,
          hasMore: data.has_more,
          nextCursor: data.next_cursor,
          error: null,
        })
        clearUnread(roomId)
        roomsApi.markRead(roomId).catch(() => {})
        onInitialLoad?.()
      } catch {
        if (currentId !== requestId.current) return
        setState({
          loading: false,
          hasMore: false,
          nextCursor: undefined,
          error: 'Could not load messages',
        })
      }
    })
  }, [roomId, setMessages, clearUnread, onInitialLoad])

  const loadMore = useCallback(async (savePosition: () => void, restorePosition: () => void) => {
    if (!roomId || !state.hasMore || state.loading) return
    const currentId = requestId.current
    setState((current) => ({ ...current, loading: true, error: null }))
    savePosition()

    try {
      const { data } = await roomsApi.getMessages(roomId, state.nextCursor, 50)
      if (currentId !== requestId.current) return

      prependMessages(roomId, data.messages ?? [])
      setState({
        loading: false,
        hasMore: data.has_more,
        nextCursor: data.next_cursor,
        error: null,
      })
      requestAnimationFrame(() => {
        requestAnimationFrame(restorePosition)
      })
    } catch {
      if (currentId !== requestId.current) return
      setState((current) => ({
        ...current,
        loading: false,
        error: 'Could not load earlier messages',
      }))
    } finally {
      if (currentId === requestId.current) {
        setState((current) => ({ ...current, loading: false }))
      }
    }
  }, [roomId, state.hasMore, state.loading, state.nextCursor, prependMessages])

  return { loading: state.loading, hasMore: state.hasMore, error: state.error, loadMore }
}
