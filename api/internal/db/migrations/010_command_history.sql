-- Undo-стек (Event Sourcing): лог команд с предыдущим состоянием — для «Ctrl+Z».

CREATE TABLE IF NOT EXISTS command_history (
    id             bigserial PRIMARY KEY,
    device_id      text NOT NULL,
    user_id        text NOT NULL,
    action         text NOT NULL,
    previous_state jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_command_history_device ON command_history (device_id, created_at DESC);
