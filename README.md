# IoT-платформа для умного дома

Четыре сервиса: Go API (REST + авторизация + БД), Elixir-гейтвей (WebSocket-шина),
Rust-парсер (бинарная телеметрия), Vue-фронтенд (дашборд).

```
датчик ──binary WS──▶ Elixir гейтвей ──subprocess──▶ Rust парсер ──JSON──▶ шина
                                                                          │
REST / логин / CRUD ──▶ Go API ──SQL──▶ Postgres          ◀── WS подписка ──┘
                                                                          │
                                                                  Vue-фронтенд
```

## Запуск (docker)

```bash
docker compose up --build
```

- фронтенд: http://localhost
- api: http://localhost:8080
- гейтвей (ws): ws://localhost:4000

Потом зарегистрируйся на фронте, создай устройство (тип `sensor`), и запусти эмулятор:

```bash
cd tools/sensor-emu
go run . -id sensor-1 -type sensor -url ws://localhost:4000/ws/device/
```

График на странице устройства начнёт шевелиться.

## Запуск вручную (dev)

1. **Postgres**: `docker run -d --name iot-pg -e POSTGRES_USER=iot -e POSTGRES_PASSWORD=iot -e POSTGRES_DB=iot -p 5432:5432 postgres:16`
2. **Go API**: `cd api && go run ./cmd/api`
3. **Rust парсер**: `cd parser && cargo build --release`
4. **Elixir гейтвей**: `cd gateway && mix deps.get && PARSER_PATH=../parser/target/release/iot-parser mix run --no-halt`
5. **Фронтенд**: `cd web && npm install && npm run dev`

## Основные эндпоинты

- `POST /api/v1/auth/register`, `POST /api/v1/auth/login` — регистрация/вход (JWT)
- `GET/POST /api/v1/devices`, `PUT/DELETE /api/v1/devices/{id}` — CRUD устройств (JWT)
- `POST /api/v1/devices/{id}/command` — команда на устройство (вкл/выкл/яркость/температура)
- `GET /api/v1/devices/{id}/telemetry` — история для графика
- `POST /api/v1/telemetry` — ингест телеметрии от гейтвея (заголовок `X-Ingest-Token`)
- `GET /ws/device/{id}` — сенсор шлёт сюда бинарные кадры
- `GET /ws/dashboard` — фронт подписывается на телеметрию

## Формат бинарного пакета

```
[magic:2 = 0xAB 0xCD][ver:1][device_id:16][kind:1][payload_len:2 BE][payload:N][crc:1]
kind=0x01 → телеметрия: temp(f32 LE), humidity(f32 LE), battery(u8)
```
