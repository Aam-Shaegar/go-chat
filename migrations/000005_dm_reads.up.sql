CREATE TABLE dm_reads (
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    other_user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, other_user_id)
);