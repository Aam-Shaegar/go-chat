import { WebSocketManager, type ConnectionState } from './websocketManager'
import type { WSEvent } from '../types'

const WS_BASE = import.meta.env.VITE_WS_URL || 'ws://localhost:5050/api/v1'
const DISCONNECT_GRACE_MS = 800

interface RoomSocketCallbacks {
  onMessage: (roomId: string, event: WSEvent) => void
  onStateChange: (roomId: string, state: ConnectionState) => void
}

class RoomSocketHub {
  private manager = new WebSocketManager()
  private callbacks: RoomSocketCallbacks | null = null
  private releaseTimer: ReturnType<typeof setTimeout> | null = null
  private token = ''

  setCallbacks(callbacks: RoomSocketCallbacks | null) {
    this.callbacks = callbacks
  }

  sync(_roomIds: string[], token: string) {
    this.cancelRelease()
    if (this.token === token && this.manager.getState() !== 'disconnected') return
    this.token = token

    const url = `${WS_BASE}/ws?token=${encodeURIComponent(token)}`
    this.manager.connect(url, {
      onMessage: (event) => this.callbacks?.onMessage('', event),
      onStateChange: (state) => this.callbacks?.onStateChange('', state),
    })
  }

  send(roomId: string, type: string, payload: unknown) {
    this.manager.send(type, withRoomId(payload, roomId))
    return true
  }

  state(roomId: string): ConnectionState {
    void roomId
    return this.manager.getState()
  }

  releaseSoon() {
    this.cancelRelease()
    this.releaseTimer = setTimeout(() => this.disconnectAll(), DISCONNECT_GRACE_MS)
  }

  disconnectAll() {
    this.cancelRelease()
    this.token = ''
    this.manager.disconnect()
  }

  private cancelRelease() {
    if (!this.releaseTimer) return
    clearTimeout(this.releaseTimer)
    this.releaseTimer = null
  }
}

export const roomSocketHub = new RoomSocketHub()

function withRoomId(payload: unknown, roomId: string) {
  if (payload && typeof payload === 'object' && !Array.isArray(payload)) {
    return { ...payload, room_id: roomId }
  }
  return { room_id: roomId }
}
