import { create } from 'zustand'
import type { Room, Message } from '../types'

export type RoomConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting'

interface ChatState {
  rooms: Room[]
  dms: Room[]
  dmNames: Record<string, string>
  activeRoomId: string | null
  messages: Record<string, Message[]>
  lastMessages: Record<string, Message>
  roomActivity: Record<string, string>
  unreadCounts: Record<string, number>
  typingUsers: Record<string, string[]>
  connectionStates: Record<string, RoomConnectionState>

  setRooms: (rooms: Room[]) => void
  setDMs: (dms: Room[]) => void
  setDMNames: (names: Record<string, string>) => void
  addDMName: (dmId: string, username: string) => void
  setActiveRoom: (roomId: string | null) => void
  addRoom: (room: Room) => void

  setMessages: (roomId: string, messages: Message[]) => void
  prependMessages: (roomId: string, messages: Message[]) => void
  addMessage: (roomId: string, message: Message) => boolean
  updateMessage: (roomId: string, messageId: string, content: string, updatedAt: string) => void
  deleteMessage: (roomId: string, messageId: string) => void
  touchRoom: (roomId: string, at: string) => void

  setUnreadCounts: (counts: Record<string, number>) => void
  incrementUnread: (roomId: string) => void
  clearUnread: (roomId: string) => void

  setTyping: (roomId: string, username: string) => void
  clearTyping: (roomId: string, username: string) => void
  clearRoomTyping: (roomId: string) => void
  setConnectionState: (roomId: string, state: RoomConnectionState) => void
  resetChat: () => void
}

const uniqueById = (rooms: Room[]) => {
  const seen = new Set<string>()
  return rooms.filter((room) => {
    if (seen.has(room.id)) return false
    seen.add(room.id)
    return true
  })
}

const compareMessages = (a: Message, b: Message) =>
  new Date(a.created_at).getTime() - new Date(b.created_at).getTime()

const mergeMessages = (existing: Message[], incoming: Message[]) => {
  const byId = new Map<string, Message>()
  for (const message of existing) byId.set(message.id, message)
  for (const message of incoming) byId.set(message.id, message)
  return [...byId.values()].sort(compareMessages)
}

const latestMessage = (messages: Message[]) => messages[messages.length - 1]

const seedActivity = (rooms: Room[], current: Record<string, string>) => {
  const next = { ...current }
  for (const room of rooms) {
    next[room.id] ??= room.last_message_at ?? room.created_at
  }
  return next
}

export const useChatStore = create<ChatState>((set) => ({
  rooms: [],
  dms: [],
  dmNames: {},
  activeRoomId: null,
  messages: {},
  lastMessages: {},
  roomActivity: {},
  unreadCounts: {},
  typingUsers: {},
  connectionStates: {},

  setRooms: (rooms) =>
    set((s) => {
      const nextRooms = uniqueById(rooms)
      return {
        rooms: nextRooms,
        roomActivity: seedActivity(nextRooms, s.roomActivity),
      }
    }),
  setDMs: (dms) =>
    set((s) => {
      const nextDms = uniqueById(dms)
      return {
        dms: nextDms,
        roomActivity: seedActivity(nextDms, s.roomActivity),
      }
    }),
  setDMNames: (names) => set({ dmNames: names }),
  addDMName: (dmId, username) =>
    set((s) => ({ dmNames: { ...s.dmNames, [dmId]: username } })),

  setActiveRoom: (roomId) => set({ activeRoomId: roomId }),
  addRoom: (room) => set((s) => ({
    rooms: room.is_dm ? s.rooms : uniqueById([room, ...s.rooms]),
    dms: room.is_dm ? uniqueById([room, ...s.dms]) : s.dms,
    roomActivity: {
      ...s.roomActivity,
      [room.id]: room.last_message_at ?? room.created_at,
    },
  })),

  setMessages: (roomId, messages) =>
    set((s) => {
      const merged = mergeMessages(s.messages[roomId] ?? [], messages)
      const last = latestMessage(merged)
      return {
        messages: { ...s.messages, [roomId]: merged },
        lastMessages: last ? { ...s.lastMessages, [roomId]: last } : s.lastMessages,
        roomActivity: last
          ? { ...s.roomActivity, [roomId]: last.created_at }
          : s.roomActivity,
      }
    }),

  prependMessages: (roomId, messages) =>
    set((s) => {
      const merged = mergeMessages(s.messages[roomId] ?? [], messages)
      const last = latestMessage(merged)
      return {
        messages: { ...s.messages, [roomId]: merged },
        lastMessages: last ? { ...s.lastMessages, [roomId]: last } : s.lastMessages,
        roomActivity: last
          ? { ...s.roomActivity, [roomId]: last.created_at }
          : s.roomActivity,
      }
    }),

  addMessage: (roomId, message) =>
    {
      let added = false
      set((s) => {
      const existing = s.messages[roomId] ?? []
      if (existing.some((m) => m.id === message.id)) return s
      added = true
      const merged = mergeMessages(existing, [message])
      return {
        messages: { ...s.messages, [roomId]: merged },
        lastMessages: { ...s.lastMessages, [roomId]: latestMessage(merged) },
        roomActivity: { ...s.roomActivity, [roomId]: message.created_at },
      }
      })
      return added
    },

  updateMessage: (roomId, messageId, content, updatedAt) =>
    set((s) => {
      const messages = (s.messages[roomId] ?? []).map((m) =>
        m.id === messageId ? { ...m, content, updated_at: updatedAt } : m
      )
      const currentLast = s.lastMessages[roomId]
      const lastMessages = currentLast?.id === messageId
        ? { ...s.lastMessages, [roomId]: { ...currentLast, content, updated_at: updatedAt } }
        : s.lastMessages

      return {
        messages: { ...s.messages, [roomId]: messages },
        lastMessages,
      }
    }),

  deleteMessage: (roomId, messageId) =>
    set((s) => {
      const messages = (s.messages[roomId] ?? []).filter((m) => m.id !== messageId)
      const last = latestMessage(messages)
      const lastMessages = { ...s.lastMessages }

      if (last) {
        lastMessages[roomId] = last
      } else {
        delete lastMessages[roomId]
      }

      return {
        messages: { ...s.messages, [roomId]: messages },
        lastMessages,
      }
    }),

  touchRoom: (roomId, at) =>
    set((s) => ({
      roomActivity: { ...s.roomActivity, [roomId]: at },
    })),

  setUnreadCounts: (counts) => set({ unreadCounts: counts }),
  incrementUnread: (roomId) =>
    set((s) => ({
      unreadCounts: { ...s.unreadCounts, [roomId]: (s.unreadCounts[roomId] ?? 0) + 1 },
    })),
  clearUnread: (roomId) =>
    set((s) => ({ unreadCounts: { ...s.unreadCounts, [roomId]: 0 } })),

  setTyping: (roomId, username) =>
    set((s) => ({
      typingUsers: {
        ...s.typingUsers,
        [roomId]: [...new Set([...(s.typingUsers[roomId] ?? []), username])],
      },
    })),

  clearTyping: (roomId, username) =>
    set((s) => ({
      typingUsers: {
        ...s.typingUsers,
        [roomId]: (s.typingUsers[roomId] ?? []).filter((u) => u !== username),
      },
    })),

  clearRoomTyping: (roomId) =>
    set((s) => {
      const typingUsers = { ...s.typingUsers }
      delete typingUsers[roomId]
      return { typingUsers }
    }),

  setConnectionState: (roomId, state) =>
    set((s) => ({
      connectionStates: { ...s.connectionStates, [roomId]: state },
    })),

  resetChat: () => set({
    rooms: [],
    dms: [],
    dmNames: {},
    activeRoomId: null,
    messages: {},
    lastMessages: {},
    roomActivity: {},
    unreadCounts: {},
    typingUsers: {},
    connectionStates: {},
  }),
}))
