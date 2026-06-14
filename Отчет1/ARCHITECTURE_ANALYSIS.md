# Анализ архитектуры GoChat/internal/features

**Дата анализа:** 18 мая 2026  
**Версия проекта:** v1.0 (разработка)  

---

## 📋 СОДЕРЖАНИЕ

1. [Обзор архитектуры](#обзор-архитектуры)
2. [Анализ по компонентам](#анализ-по-компонентам)
3. [Критические проблемы](#критические-проблемы)
4. [Проблемы архитектуры](#проблемы-архитектуры)
5. [Вопросы безопасности](#вопросы-безопасности)
6. [Вопросы производительности](#вопросы-производительности)
7. [Чек-лист рекомендаций](#чек-лист-рекомендаций)

---

## 🏗️ ОБЗОР АРХИТЕКТУРЫ

### Структура features/

```
features/
├── users/               ✅ Полная реализация (Repository/Service/Transport)
│   ├── repository/
│   ├── service/
│   └── transport/http/
├── jwt/                 ⚠️  Частичная реализация (Service/Transport+Repository)
│   ├── repository/
│   ├── service/
│   └── transport/http/
├── direct_mesages/      ❌ Incomplete (пусто, только структура)
│   ├── repository/      (пусто)
│   ├── service/         (пусто)
│   └── transport/http/  (пусто - файл есть, но пуст)
└── rooms/               ⚠️  Skeleton (только transport.go с методами)
    ├── repository/      (пусто)
    ├── service/         (пусто)
    └── transport/http/  (только скелет)
```

### Паттерн архитектуры

Используется **трёхслойная архитектура**:

```
Transport (HTTP) -> Service (бизнес-логика) -> Repository (данные)
```

**Уровень 1: Transport (HTTP)**
- Разбор HTTP запроса
- Валидация входных данных
- Преобразование в DTO
- Логирование и обработка ошибок
- Установка cookies

**Уровень 2: Service (бизнес-логика)**
- Проверка бизнес-правил
- Координация repository операций
- Обработка ошибок домена
- Вызов auth service

**Уровень 3: Repository (данные)**
- Прямой доступ к БД
- SQL запросы
- Преобразование в domain models
- Управление контекстом и timeouts

---

## 🔍 АНАЛИЗ ПО КОМПОНЕНТАМ

### 1. USERS (✅ Полная реализация)

#### Компоненты:
- `repository/postgres/` - CreateUser, GetUser, GetUsers, GetUserByEmail, UserExistsByEmail, UserExistsByUsername
- `service/` - Register, Login, GetUser, GetUsers
- `transport/http/` - Register, Login, GetUser, GetUsers

#### ✅ Достоинства:
1. **Полная цепочка**: Transport → Service → Repository
2. **Консистентная обработка ошибок**: Все ошибки оборачиваются с контекстом
3. **Правильная валидация**: Проверка на пустые значения в service
4. **Безопасность паролей**: Использование bcrypt с DefaultCost
5. **DTO слой**: Разделение между input/response объектами
6. **Контроль контекста**: Timeout в repository операциях
7. **Правильные HTTP коды**: 201 Created, 200 OK, 400 Bad Request

#### ⚠️ Проблемы:

**Опечатки в коде:**
- [auth_handler.go в старом коде](../internal/handler/auth_handler.go): переменная `offser` вместо `offset`
```go
GetUsers(ctx context.Context, limit, offser *int) ([]domain_models.User, error)
```

**Недостаточная валидация email:**
```go
// В RegisterInputDTO - используется встроенная валидация "email" tag
// Но нет проверки на валидный email формат
Email    string `json:"email" validate:"required,min=5,max=255"`
```

**Возможная race condition:**
```go
// Проверка существования + создание - не atomic!
exists, err := s.usersRepository.UserExistsByEmail(ctx, email)
if exists {
    return error
}
// Между этой проверкой и CreateUser другой goroutine может создать пользователя
createdUser, err := s.usersRepository.CreateUser(ctx, user)
```

**Логирование ошибок:**
- Нет логирования в service слое
- Ошибки только логируются в transport слое
- Сложно трассировать проблемы в production

---

### 2. JWT (⚠️ Частичная реализация)

#### Компоненты:
- `service/` - GenerateAccessToken, GenerateRefreshToken, ValidateRefreshToken, Refresh, IssueTokens, hashToken, models
- `transport/http/` - Refresh handler
- `repository/postgres/` - SaveRefreshToken, GetRefreshToken, RevokeRefreshToken

#### ✅ Достоинства:
1. **Двухуровневая токенизация**: Access + Refresh tokens
2. **Хеширование токенов**: Refresh tokens хешируются перед сохранением в БД
3. **Revocation механизм**: Refresh token revocation в БД
4. **HMAC подпись**: Правильный тип подписи (HS256)
5. **Правильное управление TTL**: Отдельные TTL для access и refresh
6. **Cookie безопасность**: HttpOnly, Secure, SameSite=Strict

#### ⚠️ Проблемы:

**1. Критическая ошибка в БД schema:**
```sql
-- migrations/000001_init_up.sql
user_id INT REFERENCES gochat.users(id)
-- ❌ ОШИБКА: users.id - UUID, но user_id - INT!
-- Должно быть: user_id UUID
```
**Последствие:** Refresh tokens НЕ МОГУТ быть сохранены! Будет ошибка constraint violation.

**2. Токен не сохраняется в БД перед использованием:**
```go
// IssueTokens: сохраняется ХЕШИРОВАН, но в Validate нет проверки существования?
err = s.tokenRepository.SaveRefreshToken(ctx, user.ID, hashToken(refreshToken), ...)

// Но user.ID может быть:
// - string (UUID)
// - int (если было обновлено)
// Несогласованность типов!
```

**3. Недостаточная валидация кода токена:**
```go
// ValidateRefreshToken не проверяет наличие токена в БД!
// Только парсит JWT
// Затем в Refresh вызывает GetRefreshToken - может быть уже удален

// Но есть race condition:
userID, err := s.ValidateRefreshToken(ctx, refreshToken)  // ✓ JWT valid
stored, err := s.tokenRepository.GetRefreshToken(ctx, ...)  // ✓ В БД
// Между этими линиями token может быть revoked другим процессом!
```

**4. Слабая хеш функция для токенов:**
```go
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
// SHA256 работает, но нет salt
// Если токен перехвачен - хеш предсказуем
```

**5. Нет логирования важных событий:**
- Refresh token generation
- Token validation failures
- Token revocation
- Потенциальные атаки (множественные попытки)

**6. Время жизни токена не валидируется:**
```go
// JWT exp claim и БД expires_at могут не совпадать!
// Если TTL в конфиге меняется между запросами
```

**7. Проблема с моделью в JWT service:**
```go
type RefreshTokenModel struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`  // ← string, но в БД это INT!
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}
```

---

### 3. DIRECT_MESSAGES (❌ Не реализовано)

#### Статус:
```
direct_mesages/
├── repository/       (пусто)
├── service/          (пусто)
└── transport/http/   (пусто - файл существует но empty)
```

#### ❌ Проблемы:
1. **Полностью не реализовано** - структура есть, но нет кода
2. **Опечатка в названии** - `direct_mesages` вместо `direct_messages`
3. **Empty transport handler** - файл существует но не содержит кода
4. **Нет интеграции** - Не включено в main.go

#### 🔧 Требуется реализация:
- [ ] Repository слой (CRUD операции)
- [ ] Service слой (бизнес-логика ДМ)
- [ ] Transport слой (HTTP endpoints)
- [ ] WebSocket интеграция
- [ ] Валидация данных

---

### 4. ROOMS (⚠️ Skeleton только)

#### Статус:
```go
// transport/http/transport.go
type RoomsHTTPHandler struct {
	roomsService RoomsService
	cfg          *core_config.Config
}

type RoomsService interface {
	GetPublicRooms(ctx context.Context, limit, offset *int) ([]domain_models.Room, error)
	GetMyRooms(ctx context.Context, userID int) ([]domain_models.Room, error)
	GetRoomByID(ctx context.Context, roomID int) (domain_models.Room, error)
	CreateRoom(ctx context.Context) (domain_models.Room, error)
	DeleteRoom(ctx context.Context, roomID int) error
	Leave(ctx context.Context, userID, roomID int) error
	Join(ctx context.Context)
	KickMember(ctx context.Context)
	GetMessages(ctx context.Context)
	GetMembers(context.Context)
}

func (h *RoomsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{},  // ❌ Пустой route!
	}
}
```

#### ❌ Проблемы:
1. **Методы без реализации** - Join, KickMember, GetMessages, GetMembers нет параметров
2. **Пустые routes** - Routes() возвращает пустой Route
3. **Непоследовательные типы** - userID и roomID как `int`, но users.id - UUID!
4. **Нет repository/service** - Полностью не реализовано
5. **Ошибка в конструкторе** - `NewUsersHTTPHandler` вместо `NewRoomsHTTPHandler`

```go
// ❌ Неправильное имя
func NewUsersHTTPHandler(roomsService RoomsService, cfg *core_config.Config) *RoomsHTTPHandler {
```

---

## 🚨 КРИТИЧЕСКИЕ ПРОБЛЕМЫ

### 🔴 Проблема #1: БД Schema Type Mismatch

**Файл:** `migrations/000001_init_up.sql`

```sql
-- ❌ КРИТИЧЕСКАЯ ОШИБКА
CREATE TABLE gochat.users(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ...
);

CREATE TABLE gochat.refresh_tokens (
    id UUID PRIMARY KEY,
    user_id INT REFERENCES gochat.users(id) ON DELETE CASCADE,  -- ❌ INT вместо UUID!
    ...
);
```

**Последствие:** 
- Refresh tokens НЕ могут быть созданы
- Падает foreign key constraint
- JWT refresh механизм сломан в production

**Решение:** Изменить `user_id INT` на `user_id UUID`

---

### 🔴 Проблема #2: Несогласованность типов User ID

**Файлы:**
- `domain/models/user.go` - `ID string` (UUID)
- `features/jwt/repository/models.go` - `UserID string` (JWT model)
- `features/rooms/transport/transport.go` - `userID int` (rooms)
- Миграция - `user_id INT` (refresh_tokens)

**Последствие:**
- Типы не совпадают между слоями
- Возможна потеря данных при преобразовании
- Непредсказуемое поведение в runtime

**Решение:** Все ID должны быть UUID (string в Go)

---

### 🔴 Проблема #3: Race Condition в Register

```go
func (s *UsersService) Register(ctx context.Context, username, email, password string) {
	// Step 1: Проверка
	exists, err := s.usersRepository.UserExistsByEmail(ctx, email)
	if exists {
		return error  // Email занят
	}
	
	// ⏱️ WINDOW МЕЖДУ ПРОВЕРКОЙ И СОЗДАНИЕМ
	
	// Step 2: Создание
	createdUser, err := s.usersRepository.CreateUser(ctx, user)
}
```

**Сценарий:**
1. Thread A: Проверяет email "test@test.com" - не существует ✓
2. Thread B: Проверяет email "test@test.com" - не существует ✓
3. Thread A: Создает пользователя ✓
4. Thread B: Пытается создать пользователя - FAILS ❌ (constraint violation)

**Решение:** Использовать database-level constraints или atomic операции

---

### 🔴 Проблема #4: Empty HTTP Routes в Rooms

```go
func (h *RoomsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{},  // ❌ Пустой route - невалидный!
	}
}
```

**Последствие:**
- Приложение может не запуститься
- Panic при регистрации маршрутов
- Комнаты вообще не доступны через HTTP

---

## ⚠️ ПРОБЛЕМЫ АРХИТЕКТУРЫ

### 1. Несоответствие Interface Определений

**Проблема:** Interface определяется в transport слое, но должна быть в service

```go
// ❌ Неправильно: interface в transport
package users_transport_http

type UsersService interface {
	GetUsers(ctx context.Context, limit, offser *int) (...) error  // ← опечатка offser!
	GetUser(ctx context.Context, userID string) (...) error
	Register(ctx context.Context, username, email, pasword string) (...)  // ← опечатка pasword!
	Login(ctx context.Context, email, password string) (...) error
}

// ✅ Правильно: interface должна быть в service пакете
```

**Последствие:** Нарушение DIP (Dependency Inversion Principle)

---

### 2. Отсутствие Error Wrapping Context

```go
// ❌ Неправильно: потеря контекста ошибок
user, err := s.usersRepository.GetUserByEmail(ctx, email)
if err != nil {
	return "", "", fmt.Errorf(
		"invalid credentials: %w",
		core_error.ErrUnauthorized,
	)
}
```

**Проблема:**
- Невозможно отличить "email не найден" от "DB connection error"
- Трудно дебажить в production
- Нарушение SLA для разных ошибок

**Решение:** Использовать typesafe ошибки

```go
// ✅ Правильно
if errors.Is(err, sql.ErrNoRows) {
	return "", "", fmt.Errorf("user not found: %w", core_error.ErrUnauthorized)
}
return "", "", fmt.Errorf("database error: %w", err)
```

---

### 3. Отсутствие Transaction Support

**Проблема:** Сложные операции без транзакций

```go
// Предположим: Register должна создать пользователя И сохранить токен
// Но это два отдельных вызова repository
createdUser, err := s.usersRepository.CreateUser(ctx, user)
// ... потом в отдельном service методе
err = s.tokenRepository.SaveRefreshToken(...)
```

**Если сбой при Save token:**
- Пользователь создан, но не может залогиниться
- Данные в несогласованном состоянии

**Решение:** Использовать транзакции или saga pattern

---

### 4. Weak Input Validation

```go
// ❌ Валидация только в service
if username == "" || email == "" || password == "" {
	return error
}

// ❌ Нет валидации email формата
// ❌ Нет проверки на SQL injection (но защищено параметризованными запросами)
// ❌ Нет проверки на резервированные имена
```

**Решение:** 
1. Использовать validator library с tags
2. Валидировать в Transport слое перед Service
3. Проверять email format с regex

---

### 5. Отсутствие Pagination Validation

```go
// ⚠️ Неправильно: limit и offset не валидируются
func (s *UsersService) GetUsers(ctx context.Context, limit, offset *int) (...) error {
	// Что если limit = -1? offset = 999999999?
	// Нет проверок!
}
```

**Решение:**
```go
const (
	MaxLimit = 100
	DefaultLimit = 20
)

if limit == nil || *limit > MaxLimit {
	*limit = DefaultLimit
}
```

---

### 6. Недостаток Логирования

**Уровни логирования не используются:**
- DEBUG: Важные события (register, login, token refresh)
- INFO: Обычные операции
- WARN: Потенциальные проблемы (failed logins)
- ERROR: Ошибки (db errors, validation)

**Текущая ситуация:** Логирование только на transport слое, no structured logging

---

## 🔐 ВОПРОСЫ БЕЗОПАСНОСТИ

### 1. Timing Attack на Password Validation

```go
// ⚠️  Опасно: время выполнения зависит от длины пароля
if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
	// Если password неправильный, это может занять разное время
}
```

**Решение:** bcrypt уже устойчив к timing attacks (constant time), но всё равно хорошо явно указать.

---

### 2. Отсутствие Rate Limiting

```go
// Нет защиты от brute force
// Кто-то может пытаться логиниться тысячи раз в секунду
h.usersService.Login(ctx, req.Email, req.Password)
```

**Решение:** Добавить middleware с rate limiting

---

### 3. Слабое Хеширование Refresh Token

```go
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))  // ⚠️ Детерминированное хеширование без salt
	return fmt.Sprintf("%x", h)
}
```

**Проблема:** SHA256 является быстрой функцией, подходящей для файлов, но не для паролей/токенов

**Решение:** Использовать bcrypt или argon2 для хеширования токенов

```go
hashed, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
```

---

### 4. CSRF Токен Отсутствует

**Проблема:** POST запросы без CSRF защиты

```go
mux.HandleFunc("POST /api/auth/register", authHandler.Register)
```

**Решение:** Добавить CSRF middleware (особенно если frontend на том же домене)

---

### 5. Утечка Информации через Error Messages

```go
// ❌ Утечка: разные сообщения об ошибках
if exists {
	return fmt.Errorf("email already taken")  // ← подтверждает существование email
}

// ✅ Лучше
return fmt.Errorf("registration failed")  // ← информация пользователя не раскрывается
```

---

### 6. Отсутствие SQL Injection Protection

```go
// ✅ Хорошо: используется параметризованные запросы
query := `INSERT INTO gochat.users (...) VALUES ($1, $2, $3, $4, $5)`
row := r.pool.QueryRow(ctx, query, user.Username, user.Email, ...)

// Это защищает от SQL injection
```

---

### 7. Cookie Security

```go
// ✅ Хорошо
http.SetCookie(w, &http.Cookie{
	Name:     "refresh_token",
	HttpOnly: true,           // ✓ Защита от XSS
	Secure:   true,           // ✓ Только HTTPS
	SameSite: http.SameSiteStrictMode,  // ✓ CSRF защита
})
```

**Но:** Отсутствует domain restriction (должно быть `Domain: "example.com"`)

---

## ⚡ ВОПРОСЫ ПРОИЗВОДИТЕЛЬНОСТИ

### 1. N+1 Query Problem

**Риск в GetUsers:**
```go
// Если GetUsers возвращает users с room references:
users, _ := repo.GetUsers(ctx, limit, offset)
// Потом для каждого пользователя запрашиваются rooms?
for _, user := range users {
	userRooms, _ := repo.GetUserRooms(user.ID)  // ← N+1 problem!
}
```

**Решение:** JOIN запросы или batch loading

---

### 2. Отсутствие Connection Pooling Configuration

```go
// core/repository/postgres/pool используется, но параметры не видны
type Pool interface {
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	Exec(ctx context.Context, query string, args ...interface{}) (CommandTag, error)
	OpTimeout() time.Duration
}
```

**Требует проверки:**
- Max connections
- Connection lifetime
- Idle timeout

---

### 3. Отсутствие Caching

**Для часто запрашиваемых данных нет кэша:**
- User profile (GetUser, GetUserByEmail)
- Public rooms (GetPublicRooms)
- Room members

**Решение:** Добавить Redis layer

---

### 4. Context Timeout Configuration

```go
// ✓ Есть timeout в repository
ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
defer cancel()

// Но: OpTimeout() не видна, может быть слишком большой
// Рекомендация: 5-10 seconds для обычных queries
```

---

### 5. Отсутствие Database Indexes

**migration/000003_add_indexes.up.sql должна содержать:**
```sql
CREATE INDEX idx_users_email ON gochat.users(email);
CREATE INDEX idx_refresh_tokens_user_id ON gochat.refresh_tokens(user_id);
CREATE INDEX idx_rooms_owner_id ON gochat.rooms(owner_id);
```

**Требует проверки:** Есть ли индексы на часто используемые колонки?

---

### 6. Отсутствие Database Statistics

**Нет информации о:**
- Количество active queries
- Slow query log
- Connection pool stats

**Решение:** Добавить metrics collection (Prometheus)

---

## ✅ ЧЕК-ЛИСТ РЕКОМЕНДАЦИЙ

### 🔴 КРИТИЧЕСКИЙ (Блокирует Production)

- [ ] **Fix migration type mismatch**: `user_id INT` → `user_id UUID` в refresh_tokens
- [ ] **Fix rooms transport.go**: Пустой Route, пустые методы
- [ ] **Fix type inconsistency**: Все ID должны быть UUID (string)
- [ ] **Fix room handler constructor**: `NewUsersHTTPHandler` → `NewRoomsHTTPHandler`
- [ ] **Implement direct_messages**: Repository/Service/Transport layers
- [ ] **Fix opечатка в названии**: `direct_mesages` → `direct_messages`

### 🟠 ВЫСОКИЙ (Серьёзные проблемы)

- [ ] **Fix race condition**: Обернуть CreateUser операцию в транзакцию
- [ ] **Fix interface definitions**: Переместить interfaces из transport в service слой
- [ ] **Add proper error handling**: Differentiate между типами ошибок
- [ ] **Add password validation**: Min 8 chars, complexity requirements
- [ ] **Add email validation**: Проверить email format (regex или library)
- [ ] **Add logging**: Structured logging для всех слоев
- [ ] **Add rate limiting**: Middleware для защиты от brute force
- [ ] **Fix JWT token hashing**: Использовать bcrypt вместо SHA256
- [ ] **Add pagination validation**: Проверить limit/offset boundaries

### 🟡 СРЕДНИЙ (Улучшение качества)

- [ ] **Add database indexes**: На email, user_id, created_at
- [ ] **Add caching layer**: Redis для часто запрашиваемых данных
- [ ] **Add transaction support**: Для сложных операций
- [ ] **Add cookie domain**: Specify domain restriction
- [ ] **Add CSRF protection**: Middleware для POST/PUT/DELETE
- [ ] **Refactor error messages**: Не раскрывать информацию о пользователе
- [ ] **Add request validation middleware**: Перед попаданием в handler
- [ ] **Add API versioning**: /api/v1/users, /api/v2/users
- [ ] **Add metrics**: Prometheus metrics для мониторинга
- [ ] **Add slow query logging**: Для оптимизации queries

### 🟢 НИЗКИЙ (Nice to have)

- [ ] **Add request ID tracking**: Для трассирования через logs
- [ ] **Add health check endpoint**: /health, /health/db, /health/redis
- [ ] **Add documentation**: OpenAPI/Swagger specs
- [ ] **Add unit tests**: Для service и repository layers
- [ ] **Add integration tests**: Для HTTP endpoints
- [ ] **Add load testing**: Определить max throughput
- [ ] **Add graceful shutdown**: Для cleanup при остановке сервера
- [ ] **Add request/response logging**: Debug middleware
- [ ] **Add CORS configuration**: Explicit CORS policy
- [ ] **Add Refresh token rotation**: New refresh token при успешном refresh

---

## 🎯 ПРИОРИТИЗИРОВАННЫЙ ПЛАН ДЕЙСТВИЙ

### Phase 1: Критический Fixes (ДЕНЬ 1)
1. Исправить миграцию user_id type mismatch
2. Реализовать rooms handlers
3. Переименовать direct_mesages на direct_messages
4. Исправить конструктор rooms handler

### Phase 2: Архитектурные Улучшения (ДЕНЬ 2-3)
1. Добавить proper error handling
2. Исправить race condition в register
3. Переместить interfaces в service layer
4. Добавить structured logging

### Phase 3: Безопасность (ДЕНЬ 4-5)
1. Добавить rate limiting
2. Улучшить JWT token hashing
3. Добавить input validation
4. Добавить CSRF protection

### Phase 4: Performance (ДЕНЬ 6-7)
1. Добавить database indexes
2. Добавить caching layer
3. Оптимизировать queries
4. Добавить metrics

---

## 📊 СВОДКА ОЦЕНОК

| Компонент | Реализация | Архитектура | Безопасность | Производительность | Общая оценка |
|-----------|-----------|-------------|-------------|------------------|------------|
| **Users** | ✅ 100% | ⚠️ 70% | ⚠️ 60% | ⚠️ 70% | **75%** |
| **JWT** | ✅ 80% | ⚠️ 60% | ⚠️ 50% | ⚠️ 70% | **65%** |
| **Direct Messages** | ❌ 0% | - | - | - | **0%** |
| **Rooms** | ⚠️ 10% | ❌ 20% | - | - | **15%** |

**Общая оценка проекта: ~39% готовности**

---

## 🔗 СВЯЗАННЫЕ ФАЙЛЫ

### Требует исправления:
- [migrations/000001_init_up.sql](../migrations/000001_init_up.sql) - Type mismatch
- [internal/features/rooms/transport/http/transport.go](../internal/features/rooms/transport/http/transport.go) - Empty routes
- [internal/features/jwt/repository/postgres/models.go](../internal/features/jwt/repository/postgres/models.go) - Type inconsistency
- [internal/features/users/transport/http/transport.go](../internal/features/users/transport/http/transport.go) - Typos in interface
- [internal/core/errors/common.go](../internal/core/errors/common.go) - Add more error types

### Требует реализации:
- [internal/features/direct_mesages/](../internal/features/direct_mesages/) - Полная реализация
- [internal/features/rooms/repository/](../internal/features/rooms/repository/) - Пусто
- [internal/features/rooms/service/](../internal/features/rooms/service/) - Пусто

---

**Конец анализа**
