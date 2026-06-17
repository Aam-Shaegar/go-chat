CREATE TABLE gochat.room_reads (
    room_id      UUID        NOT NULL REFERENCES gochat.rooms(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);
