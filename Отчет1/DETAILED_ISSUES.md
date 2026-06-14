# Детальная Таблица Проблем GoChat/features

## 1. ПРОБЛЕМЫ ДАННЫХ И ТИПОВ

| ID | Файл | Строка | Проблема | Severity | Влияние | Решение |
|---|---|---|---|---|---|---|
| DB-001 | `migrations/000001_init_up.sql` | 11 | `user_id INT` но users.id `UUID` | 🔴 CRITICAL | Foreign key не работает, токены не сохраняются | Изменить на `user_id UUID` |
| DB-002 | `migrations/000001_init_up.sql` | - | Отсутствуют primary key и index на `token_hash` | 🟠 HIGH | Slow lookups, возможны дубликаты | `CREATE UNIQUE INDEX idx_refresh_tokens_hash ON gochat.refresh_tokens(token_hash)` |
| TYPE-001 | `features/rooms/transport/http/transport.go` | 10-12 | `userID int`, но должна быть UUID | 🔴 CRITICAL | Type mismatch с User model | Изменить на `userID string` |
| TYPE-002 | `features/rooms/transport/http/transport.go` | 14 | `roomID int` но rooms.id `UUID` | 🔴 CRITICAL | Неправильный тип в interface | Изменить на `roomID string` |
| TYPE-003 | `features/jwt/repository/postgres/models.go` | 8 | `UserID string` но БД ожидает INT | 🔴 CRITICAL | Parsing error при scan | Должна быть `UserID UUID` (если миграция исправлена) |

---

## 2. ПРОБЛЕМЫ РЕАЛИЗАЦИИ

| ID | Файл | Проблема | Severity | Последствие | Решение |
|---|---|---|---|---|---|
| IMPL-001 | `features/direct_mesages/` | Полностью пусто (0% реализации) | 🔴 CRITICAL | Feature вообще не работает | Реализовать Repository/Service/Transport |
| IMPL-002 | `features/rooms/transport/http/transport.go` | Routes() возвращает `{}` (пустой Route) | 🔴 CRITICAL | Panic при запуске, комнаты недоступны | Добавить валидные routes |
| IMPL-003 | `features/rooms/transport/http/transport.go` | Methods Join, KickMember, GetMessages без параметров | 🟠 HIGH | Невозможно вызвать, не знаем что передавать | Добавить требуемые параметры |
| IMPL-004 | `features/rooms/` | repository/ и service/ пусто | 🔴 CRITICAL | Нет бизнес-логики | Полная реализация слоев |
| IMPL-005 | `features/direct_mesages/` | Файл transport.go пуст | 🟠 HIGH | Transport layer отсутствует | Реализовать handlers |

---

## 3. ПРОБЛЕМЫ КОДА И ОПЕЧАТКИ

| ID | Файл | Строка | Проблема | Severity | Влияние | Решение |
|---|---|---|---|---|---|
| TYPO-001 | `features/direct_mesages/` | - | Папка названа `mesages` вместо `messages` | 🟠 HIGH | Inconsistent naming, confusion | Переименовать на `direct_messages` |
| TYPO-002 | `features/users/transport/http/transport.go` | 15 | Parameter `offser` вместо `offset` | 🟡 MEDIUM | Confusing API, неясно что это | Исправить на `offset` |
| TYPO-003 | `features/users/transport/http/transport.go` | 16 | Parameter `pasword` вместо `password` | 🟡 MEDIUM | Неправильное имя параметра | Исправить на `password` |
| TYPO-004 | `features/rooms/transport/http/transport.go` | 30 | Function `NewUsersHTTPHandler` для rooms | 🟠 HIGH | Confusing, копипаста ошибка | Переименовать на `NewRoomsHTTPHandler` |

---

## 4. ПРОБЛЕМЫ БЕЗОПАСНОСТИ

| ID | Файл | Проблема | Severity | Risk | Mitigation |
|---|---|---|---|---|---|
| SEC-001 | `features/jwt/service/hash.go` | SHA256 без salt для хеширования токена | 🟠 HIGH | Rainbow table attack | Использовать bcrypt/argon2 |
| SEC-002 | `features/users/service/register.go` | Race condition: check email → create user | 🔴 CRITICAL | Duplicate email при concurrent register | Использовать DB constraint или транзакцию |
| SEC-003 | `features/users/service/register.go` | Нет валидации email формата | 🟡 MEDIUM | Invalid emails в БД | Добавить email regex или library |
| SEC-004 | `features/users/service/register.go` | Нет rate limiting на register | 🟠 HIGH | Brute force attack | Добавить middleware rate limiter |
| SEC-005 | `features/users/service/login.go` | Нет rate limiting на login | 🔴 CRITICAL | Brute force passwords | Добавить middleware rate limiter |
| SEC-006 | `features/users/service/` | Информативные error messages | 🟡 MEDIUM | User enumeration | Использовать generic messages |
| SEC-007 | `features/jwt/transport/http/refresh.go` | Нет CSRF token проверки | 🟠 HIGH | CSRF attack | Добавить CSRF middleware |
| SEC-008 | `features/users/transport/http/register.go` | Cookie без Domain и Path ограничений | 🟡 MEDIUM | Potential cookie sharing | Добавить `Domain: "example.com"` |

---

## 5. ПРОБЛЕМЫ АРХИТЕКТУРЫ

| ID | Файл | Проблема | Severity | Нарушение | Решение |
|---|---|---|---|---|---|
| ARCH-001 | `features/users/transport/http/transport.go` | Interface определена в transport | 🟠 HIGH | DIP (Dependency Inversion) | Переместить в service пакет |
| ARCH-002 | `features/jwt/service/service.go` | Нет transaction support | 🟠 HIGH | Data consistency | Добавить tx layer |
| ARCH-003 | `features/users/` | Отсутствует pagination validation | 🟡 MEDIUM | DoS (большие limit) | Валидировать limit ≤ 100 |
| ARCH-004 | `features/users/` | Нет error type differentiation | 🟡 MEDIUM | Можно отличить "not found" от "db error" | Использовать error types |
| ARCH-005 | `features/jwt/` | Отсутствует token rotation | 🟡 MEDIUM | JWT refresh предсказуем | Генерировать новый refresh token |
| ARCH-006 | `features/users/repository/` | Нет logging в repository | 🟡 MEDIUM | Трудно дебажить SQL issues | Добавить structured logging |

---

## 6. ПРОБЛЕМЫ ПРОИЗВОДИТЕЛЬНОСТИ

| ID | Файл | Проблема | Severity | Impact | Решение |
|---|---|---|---|---|---|
| PERF-001 | `migrations/` | Отсутствуют индексы на email, user_id | 🟠 HIGH | O(n) lookups вместо O(1) | Добавить индексы в migration |
| PERF-002 | `features/users/repository/` | N+1 query risk (если GetUsers + rooms) | 🟡 MEDIUM | Linear performance degradation | Использовать JOIN |
| PERF-003 | `features/` | Отсутствует caching layer | 🟡 MEDIUM | Много DB calls для same data | Добавить Redis |
| PERF-004 | `internal/core/repository/` | OpTimeout() не видна | 🟡 MEDIUM | Может быть слишком большой/малой | Убедиться 5-10 seconds |
| PERF-005 | `features/users/` | GetUserByEmail для каждого login | 🟡 MEDIUM | В session часто вызывается | Добавить user cache |

---

## 7. ПРОБЛЕМЫ ЛОГИРОВАНИЯ И МОНИТОРИНГА

| ID | Файл | Проблема | Severity | Impact | Решение |
|---|---|---|---|---|---|
| LOG-001 | `features/users/service/` | Нет логирования в service | 🟡 MEDIUM | Невозможно trace успешные операции | Добавить .Info() logs |
| LOG-002 | `features/jwt/service/` | Нет логирования token generation | 🟡 MEDIUM | Невозможно detect token spam | Добавить .Warn() для failed attempts |
| LOG-003 | `features/` | Нет structured logging | 🟡 MEDIUM | Сложно парсить logs | Использовать JSON logging |
| LOG-004 | `internal/core/` | Нет metrics (Prometheus) | 🟡 MEDIUM | Нет visibility в production | Добавить metrics middleware |
| LOG-005 | `features/users/service/register.go` | Нет rate limit logging | 🟡 MEDIUM | Невозможно detect attacks | Логировать каждый failure |

---

## 8. ПРОБЛЕМЫ ТЕСТИРОВАНИЯ

| ID | Компонент | Проблема | Severity | Последствие | Решение |
|---|---|---|---|---|---|
| TEST-001 | `features/users/` | Нет unit tests | 🟡 MEDIUM | Рефакторинг = риск регрессии | Написать tests |
| TEST-002 | `features/jwt/` | Нет integration tests | 🟠 HIGH | JWT bugs выявляются в prod | Написать e2e tests |
| TEST-003 | `features/` | Нет race condition tests | 🟠 HIGH | Concurrency bugs скрываются | `go test -race` |
| TEST-004 | `features/users/service/register.go` | Нет edge case tests | 🟡 MEDIUM | SQL injection, encoding issues | Fuzz testing |

---

## 9. ВНЕШНИЕ ЗАВИСИМОСТИ И КОНФИГУРАЦИЯ

| ID | Файл | Проблема | Severity | Impact | Решение |
|---|---|---|---|---|---|
| CONFIG-001 | `internal/core/config/config.go` | JWT secrets в env (plaintext risk) | 🟠 HIGH | Secrets могут быть expose | Использовать HashiCorp Vault |
| CONFIG-002 | `internal/core/` | Нет database config validation | 🟡 MEDIUM | Неправильный connection string = crash | Validate config в startup |
| CONFIG-003 | `features/` | Нет feature flags | 🟡 MEDIUM | Нельзя disable feature в runtime | Добавить feature flags |

---

## 10. ДОКУМЕНТАЦИЯ И API

| ID | Файл | Проблема | Severity | Impact | Решение |
|---|---|---|---|---|---|
| DOC-001 | `features/` | Нет OpenAPI/Swagger specs | 🟡 MEDIUM | Frontend не знает API contract | Генерировать Swagger docs |
| DOC-002 | `features/` | Нет code comments | 🟡 MEDIUM | Трудно понять intent | Добавить godoc comments |
| DOC-003 | `features/` | Нет error response documentation | 🟡 MEDIUM | Frontend не знает error codes | Документировать error responses |
| DOC-004 | Repository interfaces | Нет契約 documentation | 🟡 MEDIUM | Новые разработчики confused | Добавить interface docs |

---

## КРИТИЧЕСКИЙ ПУТЬ ФИКСОВ (Sequencing Dependencies)

```
Phase 1 (must fix first):
├── DB-001: Исправить migration user_id type
│   └── TYPE-003: Обновить JWT model
├── TYPO-001: Переименовать direct_mesages
├── TYPO-004: Исправить RoomsHandler constructor
└── IMPL-002: Добавить валидные routes в rooms

Phase 2 (depends on Phase 1):
├── SEC-002: Исправить race condition в register
├── IMPL-003: Реализовать rooms service/repository
└── IMPL-001: Реализовать direct_messages

Phase 3 (stability):
├── SEC-004/005: Добавить rate limiting
├── ARCH-001: Переместить interfaces
└── LOG-001/002: Добавить logging

Phase 4 (performance):
├── PERF-001: Добавить индексы
├── PERF-003: Добавить caching
└── PERF-005: Оптимизировать queries
```

---

## SUMMARY BY SEVERITY

### 🔴 CRITICAL (5)
- DB-001: user_id type mismatch
- TYPE-001,002: Room model type errors
- IMPL-001: Direct messages пусто
- IMPL-002: Empty routes в rooms
- SEC-005: No rate limiting on login

### 🟠 HIGH (12)
- DB-002: Missing indexes/constraints
- TYPO-001,004: Wrong names
- SEC-001,004,007: Security issues
- ARCH-002: No transactions
- PERF-001: No database indexes

### 🟡 MEDIUM (15)
- TYPO-002,003: Parameter name typos
- SEC-003,006,008: Security concerns
- ARCH-003,004,005: Architecture issues
- PERF-002,003,004,05: Performance
- LOG-001-005: Logging
- TEST-001,003,004: Testing

### 🟢 LOW (10)
- CONFIG-001,002,003: Configuration
- DOC-001-004: Documentation

**Всего: 42 проблемы**
- Критических: 5 (12%)
- Высоких: 12 (29%)
- Средних: 15 (36%)
- Низких: 10 (24%)

