CREATE TABLE IF NOT EXISTS users (
    id            text PRIMARY KEY,
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- FK на users не вешаю пока: с ALTER + IF NOT EXISTS возиться задолбался.
-- потом поправлю, когда руки дойдут до нормальных миграций
ALTER TABLE devices ADD COLUMN IF NOT EXISTS owner_id text;
