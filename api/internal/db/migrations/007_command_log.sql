-- Лог команд на устройства — сырьё для предиктивного анализа привычек (AI).

CREATE TABLE IF NOT EXISTS device_commands (
    id        bigserial PRIMARY KEY,
    device_id text NOT NULL,
    user_id   text NOT NULL,
    action    text NOT NULL,
    value     jsonb,
    ts        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_device_commands_device_ts ON device_commands (device_id, ts DESC);
