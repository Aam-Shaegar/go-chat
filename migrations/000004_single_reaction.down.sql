-- Revert: restore original primary key (message_id, user_id, emoji)

ALTER TABLE gochat.message_reactions DROP CONSTRAINT message_reactions_pkey;
ALTER TABLE gochat.message_reactions ADD PRIMARY KEY (message_id, user_id, emoji);