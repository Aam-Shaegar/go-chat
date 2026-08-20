-- Remove mute functionality from room_members
DROP INDEX IF EXISTS gochat.idx_room_members_muted;
ALTER TABLE gochat.room_members DROP COLUMN IF EXISTS muted_until;