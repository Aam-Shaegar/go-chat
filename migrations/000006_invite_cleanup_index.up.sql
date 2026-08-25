-- Частичный индекс для ускорения очистки просроченных инвайтов
CREATE INDEX IF NOT EXISTS idx_room_invites_cleanup
ON gochat.room_invites (is_active, expires_at, max_uses, uses)
WHERE is_active = false
   OR expires_at IS NOT NULL
   OR max_uses > 0;