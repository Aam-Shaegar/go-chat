CREATE SCHEMA IF NOT EXISTS gochat;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS gochat.users (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    username   VARCHAR(32)  NOT NULL UNIQUE,
    email      VARCHAR(255) NOT NULL UNIQUE,
    password   VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email    ON gochat.users(email);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON gochat.users(username);

CREATE TABLE IF NOT EXISTS gochat.refresh_tokens (
    id         UUID        PRIMARY KEY,
    user_id    UUID        REFERENCES gochat.users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP   NOT NULL,
    created_at TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON gochat.refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires  ON gochat.refresh_tokens(expires_at);

CREATE TYPE gochat.member_role AS ENUM ('owner', 'admin', 'member');

CREATE TABLE IF NOT EXISTS gochat.rooms (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(64)  NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    is_private  BOOLEAN      NOT NULL DEFAULT FALSE,
    is_dm       BOOLEAN      NOT NULL DEFAULT FALSE,
    owner_id    UUID         NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gochat.room_members (
    room_id   UUID               NOT NULL REFERENCES gochat.rooms(id) ON DELETE CASCADE,
    user_id   UUID               NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    role      gochat.member_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_room_members_user ON gochat.room_members(user_id);

CREATE TABLE IF NOT EXISTS gochat.room_invites (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id    UUID        NOT NULL REFERENCES gochat.rooms(id) ON DELETE CASCADE,
    token      VARCHAR(64) NOT NULL UNIQUE,
    created_by UUID        NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    max_uses   INT         NOT NULL DEFAULT 1,
    uses       INT         NOT NULL DEFAULT 0,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_room_invites_token ON gochat.room_invites(token);

CREATE TABLE IF NOT EXISTS gochat.messages (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id      UUID        NOT NULL REFERENCES gochat.rooms(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    reply_to_id  UUID        REFERENCES gochat.messages(id) ON DELETE SET NULL,
    content      TEXT        NOT NULL,
    is_encrypted BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_messages_room_created ON gochat.messages(room_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_reply_to     ON gochat.messages(reply_to_id);

CREATE TABLE IF NOT EXISTS gochat.message_reactions (
    message_id UUID        NOT NULL REFERENCES gochat.messages(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    emoji      VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);