-- TimescaleDB: телеметрия становится гипертаблицей со сжатием старых данных.

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- PK на id мешает гипертаблице: ограничение должно включать колонку партиционирования ts.
-- bigserial id и так глобально уникален (общая последовательность), поэтому (id, ts) валиден.
ALTER TABLE telemetry DROP CONSTRAINT IF EXISTS telemetry_pkey;

SELECT create_hypertable('telemetry', 'ts', if_not_exists => TRUE, migrate_data => TRUE);

ALTER TABLE telemetry ADD PRIMARY KEY (id, ts);

-- включаем сжатие, сегментируя по device_id (все запросы идут по одному устройству)
ALTER TABLE telemetry SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_id'
);

-- сжимаем куски старше 7 дней
SELECT add_compression_policy('telemetry', INTERVAL '7 days', if_not_exists => TRUE);
