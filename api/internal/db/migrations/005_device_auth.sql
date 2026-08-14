-- Аутентификация устройств на WebSocket-хендшейке + зоны для RBAC.
-- device_token_hash — sha256(токена), сам токен нигде не храним.

ALTER TABLE devices ADD COLUMN IF NOT EXISTS device_token_hash text;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS zone text NOT NULL DEFAULT 'home';
