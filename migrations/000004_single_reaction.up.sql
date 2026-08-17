-- Change message_reactions to allow only one reaction per user per message
-- Primary key changes from (message_id, user_id, emoji) to (message_id, user_id)

ALTER TABLE gochat.message_reactions DROP CONSTRAINT message_reactions_pkey;
ALTER TABLE gochat.message_reactions ADD PRIMARY KEY (message_id, user_id);