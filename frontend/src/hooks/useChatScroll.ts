import { useRef, useCallback, useState } from 'react'

const NEAR_BOTTOM_THRESHOLD = 80

export function useChatScroll() {
  const containerRef = useRef<HTMLDivElement>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const prevScrollHeight = useRef(0)
  const prevScrollTop = useRef(0)
  const anchorRef = useRef<{ id: string; top: number } | null>(null)
  const nearBottomRef = useRef(true)
  const [isAtBottom, setIsAtBottom] = useState(true)

  const isNearBottom = useCallback((): boolean => {
    const el = containerRef.current
    if (!el) return true
    return el.scrollHeight - el.scrollTop - el.clientHeight < NEAR_BOTTOM_THRESHOLD
  }, [])

  const updateNearBottom = useCallback(() => {
    const next = isNearBottom()
    nearBottomRef.current = next
    setIsAtBottom(next)
    return next
  }, [isNearBottom])

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'smooth') => {
    bottomRef.current?.scrollIntoView({ behavior })
    window.setTimeout(() => {
      nearBottomRef.current = true
      setIsAtBottom(true)
    }, behavior === 'smooth' ? 180 : 0)
  }, [])

  const shouldAutoScrollForNewMessage = useCallback(() => nearBottomRef.current || isNearBottom(), [isNearBottom])

  const saveScrollPosition = useCallback(() => {
    const container = containerRef.current
    prevScrollHeight.current = container?.scrollHeight ?? 0
    prevScrollTop.current = container?.scrollTop ?? 0

    if (!container) {
      anchorRef.current = null
      return
    }

    const containerTop = container.getBoundingClientRect().top
    const items = [...container.querySelectorAll<HTMLElement>('[data-message-id]')]
    const anchor = items.find((item) => item.getBoundingClientRect().bottom >= containerTop)

    anchorRef.current = anchor
      ? { id: anchor.dataset.messageId ?? '', top: anchor.getBoundingClientRect().top - containerTop }
      : null
  }, [])

  const restoreScrollPosition = useCallback(() => {
    const el = containerRef.current
    if (!el) return

    const anchor = anchorRef.current
    if (anchor?.id) {
      const item = el.querySelector<HTMLElement>(`[data-message-id="${CSS.escape(anchor.id)}"]`)
      if (item) {
        const containerTop = el.getBoundingClientRect().top
        const nextTop = item.getBoundingClientRect().top - containerTop
        el.scrollTop += nextTop - anchor.top
        anchorRef.current = null
        updateNearBottom()
        return
      }
    }

    const diff = el.scrollHeight - prevScrollHeight.current
    el.scrollTop = prevScrollTop.current + diff
    anchorRef.current = null
    updateNearBottom()
  }, [updateNearBottom])

  const trackScroll = useCallback(() => {
    return updateNearBottom()
  }, [updateNearBottom])

  const onInitialLoad = useCallback(() => {
    requestAnimationFrame(() => {
      scrollToBottom('auto')
      nearBottomRef.current = true
      setIsAtBottom(true)
    })
  }, [scrollToBottom])

  return {
    containerRef,
    bottomRef,
    isAtBottom,
    isNearBottom,
    scrollToBottom,
    shouldAutoScrollForNewMessage,
    saveScrollPosition,
    restoreScrollPosition,
    trackScroll,
    onInitialLoad,
  }
}
