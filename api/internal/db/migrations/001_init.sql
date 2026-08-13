CREATE TABLE IF NOT EXISTS devices (
    id          text PRIMARY KEY,
    name        text NOT NULL,
    type        text NOT NULL DEFAULT '',
    status      text NOT NULL DEFAULT 'offline',
    created_at  timestamptz NOT NULL DEFAULT now(),
    last_seen   timestamptz NOT NULL DEFAULT now()
);
