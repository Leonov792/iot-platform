-- Лог срабатываний правил автоматизации — для бизнес-метрик Grafana.

CREATE TABLE IF NOT EXISTS automation_events (
    id        bigserial PRIMARY KEY,
    rule_id   text NOT NULL,
    rule_name text,
    device_id text,
    fired_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_automation_events_fired ON automation_events (fired_at DESC);
