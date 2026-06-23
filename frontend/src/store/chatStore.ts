import { create } from 'zustand'
import type { Room, Message } from '../types'

interface ChatState {
  rooms: Room[]
  dms: Room[]
  activeRoomId: string | null
  messages: Record<string, Message[]> // roomId → messages
  unreadCounts: Record<string, number>
  typingUsers: Record<string, string[]> // roomId → usernames

  setRooms: (rooms: Room[]) => void
  setDMs: (dms: Room[]) => void
  setActiveRoom: (roomId: string | null) => void
  addRoom: (room: Room) => void

  setMessages: (roomId: string, messages: Message[]) => void
  prependMessages: (roomId: string, messages: Message[]) => void // для пагинации
  addMessage: (roomId: string, message: Message) => void
  updateMessage: (roomId: string, messageId: string, content: string, updatedAt: string) => void
  deleteMessage: (roomId: string, messageId: string) => void

  setUnreadCounts: (counts: Record<string, number>) => void
  incrementUnread: (roomId: string) => void
  clearUnread: (roomId: string) => void

  setTyping: (roomId: string, username: string) => void
  clearTyping: (roomId: string, username: string) => void
}

export const useChatStore = create<ChatState>((set) => ({
  rooms: [],
  dms: [],
  activeRoomId: null,
  messages: {},
  unreadCounts: {},
  typingUsers: {},

  setRooms: (rooms) => set({ rooms }),
  setDMs: (dms) => set({ dms }),
  setActiveRoom: (roomId) => set({ activeRoomId: roomId }),
  addRoom: (room) => set((s) => ({
    rooms: room.is_dm ? s.rooms : [room, ...s.rooms],
    dms: room.is_dm ? [room, ...s.dms] : s.dms,
  })),

  setMessages: (roomId, messages) =>
    set((s) => ({ messages: { ...s.messages, [roomId]: messages } })),

  prependMessages: (roomId, messages) =>
    set((s) => ({
      messages: {
        ...s.messages,
        [roomId]: [...messages, ...(s.messages[roomId] ?? [])],
      },
    })),

  addMessage: (roomId, message) =>
    set((s) => ({
      messages: {
        ...s.messages,
        [roomId]: [...(s.messages[roomId] ?? []), message],
      },
    })),

  updateMessage: (roomId, messageId, content, updatedAt) =>
    set((s) => ({
      messages: {
        ...s.messages,
        [roomId]: (s.messages[roomId] ?? []).map((m) =>
          m.id === messageId ? { ...m, content, updated_at: updatedAt } : m
        ),
      },
    })),

  deleteMessage: (roomId, messageId) =>
    set((s) => ({
      messages: {
        ...s.messages,
        [roomId]: (s.messages[roomId] ?? []).filter((m) => m.id !== messageId),
      },
    })),

  setUnreadCounts: (counts) => set({ unreadCounts: counts }),
  incrementUnread: (roomId) =>
    set((s) => ({
      unreadCounts: {
        ...s.unreadCounts,
        [roomId]: (s.unreadCounts[roomId] ?? 0) + 1,
      },
    })),
  clearUnread: (roomId) =>
    set((s) => ({
      unreadCounts: { ...s.unreadCounts, [roomId]: 0 },
    })),

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
}))