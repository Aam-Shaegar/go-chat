# GoChat

Реал-тайм чат-приложение с поддержкой комнат, прямых сообщений и приглашений.

---

## Возможности

- **Публичные и приватные комнаты** — создание, управление членами, приглашение через токены
- **Прямые сообщения** — переписка между пользователями
- **Реал-тайм сообщения** — через WebSocket
- **Отслеживание прочитанных сообщений** — система меток "прочитано"
- **Аутентификация** — регистрация, вход, JWT + refresh-токены
- **Управление сессиями** — автоматическая чистка старых токенов

---

## Стек

**Backend:**
- Go 1.20+ (echo, pgx, redis)
- PostgreSQL 16
- Redis 7
- Миграции (migrate/migrate)

**Frontend:**
- React 18 + TypeScript
- Vite
- TailwindCSS, Zustand (state management)
- WebSocket для live-обновлений

---

## Архитектура

**Backend:** чистая слоистая архитектура (transport → service → repository). Каждый feature (users, rooms, messages, ws) — независимый пакет с собственным слоем хранилища.

**Frontend:** компоненты + хуки для WebSocket и состояния (chatStore, authStore).

**Почему:**
- Слои разделены для тестируемости и масштабируемости.
- Redis нужен для WebSocket hub (pub/sub между инстансами).
- JWT для stateless auth.

---

## Как запустить

### Требования
- Go 1.20+
- Node.js 18+ (npm)
- Docker & Docker Compose

### Шаги

**1. Подготовка**
```bash
# Скопируйте и заполните .env (уже есть в репо)
# Убедитесь, что Docker Compose запущен
```

**2. Инфраструктура (PostgreSQL + Redis)**
```bash
make infra
```

**3. Миграции БД**
```bash
make migrate-up
```

**4. Сборка фронтенда (production)**
```bash
make frontend-build
# Результат в frontend/dist
```

**5. Запуск**

Опция A — разработка (go run + npm dev):
```bash
make run              # бэкенд на :8080
make frontend-dev     # фронтенд на :5173
```

Опция B — production-подобная сборка:
```bash
make build-all        # собрать оба
make backend-run      # бэкенд (bin/gochat)
make frontend-serve   # статический сервер для dist
```

**6. Откройте приложение**
```
http://localhost:5173  (dev)
http://localhost:3000  (serve)
```

