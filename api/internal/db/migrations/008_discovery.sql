-- Обнаруженные в сети устройства (сканер подсети: Modbus 502 / MQTT 1883).

CREATE TABLE IF NOT EXISTS discovered_devices (
    id         bigserial PRIMARY KEY,
    mac        text,
    ip         text NOT NULL,
    port       int  NOT NULL,
    service    text NOT NULL,                       -- modbus | mqtt
    status     text NOT NULL DEFAULT 'pending',     -- pending | approved | ignored
    first_seen timestamptz NOT NULL DEFAULT now(),
    last_seen  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_discovered_ip_port ON discovered_devices (ip, port);
