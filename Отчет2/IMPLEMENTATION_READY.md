# ✅ IMPLEMENTATION CHECKLIST - Ready to Build

**Статус:** Готов к выполнению  
**Total Effort:** 60-70 часов  
**Timeline:** 2 недели (интенсив)

---

## 🔴 КРИТИЧЕСКИЕ ЗАДАЧИ (Do First!)

### TASK 1: DATABASE & MIGRATIONS (1 день, 4 часа)

```
❌ [ ] Create migrations/000006_create_rooms.up.sql
❌ [ ] Create migrations/000006_create_rooms.down.sql
❌ [ ] Create migrations/000007_create_direct_messages.up.sql
❌ [ ] Create migrations/000007_create_direct_messages.down.sql

Database Tables Needed:
────────────────────────
rooms:
  id UUID PRIMARY KEY
  name VARCHAR(255) NOT NULL
  description TEXT
  is_private BOOLEAN DEFAULT FALSE
  owner_id UUID NOT NULL REFERENCES users(id)
  created_at TIMESTAMP DEFAULT NOW()
  updated_at TIMESTAMP DEFAULT NOW()

room_members:
  id UUID PRIMARY KEY
  room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
  role VARCHAR(50) DEFAULT 'member' -- 'owner', 'moderator', 'member'
  joined_at TIMESTAMP DEFAULT NOW()
  UNIQUE(room_id, user_id)

messages:
  id UUID PRIMARY KEY
  room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE
  user_id UUID NOT NULL REFERENCES users(id)
  content TEXT NOT NULL
  created_at TIMESTAMP DEFAULT NOW()
  edited_at TIMESTAMP NULL
  deleted_at TIMESTAMP NULL

direct_messages:
  id UUID PRIMARY KEY
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
  to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
  content TEXT NOT NULL
  read_at TIMESTAMP NULL
  created_at TIMESTAMP DEFAULT NOW()
  edited_at TIMESTAMP NULL
  deleted_at TIMESTAMP NULL

user_blocks:
  id UUID PRIMARY KEY
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
  blocked_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
  created_at TIMESTAMP DEFAULT NOW()
  UNIQUE(user_id, blocked_user_id)

Indices:
  CREATE INDEX idx_rooms_owner_id ON rooms(owner_id);
  CREATE INDEX idx_rooms_is_private ON rooms(is_private);
  CREATE INDEX idx_room_members_room_id ON room_members(room_id);
  CREATE INDEX idx_room_members_user_id ON room_members(user_id);
  CREATE INDEX idx_messages_room_id ON messages(room_id);
  CREATE INDEX idx_messages_user_id ON messages(user_id);
  CREATE INDEX idx_messages_created_at ON messages(created_at);
  CREATE INDEX idx_direct_messages_from_to ON direct_messages(from_user_id, to_user_id);
  CREATE INDEX idx_direct_messages_to ON direct_messages(to_user_id);
  CREATE INDEX idx_direct_messages_created ON direct_messages(created_at);
```

---

### TASK 2: ROOMS FEATURE (3-4 дня, 50 часов)

```
Day 1: Repository Layer (8 часов)
─────────────────────────────────

❌ [ ] Create: internal/features/rooms/repository/postgres/models.go
       - Room struct
       - RoomMember struct
       - Message struct

❌ [ ] Create: internal/features/rooms/repository/postgres/room_repository.go
       Methods:
       ❌ CreateRoom(ctx, room *Room) (*Room, error)
       ❌ GetRoom(ctx, roomID string) (*Room, error)
       ❌ GetPublicRooms(ctx, limit, offset int) ([]Room, error)
       ❌ GetUserRooms(ctx, userID string) ([]Room, error)
       ❌ UpdateRoom(ctx, roomID, name, desc string, isPrivate bool) error
       ❌ DeleteRoom(ctx, roomID string) error
       ❌ AddMember(ctx, roomID, userID string, role string) error
       ❌ RemoveMember(ctx, roomID, userID string) error
       ❌ GetMembers(ctx, roomID string) ([]RoomMember, error)
       ❌ IsMember(ctx, roomID, userID string) (bool, error)

❌ [ ] Create: internal/features/rooms/repository/postgres/message_repository.go
       Methods:
       ❌ SaveMessage(ctx, msg *Message) (*Message, error)
       ❌ GetMessages(ctx, roomID string, limit, offset int) ([]Message, error)
       ❌ EditMessage(ctx, msgID, content string) error
       ❌ DeleteMessage(ctx, msgID string) error
       ❌ GetMessage(ctx, msgID string) (*Message, error)

❌ [ ] Create: internal/features/rooms/repository/postgres/repository.go
       - New constructor
       - Dependency injection


Day 2: Service Layer (8 часов)
─────────────────────────────

❌ [ ] Create: internal/features/rooms/service/room_service.go
       Methods with validation + permission checks:
       ❌ CreateRoom(ctx, req *CreateRoomRequest) (*Room, error)
       ❌ GetRoom(ctx, roomID string) (*Room, error)
       ❌ ListPublic(ctx, limit, offset int) ([]Room, error)
       ❌ ListMyRooms(ctx, userID string) ([]Room, error)
       ❌ UpdateRoom(ctx, roomID, userID string, req *UpdateRoomRequest) error
       ❌ DeleteRoom(ctx, roomID, userID string) error  // only owner
       ❌ JoinRoom(ctx, roomID, userID string) error
       ❌ LeaveRoom(ctx, roomID, userID string) error
       ❌ KickMember(ctx, roomID, memberID, userID string) error  // only owner/moderator
       ❌ GetMembers(ctx, roomID string) ([]RoomMember, error)

❌ [ ] Create: internal/features/rooms/service/message_service.go
       Methods:
       ❌ SendMessage(ctx, roomID, userID string, req *SendMessageRequest) (*Message, error)
       ❌ GetMessages(ctx, roomID string, limit, offset int) ([]Message, error)
       ❌ EditMessage(ctx, msgID, userID, content string) error  // only author
       ❌ DeleteMessage(ctx, msgID, userID string) error  // only author/owner

❌ [ ] Create: internal/features/rooms/service/validation.go
       - ValidateRoomName(name string) error
       - ValidateRoomDescription(desc string) error
       - ValidateMessageContent(content string) error

❌ [ ] Create: internal/features/rooms/service/service.go
       - Service interface definition
       - Constructor


Day 3: Transport Layer (8 часов)
────────────────────────────────

❌ [ ] Create: internal/features/rooms/transport/http/dto.go
       Request/Response structs:
       - CreateRoomRequest
       - UpdateRoomRequest
       - SendMessageRequest
       - RoomResponse
       - MessageResponse

❌ [ ] Create: internal/features/rooms/transport/http/handler.go
       HTTP Handlers:
       ❌ POST /rooms (Create)
       ❌ GET /rooms/{id} (Get)
       ❌ GET /rooms/public (List Public)
       ❌ GET /me/rooms (List My Rooms)
       ❌ PUT /rooms/{id} (Update)
       ❌ DELETE /rooms/{id} (Delete)
       ❌ POST /rooms/{id}/join (Join)
       ❌ POST /rooms/{id}/leave (Leave)
       ❌ DELETE /rooms/{id}/members/{userId} (Kick)
       ❌ GET /rooms/{id}/members (Get Members)
       ❌ POST /rooms/{id}/messages (Send Message)
       ❌ GET /rooms/{id}/messages (Get Messages)
       ❌ PUT /messages/{id} (Edit Message)
       ❌ DELETE /messages/{id} (Delete Message)

❌ [ ] Create: internal/features/rooms/transport/http/transport.go
       - Transport interface
       - Routes definition


Day 4: WebSocket Integration (10 часов)
──────────────────────────────────────

❌ [ ] Create: internal/ws/room_hub.go
       Methods:
       ❌ SubscribeRoom(userID, roomID)
       ❌ UnsubscribeRoom(userID, roomID)
       ❌ BroadcastRoomMessage(roomID, message)
       ❌ HandleRoomConnection(roomID, conn)
       ❌ HandleRoomDisconnect(roomID, userID)

❌ [ ] Create: internal/handler/room_ws_handler.go
       WebSocket endpoint:
       ❌ GET /ws/room/{roomId}

❌ [ ] Update: internal/handler/middleware.go
       - Add auth middleware to WebSocket


Day 5: Integration & Testing (8 часов)
──────────────────────────────────────

❌ [ ] Update: cmd/server/main.go
       - Register room routes
       - Register room WebSocket handler

❌ [ ] Test CRUD operations
❌ [ ] Test permissions (owner can kick, member cannot)
❌ [ ] Test join/leave
❌ [ ] Test WebSocket messaging
❌ [ ] Test message history retrieval
```

---

### TASK 3: DIRECT MESSAGES (2-3 дня, 35 часов)

```
Day 1: Repository Layer (6 часов)
─────────────────────────────────

❌ [ ] Create: internal/features/direct_messages/repository/postgres/models.go
       - DirectMessage struct
       - UserBlock struct
       - Conversation struct

❌ [ ] Create: internal/features/direct_messages/repository/postgres/dm_repository.go
       Methods:
       ❌ SendMessage(ctx, fromID, toID, content string) (*DirectMessage, error)
       ❌ GetHistory(ctx, user1ID, user2ID string, limit, offset int) ([]DirectMessage, error)
       ❌ GetConversations(ctx, userID string) ([]Conversation, error)
       ❌ MarkAsRead(ctx, userID, otherUserID string, readAt time.Time) error
       ❌ EditMessage(ctx, msgID, content string) error
       ❌ DeleteMessage(ctx, msgID string) error
       ❌ GetMessage(ctx, msgID string) (*DirectMessage, error)
       ❌ GetUnreadCount(ctx, userID, otherUserID string) (int, error)

❌ [ ] Create: internal/features/direct_messages/repository/postgres/block_repository.go
       Methods:
       ❌ BlockUser(ctx, userID, blockedID string) error
       ❌ UnblockUser(ctx, userID, blockedID string) error
       ❌ IsBlocked(ctx, userID, blockedID string) (bool, error)
       ❌ GetBlockedUsers(ctx, userID string) ([]User, error)


Day 2: Service Layer (6 часов)
──────────────────────────────

❌ [ ] Create: internal/features/direct_messages/service/dm_service.go
       Methods with block checking:
       ❌ SendMessage(ctx, fromID, toID string, req *SendMessageRequest) (*DirectMessage, error)
       ❌ GetHistory(ctx, userID, otherID string, limit, offset int) ([]DirectMessage, error)
       ❌ GetConversations(ctx, userID string) ([]Conversation, error)
       ❌ MarkAsRead(ctx, userID, otherID string) error
       ❌ EditMessage(ctx, msgID, userID, content string) error
       ❌ DeleteMessage(ctx, msgID, userID string) error

❌ [ ] Create: internal/features/direct_messages/service/block_service.go
       Methods:
       ❌ BlockUser(ctx, userID, blockedID string) error
       ❌ UnblockUser(ctx, userID, blockedID string) error
       ❌ GetBlockedUsers(ctx, userID string) ([]User, error)

❌ [ ] Create: internal/features/direct_messages/service/validation.go


Day 3: Transport & WebSocket (8 часов)
──────────────────────────────────────

❌ [ ] Create: internal/features/direct_messages/transport/http/dto.go
       - SendMessageRequest
       - EditMessageRequest
       - DirectMessageResponse
       - ConversationResponse

❌ [ ] Create: internal/features/direct_messages/transport/http/handler.go
       HTTP Handlers:
       ❌ POST /dm/send (Send)
       ❌ GET /dm/history/{userId} (Get History)
       ❌ GET /dm/conversations (Get Conversations)
       ❌ PUT /dm/{msgId}/read (Mark as Read)
       ❌ PUT /dm/{msgId} (Edit)
       ❌ DELETE /dm/{msgId} (Delete)
       ❌ POST /dm/block/{userId} (Block)
       ❌ DELETE /dm/block/{userId} (Unblock)
       ❌ GET /dm/blocks (Get Blocked Users)

❌ [ ] Create: internal/ws/dm_hub.go
       Methods:
       ❌ Subscribe(userID)
       ❌ Unsubscribe(userID)
       ❌ BroadcastMessage(fromID, toID, message)

❌ [ ] Create: internal/handler/dm_ws_handler.go
       WebSocket endpoint:
       ❌ GET /ws/dm

❌ [ ] Update: cmd/server/main.go
       - Register DM routes
       - Register DM WebSocket handler
```

---

### TASK 4: SECURITY FIXES (1 день, 8 часов)

```
❌ [ ] Create: internal/middleware/rate_limit.go
       - RateLimit middleware
       - Config: requests per time window
       - Apply to:
         - POST /auth/register (5/hour/IP)
         - POST /auth/login (10/hour/IP)
         - POST /jwt/refresh (60/hour/user)
         - POST /dm/send (100/hour/user)

❌ [ ] Update: internal/features/jwt/service/hash.go
       - Улучшить token hashing (добавить salt или bcrypt)
       - Документировать решение

❌ [ ] Create: internal/core/domain/validation.go
       - ValidateEmail(email string) bool
       - ValidateUsername(username string) bool
       - ValidatePassword(password string) bool

❌ [ ] Create: internal/core/logger/audit.go
       - AuditLog struct
       - LogUserRegister(userID, email)
       - LogUserLogin(userID, email)
       - LogMessageSend(userID, roomID/recipientID)
       - LogRoomCreate(userID, roomID)
```

---

### TASK 5: USER ENHANCEMENTS (1 день, 10 часов)

```
❌ [ ] Add to User model: Bio, Avatar, Status
       File: internal/core/domain/models/user.go

❌ [ ] Create: internal/features/users/repository/postgres/search_user.go
       Methods:
       ❌ SearchByUsername(ctx, query string, limit int) ([]User, error)
       ❌ SearchByEmail(ctx, query string, limit int) ([]User, error)

❌ [ ] Create: internal/features/users/service/edit_profile.go
       Methods:
       ❌ UpdateUsername(ctx, userID, newUsername string) error
       ❌ UpdateBio(ctx, userID, bio string) error
       ❌ UpdateStatus(ctx, userID, status string) error

❌ [ ] Create: internal/features/users/service/change_password.go
       Methods:
       ❌ ChangePassword(ctx, userID, oldPassword, newPassword string) error

❌ [ ] Update: internal/features/users/transport/http/handler.go
       New endpoints:
       ❌ GET /users/search?q=... (Search)
       ❌ PUT /users/{id}/profile (Edit profile)
       ❌ POST /users/change-password (Change password)
       ❌ GET /users/{id}/status (Get status)
       ❌ PUT /users/{id}/status (Set status)
```

---

### TASK 6: TESTING (2 дня, 20 часов)

```
❌ [ ] Create: internal/features/rooms/service/room_service_test.go
       - Test CreateRoom (valid/invalid)
       - Test permissions (owner can delete, member cannot)
       - Test join/leave logic
       - Test member operations

❌ [ ] Create: internal/features/direct_messages/service/dm_service_test.go
       - Test SendMessage
       - Test block checking
       - Test history retrieval
       - Test unread count

❌ [ ] Create: internal/features/users/service/user_service_test.go
       - Test register (duplicate email/username)
       - Test login (invalid password)
       - Test search

❌ [ ] Create: cmd/tests/integration_test.go
       - Test end-to-end flow:
         1. Register users
         2. Create room
         3. Join room
         4. Send message
         5. Verify history
         6. Leave room

❌ [ ] Load testing WebSocket
       - 100 concurrent connections
       - 1000 messages/second
       - Measure latency

❌ [ ] Test database migrations
       - Run all migrations
       - Verify schema
       - Test rollback
```

---

## 🗺️ VISUAL DEPENDENCY GRAPH

```
                    ┌─────────────────────┐
                    │   Users Feature     │
                    │  (Register/Login)   │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │   JWT Feature       │
                    │ (Tokens/Auth)       │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
      ┌───────▼────────┐   ┌───▼────────────┐  │
      │  Rooms Feature │   │  DM Feature    │  │
      │ (Chat/Groups)  │   │ (1-on-1 Chat)  │  │
      └───────┬────────┘   └───┬────────────┘  │
              │                │                │
              └────────┬───────┘                │
                       │                        │
              ┌────────▼──────────────┐        │
              │  WebSocket Hub       │◄───────┘
              │ (Real-time Messaging)│
              └──────────────────────┘
```

---

## 🚀 QUICK START

```bash
# 1. Setup database
migrate -path ./migrations -database "postgres://..." up

# 2. Generate migrations
migrate create -ext sql -dir ./migrations -seq create_rooms

# 3. Create directory structure
mkdir -p internal/features/rooms/{repository/postgres,service,transport/http}
mkdir -p internal/features/direct_messages/{repository/postgres,service,transport/http}

# 4. Build & Run
go build ./cmd/server
./server

# 5. Test endpoints
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com","password":"pass123"}'
```

---

**Ready to build? Let's go! 🚀**

