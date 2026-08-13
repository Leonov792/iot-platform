-- устройства: комната + управляемое состояние (jsonb)
ALTER TABLE devices ADD COLUMN IF NOT EXISTS room text NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN IF NOT EXISTS state jsonb NOT NULL DEFAULT '{}'::jsonb;

-- история телеметрии для графиков
CREATE TABLE IF NOT EXISTS telemetry (
    id        bigserial PRIMARY KEY,
    device_id text NOT NULL,
    ts        timestamptz NOT NULL DEFAULT now(),
    payload   jsonb NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_telemetry_device_ts ON telemetry (device_id, ts DESC);
