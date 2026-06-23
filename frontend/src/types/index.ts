export interface User {
  id: string
  username: string
  email: string
  created_at: string
  updated_at: string
}

export interface AuthResponse {
  user: User
  access_token: string
}

export interface Room {
  id: string
  name: string
  description: string
  is_private: boolean
  is_dm: boolean
  owner_id: string
  created_at: string
}

export interface RoomMember {
  room_id: string
  user_id: string
  username: string
  role: 'owner' | 'admin' | 'member'
  joined_at: string
}

export interface Message {
  id: string
  room_id: string
  user_id: string
  username: string
  reply_to_id?: string
  content: string
  is_encrypted: boolean
  created_at: string
  updated_at: string
  deleted_at?: string
}

export interface MessagesResponse {
  messages: Message[]
  next_cursor?: number
  has_more: boolean
}

export type WSEventType =
  | 'send_message'
  | 'edit_message'
  | 'delete_message'
  | 'add_reaction'
  | 'remove_reaction'
  | 'typing'
  | 'new_message'
  | 'message_edited'
  | 'message_deleted'
  | 'reaction_added'
  | 'reaction_removed'
  | 'user_typing'
  | 'user_joined'
  | 'user_left'
  | 'error'

export interface WSEvent<T = unknown> {
  type: WSEventType
  payload: T
}

export interface NewMessagePayload {
  id: string
  room_id: string
  user_id: string
  username: string
  reply_to_id?: string
  content: string
  is_encrypted: boolean
  created_at: string
}

export interface MessageEditedPayload {
  message_id: string
  room_id: string
  content: string
  updated_at: string
}

export interface MessageDeletedPayload {
  message_id: string
  room_id: string
  deleted_at: string
}

export interface UserTypingPayload {
  room_id: string
  user_id: string
  username: string
}

export interface ErrorPayload {
  message: string
}