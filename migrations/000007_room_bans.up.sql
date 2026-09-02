CREATE TABLE IF NOT EXISTS gochat.room_bans (
    room_id UUID NOT NULL REFERENCES gochat.rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    banned_by UUID NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    reason VARCHAR(255),
    expires_at TIMESTAMPTZ, -- NULL = permanent
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_room_bans_user ON gochat.room_bans(user_id);
CREATE INDEX IF NOT EXISTS idx_room_bans_expires ON gochat.room_bans(expires_at) WHERE expires_at IS NOT NULL;