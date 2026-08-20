-- Add mute functionality to room_members
ALTER TABLE gochat.room_members ADD COLUMN IF NOT EXISTS muted_until TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_room_members_muted ON gochat.room_members(muted_until) WHERE muted_until IS NOT NULL;