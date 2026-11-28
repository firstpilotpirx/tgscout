-- The found chats and owner of docs/architecture/store.html: a restart uses
-- them instead of a lookup by name. No message text is kept here.
CREATE TABLE known_peer (
    role        TEXT NOT NULL,      -- 'chat' or 'owner'
    ref         TEXT NOT NULL,      -- the config text, as the config writes it
    kind        TEXT NOT NULL,      -- 'channel', 'chat' or 'user'
    peer_id     INTEGER NOT NULL,
    access_hash INTEGER NOT NULL,   -- 0 for a basic group
    username    TEXT NOT NULL,
    title       TEXT NOT NULL,
    resolved_at TIMESTAMP NOT NULL,
    PRIMARY KEY (role, ref)
);
