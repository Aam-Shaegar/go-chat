import type { WSEvent } from '../types'

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting'

interface WSManagerCallbacks {
  onOpen?: () => void
  onClose?: () => void
  onError?: (err: Event) => void
  onMessage?: (event: WSEvent) => void
  onReconnect?: () => void
  onStateChange?: (state: ConnectionState) => void
}

const RECONNECT_BASE_MS = 1000
const RECONNECT_MAX_MS = 30000

export class WebSocketManager {
  private socket: WebSocket | null = null
  private queue: string[] = []
  private callbacks: WSManagerCallbacks = {}
  private reconnectAttempt = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private shouldReconnect = false
  private url = ''
  private state: ConnectionState = 'disconnected'

  connect(url: string, callbacks: WSManagerCallbacks) {
    const isSameSocket = this.url === url
      && (this.socket?.readyState === WebSocket.OPEN || this.socket?.readyState === WebSocket.CONNECTING)

    this.url = url
    this.callbacks = callbacks
    this.shouldReconnect = true

    if (isSameSocket) return

    this._clearTimers()
    if (this.socket) {
      const previousSocket = this.socket
      previousSocket.onclose = null
      previousSocket.close(1000, 'reconnect to another room')
      this.socket = null
    }

    this._connect()
  }

  disconnect() {
    this.shouldReconnect = false
    this._clearTimers()
    this.queue = []
    if (this.socket) {
      this.socket.close(1000, 'intentional disconnect')
      this.socket = null
    }
    this._setState('disconnected')
  }

  send(type: string, payload: unknown) {
    const msg = JSON.stringify({ type, payload })
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(msg)
    } else {
      this.queue.push(msg)
    }
  }

  getState(): ConnectionState {
    return this.state
  }

  private _connect() {
    if (this.socket?.readyState === WebSocket.OPEN) return

    this._setState(this.reconnectAttempt === 0 ? 'connecting' : 'reconnecting')
    const isReconnect = this.reconnectAttempt > 0

    this.socket = new WebSocket(this.url)

    this.socket.onopen = () => {
      this.reconnectAttempt = 0
      this._setState('connected')
      this._flushQueue()
      this.callbacks.onOpen?.()
      if (isReconnect) {
        this.callbacks.onReconnect?.()
      }
    }

    this.socket.onmessage = (e) => {
      try {
        const event: WSEvent = JSON.parse(e.data)
        this.callbacks.onMessage?.(event)
      } catch {
        console.error('[WS] failed to parse message', e.data)
      }
    }

    this.socket.onerror = (err) => {
      this.callbacks.onError?.(err)
    }

    this.socket.onclose = () => {
      this.callbacks.onClose?.()
      if (this.shouldReconnect) {
        this._scheduleReconnect()
      } else {
        this._setState('disconnected')
      }
    }
  }

  private _flushQueue() {
    while (this.queue.length > 0 && this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(this.queue.shift()!)
    }
  }

  private _scheduleReconnect() {
    if (this.reconnectTimer) return

    const delay = Math.min(
      RECONNECT_BASE_MS * Math.pow(2, this.reconnectAttempt),
      RECONNECT_MAX_MS
    )
    this.reconnectAttempt++
    this._setState('reconnecting')
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this._connect()
    }, delay)
  }

  private _clearTimers() {
    if (this.reconnectTimer) { clearTimeout(this.reconnectTimer); this.reconnectTimer = null }
  }

  private _setState(state: ConnectionState) {
    this.state = state
    this.callbacks.onStateChange?.(state)
  }
}
