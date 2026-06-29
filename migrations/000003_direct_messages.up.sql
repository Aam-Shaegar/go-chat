CREATE TABLE IF NOT EXISTS gochat.direct_messages (
    room_id      UUID PRIMARY KEY REFERENCES gochat.rooms(id) ON DELETE CASCADE,
    user_id_low  UUID NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    user_id_high UUID NOT NULL REFERENCES gochat.users(id) ON DELETE CASCADE,
    CHECK (user_id_low <> user_id_high),
    UNIQUE (user_id_low, user_id_high)
);

INSERT INTO gochat.direct_messages (room_id, user_id_low, user_id_high)
SELECT
    r.id,
    LEAST(rm1.user_id, rm2.user_id),
    GREATEST(rm1.user_id, rm2.user_id)
FROM gochat.rooms r
JOIN gochat.room_members rm1 ON rm1.room_id = r.id
JOIN gochat.room_members rm2 ON rm2.room_id = r.id AND rm1.user_id < rm2.user_id
WHERE r.is_dm = TRUE
ON CONFLICT DO NOTHING;
