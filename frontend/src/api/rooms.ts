import { client } from './client'
import type { Room, RoomMember, MessagesResponse } from '../types'

export const roomsApi = {
    acceptInvite: (token: string) =>
    client.post<Room>(`/invites/${token}/accept`),

  getPublic: (limit = 20, offset = 0) =>
    client.get<Room[]>('/rooms', { params: { limit, offset } }),

  getMy: () =>
    client.get<Room[]>('/rooms/my'),

  getById: (id: string) =>
    client.get<Room>(`/rooms/${id}`),

  create: (name: string, description: string, isPrivate: boolean) =>
    client.post<Room>('/rooms', { name, description, is_private: isPrivate }),

  join: (id: string) =>
    client.post(`/rooms/${id}/join`),

  leave: (id: string) =>
    client.post(`/rooms/${id}/leave`),

  getMembers: (id: string) =>
    client.get<RoomMember[]>(`/rooms/${id}/members`),

  getMessages: (id: string, before?: number, limit = 50) =>
    client.get<MessagesResponse>(`/rooms/${id}/messages`, {
      params: { before, limit },
    }),

  markRead: (id: string) =>
    client.post(`/rooms/${id}/read`),

  getUnread: (id: string) =>
    client.get<{ unread: number }>(`/rooms/${id}/unread`),
}

export const dmApi = {
  openDM: (userId: string) =>
    client.post<Room>(`/dm/${userId}`),

  getAll: () =>
    client.get<Room[]>('/dm'),
}

export const readsApi = {
  getAllUnread: () =>
    client.get<Record<string, number>>('/reads/unread'),
}