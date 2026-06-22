# 🚀 Quick Action Checklist для GoChat Features

> **Статус:** 39% готовности к production  
> **Приоритет:** 5 критических + 12 серьёзных + 15 средних = 32 проблемы  
> **Время:** ~2-3 недели для полного fix

---

## 🔴 ДЕНЬ 1: КРИТИЧЕСКИЕ ИСПРАВЛЕНИЯ

### Task 1.1: Исправить миграцию БД
**Файл:** `migrations/000001_init_up.sql`  
**Строка:** 11

```sql
-- ❌ БЫЛО
user_id INT REFERENCES gochat.users(id) ON DELETE CASCADE,

-- ✅ СТАЛО
user_id UUID REFERENCES gochat.users(id) ON DELETE CASCADE,
```

**Проверка:** 
- [ ] Миграция применяется без errors
- [ ] Refresh tokens таблица создалась
- [ ] Foreign key работает

**Команда запуска:**
```bash
# Откатить миграцию
migrate -path migrations -database "postgres://..." down 1

# Применить исправленную версию
migrate -path migrations -database "postgres://..." up 1
```

---

### Task 1.2: Исправить rooms transport handler
**Файл:** `features/rooms/transport/http/transport.go`

```go
// ❌ БЫЛО
func NewUsersHTTPHandler(roomsService RoomsService, cfg *core_config.Config) *RoomsHTTPHandler {
	return &RoomsHTTPHandler{
		roomsService: roomsService,
		cfg:          cfg,
	}
}

func (h *RoomsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{},
	}
}

// ✅ СТАЛО
func NewRoomsHTTPHandler(roomsService RoomsService, cfg *core_config.Config) *RoomsHTTPHandler {
	return &RoomsHTTPHandler{
		roomsService: roomsService,
		cfg:          cfg,
	}
}

func (h *RoomsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  http.MethodGet,
			Path:    "/rooms",
			Handler: h.GetPublicRooms,
		},
		{
			Method:  http.MethodGet,
			Path:    "/rooms/my",
			Handler: h.GetMyRooms,
		},
		{
			Method:  http.MethodPost,
			Path:    "/rooms",
			Handler: h.CreateRoom,
		},
		// ... остальные routes
	}
}
```

**Проверка:**
- [ ] Конструктор имеет правильное имя
- [ ] Routes() не пустой
- [ ] Все методы interface имеют route

---

### Task 1.3: Переименовать direct_mesages → direct_messages
**Команда:**
```bash
cd internal/features
mv direct_mesages direct_messages
```

**Проверка:**
- [ ] Папка переименована
- [ ] Импорты обновлены
- [ ] Приложение стартует

---

### Task 1.4: Исправить type inconsistencies в rooms
**Файл:** `features/rooms/transport/http/transport.go`

```go
// ❌ БЫЛО
type RoomsService interface {
	GetMyRooms(ctx context.Context, userID int) (...) error
	GetRoomByID(ctx context.Context, roomID int) (...) error
	DeleteRoom(ctx context.Context, roomID int) error
	Leave(ctx context.Context, userID, roomID int) error
}

// ✅ СТАЛО (все ID - UUID)
type RoomsService interface {
	GetMyRooms(ctx context.Context, userID string) (...) error
	GetRoomByID(ctx context.Context, roomID string) (...) error
	DeleteRoom(ctx context.Context, roomID string) error
	Leave(ctx context.Context, userID, roomID string) error
}
```

**Проверка:**
- [ ] Все int ID изменены на string
- [ ] Domain models используют UUID
- [ ] Migrations используют UUID

---

## 🟠 ДЕНЬ 2: СЕРЬЁЗНЫЕ АРХИТЕКТУРНЫЕ ИСПРАВЛЕНИЯ

### Task 2.1: Исправить parameter typos в users
**Файл:** `features/users/transport/http/transport.go`

```go
// ❌ БЫЛО
GetUsers(ctx context.Context, limit, offser *int) ([]domain_models.User, error)
Register(ctx context.Context, username, email, pasword string) (...)

// ✅ СТАЛО
GetUsers(ctx context.Context, limit, offset *int) ([]domain_models.User, error)
Register(ctx context.Context, username, email, password string) (...)
```

**Проверка:**
- [ ] Все параметры правильно названы
- [ ] Компилируется без ошибок

---

### Task 2.2: Добавить race condition fix в register
**Файл:** `features/users/service/register.go`

```go
// Вместо отдельных проверок, использовать database constraint
func (s *UsersService) Register(ctx context.Context, username, email, password string) (...) error {
	// Валидация только в service
	if username == "" || email == "" || password == "" {
		return fmt.Errorf("username, email, password are required: %w", core_error.ErrInvlalidArgument)
	}
	
	// Хеширование
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	
	user := domain_models.NewUser("", username, email, string(hashed), time.Now(), time.Now())
	
	// БД гарантирует uniqueness
	createdUser, err := s.usersRepository.CreateUser(ctx, user)
	if err != nil {
		// Проверить на constraint violation
		if errors.Is(err, ...) {
			return fmt.Errorf("email already taken: %w", core_error.ErrInvlalidArgument)
		}
		return fmt.Errorf("create user: %w", err)
	}
	
	return s.authService.IssueTokens(ctx, createdUser)
}
```

**Миграция для БД:**
```sql
-- Убедиться что indices существуют
CREATE UNIQUE INDEX idx_users_email ON gochat.users(email);
CREATE UNIQUE INDEX idx_users_username ON gochat.users(username);
```

**Проверка:**
- [ ] Race condition test пройден
- [ ] Constraint violation обработан правильно

---

### Task 2.3: Добавить email validation
**Файл:** `features/users/transport/http/register.go`

```go
import (
	"regexp"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (h *UsersHTTPHandler) Register(w http.ResponseWriter, r *http.Request) {
	// ...
	
	if !emailRegex.MatchString(req.Email) {
		responseHandler.ErrorResponse(
			fmt.Errorf("invalid email format"),
			"email format is invalid",
		)
		return
	}
	
	// ... rest of handler
}
```

**Проверка:**
- [ ] Invalid emails отклоняются
- [ ] Valid emails принимаются
- [ ] Error response правильный

---

### Task 2.4: Переместить interfaces в service layer
**Файл:** `features/users/service/interfaces.go` (новый файл)

```go
package users_service

import (
	"context"
	domain_dtos "go-chat/internal/core/domain/dtos"
	domain_models "go-chat/internal/core/domain/models"
)

// Переместить из transport/http/transport.go
type UsersRepository interface {
	GetUsers(ctx context.Context, limit, offset *int) ([]domain_models.User, error)
	GetUser(ctx context.Context, userID string) (domain_models.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain_models.User, error)
	UserExistsByEmail(ctx context.Context, email string) (bool, error)
	UserExistsByUsername(ctx context.Context, username string) (bool, error)
	CreateUser(ctx context.Context, user domain_models.User) (domain_models.User, error)
}

type AuthService interface {
	IssueTokens(ctx context.Context, user domain_models.User) (domain_dtos.AuthResponseDTO, string, error)
}
```

**Обновить transport:**
```go
// features/users/transport/http/transport.go
package users_transport_http

import (
	users_service "go-chat/internal/features/users/service"
)

type UsersHTTPHandler struct {
	usersService users_service.UsersService
	cfg          *core_config.Config
}

type UsersService interface {
	GetUsers(ctx context.Context, limit, offset *int) ([]domain_models.User, error)
	GetUser(ctx context.Context, userID string) (domain_models.User, error)
	Register(ctx context.Context, username, email, password string) (domain_dtos.AuthResponseDTO, string, error)
	Login(ctx context.Context, email, password string) (domain_dtos.AuthResponseDTO, string, error)
}
```

**Проверка:**
- [ ] Interfaces в правильном месте
- [ ] Нет circular imports
- [ ] Компилируется

---

### Task 2.5: Добавить structured logging
**Файл:** `features/users/service/register.go`

```go
import (
	"context"
	core_logger "go-chat/internal/core/logger"
)

func (s *UsersService) Register(ctx context.Context, username, email, password string) (...) error {
	log := core_logger.FromContext(ctx)
	
	log.Debug("register attempt", "username", username, "email", email)
	
	if username == "" || email == "" || password == "" {
		log.Warn("register validation failed", "reason", "missing fields", "email", email)
		return fmt.Errorf("username, email, password are required: %w", core_error.ErrInvlalidArgument)
	}
	
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("password hashing failed", "error", err, "email", email)
		return fmt.Errorf("hash password: %w", err)
	}
	
	user := domain_models.NewUser("", username, email, string(hashed), time.Now(), time.Now())
	createdUser, err := s.usersRepository.CreateUser(ctx, user)
	if err != nil {
		log.Error("create user failed", "error", err, "email", email)
		return fmt.Errorf("create user: %w", err)
	}
	
	log.Info("user registered successfully", "user_id", createdUser.ID, "email", email)
	return s.authService.IssueTokens(ctx, createdUser)
}
```

**Проверка:**
- [ ] Логи выводятся при register
- [ ] Структурированное логирование работает
- [ ] Нет PII в логах (пароли)

---

## 🟡 ДЕНЬ 3-4: БЕЗОПАСНОСТЬ И ПРОИЗВОДИТЕЛЬНОСТЬ

### Task 3.1: Добавить rate limiting middleware
**Файл:** `internal/core/transport/http/middleware/rate_limit.go` (новый)

```go
package middleware

import (
	"golang.org/x/time/rate"
	"net/http"
)

type RateLimiter struct {
	limiter *rate.Limiter
}

func NewRateLimiter(requestsPerSecond float64) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.limiter.Allow() {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

**Использование:**
```go
// cmd/server/main.go
loginLimiter := middleware.NewRateLimiter(5.0) // 5 requests/sec
mux.Handle("POST /api/auth/login", loginLimiter.Middleware(http.HandlerFunc(...)))
```

**Проверка:**
- [ ] Rate limiting работает
- [ ] 429 status code возвращается
- [ ] Нет false positives

---

### Task 3.2: Улучшить JWT token hashing
**Файл:** `features/jwt/service/hash.go`

```go
// ❌ БЫЛО
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

// ✅ СТАЛО
func hashToken(token string) string {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	return string(hashed)
}
```

**Проверка:**
- [ ] Bcrypt используется для токенов
- [ ] Хеширование детерминировано
- [ ] Сравнение работает

---

### Task 3.3: Добавить database indexes
**Файл:** `migrations/000003_add_indexes.up.sql` (обновить)

```sql
-- Если это не уже добавлено
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON gochat.users(email);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON gochat.users(username);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON gochat.refresh_tokens(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON gochat.refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_rooms_owner_id ON gochat.rooms(owner_id);
CREATE INDEX IF NOT EXISTS idx_rooms_is_private ON gochat.rooms(is_private);
```

**Проверка:**
- [ ] Миграция применяется
- [ ] Индексы видны в БД
- [ ] Queries быстрее

---

### Task 3.4: Добавить pagination validation
**Файл:** `features/users/service/service.go`

```go
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

func validatePagination(limit, offset *int) {
	if limit == nil {
		l := DefaultLimit
		limit = &l
	} else if *limit > MaxLimit {
		*limit = MaxLimit
	} else if *limit < 1 {
		l := 1
		limit = &l
	}
	
	if offset == nil {
		o := 0
		offset = &o
	} else if *offset < 0 {
		o := 0
		offset = &o
	}
}

func (s *UsersService) GetUsers(ctx context.Context, limit, offset *int) (...) error {
	validatePagination(limit, offset)
	return s.usersRepository.GetUsers(ctx, limit, offset)
}
```

**Проверка:**
- [ ] Отрицательные значения отклоняются
- [ ] Limit ≤ 100
- [ ] Default values работают

---

## 📋 ДЕНЬ 5: РЕАЛИЗАЦИЯ MISSING FEATURES

### Task 4.1: Реализовать Rooms Repository
**Файл:** `features/rooms/repository/postgres/repository.go` (новый)

```go
package rooms_repository_postgres

import (
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
)

type RoomsRepository struct {
	pool core_postgres_pool.Pool
}

func NewRoomsRepository(pool core_postgres_pool.Pool) *RoomsRepository {
	return &RoomsRepository{pool: pool}
}
```

Добавить методы:
- [ ] GetPublicRooms
- [ ] GetMyRooms
- [ ] GetRoomByID
- [ ] CreateRoom
- [ ] DeleteRoom
- [ ] AddMember
- [ ] RemoveMember

---

### Task 4.2: Реализовать Rooms Service
**Файл:** `features/rooms/service/service.go` (новый)

```go
package rooms_service

import (
	"context"
	domain_models "go-chat/internal/core/domain/models"
)

type RoomsService struct {
	repository RoomsRepository
}

type RoomsRepository interface {
	GetPublicRooms(ctx context.Context, limit, offset *int) ([]domain_models.Room, error)
	GetMyRooms(ctx context.Context, userID string) ([]domain_models.Room, error)
	// ... остальные методы
}

func NewRoomsService(repository RoomsRepository) *RoomsService {
	return &RoomsService{repository: repository}
}
```

Добавить методы:
- [ ] GetPublicRooms (с логированием)
- [ ] GetMyRooms (с логированием)
- [ ] CreateRoom (с валидацией)
- [ ] DeleteRoom (с проверкой owner)
- [ ] Join (с проверкой access)
- [ ] Leave (с проверкой)

---

### Task 4.3: Реализовать Direct Messages
**Файл:** `features/direct_messages/repository/postgres/repository.go` (новый)

```go
package dm_repository_postgres

import core_postgres_pool "go-chat/internal/core/repository/postgres/pool"

type DirectMessagesRepository struct {
	pool core_postgres_pool.Pool
}

func NewDirectMessagesRepository(pool core_postgres_pool.Pool) *DirectMessagesRepository {
	return &DirectMessagesRepository{pool: pool}
}
```

Методы:
- [ ] SendMessage
- [ ] GetMessages
- [ ] GetConversations
- [ ] DeleteMessage
- [ ] MarkAsRead

---

**Файл:** `features/direct_messages/service/service.go` (новый)

```go
package dm_service

type DirectMessagesService struct {
	repository DirectMessagesRepository
	userRepo   UserRepository
}
```

**Файл:** `features/direct_messages/transport/http/transport.go` (обновить)

```go
package dm_transport_http

import (
	"context"
	core_http_server "go-chat/internal/core/transport/http/server"
)

type DirectMessagesHTTPHandler struct {
	dmService DirectMessagesService
}

type DirectMessagesService interface {
	SendMessage(ctx context.Context, ...) error
	GetMessages(ctx context.Context, ...) error
	// ... остальные методы
}

func (h *DirectMessagesHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  "POST",
			Path:    "/dms",
			Handler: h.SendMessage,
		},
		// ... остальные routes
	}
}
```

---

## ✅ ФИНАЛЬНАЯ ПРОВЕРКА

### Checklist перед Production

```
MIGRATION & DATA:
- [ ] Все migrations применяются
- [ ] user_id правильного типа (UUID)
- [ ] Индексы созданы
- [ ] Constraints работают

CODE QUALITY:
- [ ] Нет typos в parameter names
- [ ] Правильные имена функций
- [ ] Типы согласованы (UUID everywhere)
- [ ] Interfaces в правильном месте

SECURITY:
- [ ] Race condition fixed
- [ ] Email validation работает
- [ ] Rate limiting включен
- [ ] Token hashing усилено
- [ ] Cookie security настроена

FEATURES:
- [ ] Users: полная реализация
- [ ] JWT: полная реализация
- [ ] Rooms: реализовано
- [ ] Direct Messages: реализовано

LOGGING & MONITORING:
- [ ] Structured logging везде
- [ ] Нет PII в логах
- [ ] Error handling proper
- [ ] Metrics собираются

TESTING:
- [ ] Unit tests написаны
- [ ] Integration tests написаны
- [ ] Race condition tests пройдены
- [ ] Load testing сделано

DEPLOYMENT:
- [ ] Docker образ создан
- [ ] Health check endpoint
- [ ] Graceful shutdown работает
- [ ] Config validation в place
```

---

## 📊 METRICS ДО И ПОСЛЕ

| Метрика | До | После | Улучшение |
|---------|---|-------|-----------|
| Code Readiness | 39% | 95%+ | +56% |
| Security Issues | 7 | 0 | -100% |
| Critical Bugs | 5 | 0 | -100% |
| Features Complete | 50% | 100% | +50% |
| Test Coverage | 0% | 60%+ | +60% |

---

**Estimated Time: 2-3 weeks for complete implementation**

