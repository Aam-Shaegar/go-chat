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
  last_message_at?: string
  other_user_id?: string
}

export interface RoomMember {
  room_id: string
  user_id: string
  username: string
  role: 'owner' | 'admin' | 'member'
  joined_at: string
  muted_until?: string
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
  reactions?: MessageReaction[]
}

export interface MessagesResponse {
  messages: Message[]
  next_cursor?: string
  has_more: boolean
}

export interface RoomInvite {
  id: string
  room_id: string
  token: string
  created_by: string
  max_uses: number
  uses: number
  is_active: boolean
  expires_at?: string
  created_at: string
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
  | 'user_muted'
  | 'user_unmuted'
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

export interface UserJoinedPayload {
  room_id: string
  user_id: string
  username: string
}

export interface UserLeftPayload {
  room_id: string
  user_id: string
  username: string
  kicked_by?: string
  kicked_by_name?: string
}

export interface UserMutedPayload {
  room_id: string
  user_id: string
  username: string
  muted_until: string
  muted_by: string
  muted_by_name: string
}

export interface UserUnmutedPayload {
  room_id: string
  user_id: string
  username: string
  unmuted_by: string
  unmuted_by_name: string
}

export interface InviteDeactivatedPayload {
  room_id: string
  token: string
}

export interface InviteUsedPayload {
  room_id: string
  token: string
  uses: number
  max_uses: number
}

export interface ErrorPayload {
  message: string
}

export interface ReactionPayload {
  message_id: string
  room_id: string
  user_id: string
  emoji: string
  is_reacted_by_me: boolean
}

export interface MessageReaction {
  emoji: string
  count: number
  users: string[]
  isReactedByMe: boolean
}
