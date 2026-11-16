-- The store schema of docs/architecture/store.html.
CREATE TABLE chat_cursor (
    chat_id         INTEGER PRIMARY KEY,
    last_message_id INTEGER NOT NULL,
    updated_at      TIMESTAMP NOT NULL
);

CREATE TABLE processed_message (
    chat_id      INTEGER NOT NULL,
    message_id   INTEGER NOT NULL,
    processed_at TIMESTAMP NOT NULL,
    PRIMARY KEY (chat_id, message_id)
);

-- request_id is the id of a request in tgscout.yaml; nothing in SQLite enforces it.
CREATE TABLE decision (
    chat_id    INTEGER NOT NULL,
    message_id INTEGER NOT NULL,
    request_id TEXT NOT NULL,
    matched    BOOLEAN NOT NULL,
    decided_at TIMESTAMP NOT NULL,
    sent_at    TIMESTAMP NULL,
    PRIMARY KEY (chat_id, message_id, request_id)
);
