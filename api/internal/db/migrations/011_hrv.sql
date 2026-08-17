-- HRV (вариабельность сердечного ритма) — для предсказания фазы пробуждения.

CREATE TABLE IF NOT EXISTS hrv_samples (
    id      bigserial PRIMARY KEY,
    user_id text NOT NULL,
    value   real NOT NULL,
    ts      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_hrv_user_ts ON hrv_samples (user_id, ts DESC);
