# GoChat

Современный real-time мессенджер, разработанный на Go с использованием чистой слоистой архитектуры. Проект поддерживает комнаты, личные сообщения, JWT-аутентификацию и обмен сообщениями через WebSocket.

---

# Возможности

- Регистрация и авторизация пользователей
- JWT Access + Refresh Tokens
- Создание публичных и приватных комнат
- Управление участниками комнаты
- Приглашение пользователей по инвайт-ссылкам
- Прямые сообщения (Direct Messages)
- Обмен сообщениями в режиме реального времени через WebSocket
- Отслеживание непрочитанных сообщений
- Автоматическая очистка просроченных refresh-токенов
- Поддержка нескольких WebSocket-инстансов через Redis Pub/Sub
- Docker-based deployment

---

# Технологии

## Backend

- Go 1.25
- net/http
- PostgreSQL
- pgx
- Redis
- JWT
- Gorilla WebSocket
- Docker
- Docker Compose

## Frontend

- React 19
- TypeScript
- Vite
- Zustand
- React Query
- Axios

## Infrastructure

- Nginx
- Docker Compose
- PostgreSQL 16
- Redis 7

---

# Архитектура

Backend построен по принципам слоистой архитектуры.

```

HTTP
↓
Transport
↓
Service
↓
Repository
↓
PostgreSQL

```

Каждая функциональность выделена в отдельный feature:

```

internal/
features/
users/
rooms/
messages/
dm/
reads/
jwt/
ws/

```

Каждый feature содержит собственные слои:

```

transport/
service/
repository/

```

Такой подход позволяет независимо развивать функциональность проекта и упрощает тестирование.

---

# Структура проекта

```

.
├── cmd/
│   └── server/
├── internal/
├── frontend/
├── migrations/
├── docker/
│   └── nginx/
├── build/
├── Dockerfile.backend
├── Dockerfile.backend.runtime
├── docker-compose.yml
└── README.md

```

---

# Запуск проекта (Production)

## 1. Клонировать репозиторий

```bash
git clone https://github.com/<your-repository>.git

cd go-chat
```

## 2. Создать .env

```bash
cp .env.example .env
```

Заполнить необходимые переменные окружения.

---

## 3. Запустить приложение

```bash
docker compose up -d
```

Docker автоматически:

- поднимет PostgreSQL;
- поднимет Redis;
- выполнит миграции;
- запустит backend;
- запустит Nginx.

---

## Проверка

```bash
docker compose ps
```

Просмотр логов:

```bash
docker compose logs backend

docker compose logs nginx

docker compose logs postgres
```

---

# Обновление приложения

После внесения изменений:

```bash
git pull

docker compose build

docker compose up -d
```

---

# Переменные окружения

Минимально необходимые:

```text
POSTGRES_HOST
POSTGRES_PORT
POSTGRES_USER
POSTGRES_PASSWORD
POSTGRES_NAME
POSTGRES_TIMEOUT

HTTP_ADDR
HTTP_SHUTDOWN_TIMEOUT

REDIS_ADDR

JWT_ACCESS_SECRET
JWT_REFRESH_SECRET
JWT_ACCESS_TTL
JWT_REFRESH_TTL

TIME_ZONE

CORS_ALLOWED_ORIGINS
SECURE_REFRESH_COOKIE

LOGGER_LEVEL
LOGGER_FOLDER
```

---

# API

Все HTTP-эндпоинты доступны по адресу

```
/api/v1
```

Основные группы:

- Authentication
- Users
- Rooms
- Messages
- Direct Messages
- Invites
- Reads
- WebSocket

---

# WebSocket

Подключение:

```
/api/v1/ws
```

Передача сообщений осуществляется через WebSocket Hub.

Для масштабирования между несколькими экземплярами приложения используется Redis Pub/Sub.

---

# Используемая инфраструктура

```

Internet
│
▼
Nginx
│
├───────────────┐
▼               ▼
Backend      Static React
│
├───────────────┐
▼               ▼
PostgreSQL    Redis

```

---

# Основные возможности

## Комнаты

- создание комнат;
- публичные и приватные комнаты;
- управление участниками;
- роли участников;
- приглашения по токенам.

## Сообщения

- отправка сообщений;
- история сообщений;
- непрочитанные сообщения;
- отметка о прочтении.

## Личные сообщения

- создание диалогов;
- история переписки;
- WebSocket-обновления.

## Аутентификация

- регистрация;
- вход;
- refresh токены;
- автоматическое обновление access token.

---

# Roadmap

Планируемые улучшения:

- загрузка файлов;
- редактирование сообщений;
- удаление сообщений;
- реакции на сообщения;
- поиск по сообщениям;
- Docker Hub CI/CD;
- GitHub Actions;
- интеграционные тесты;
- OpenAPI/Swagger;
- мониторинг (Prometheus + Grafana).

---

# Лицензия

Проект создан в образовательных целях и используется как pet-проект для изучения разработки высоконагруженных backend-приложений на Go.