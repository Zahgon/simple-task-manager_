CREATE TABLE IF NOT EXISTS tasks (
    id          VARCHAR(36) PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    completed   BOOLEAN NOT NULL DEFAULT FALSE,
    due_date    TIMESTAMPTZ
);
