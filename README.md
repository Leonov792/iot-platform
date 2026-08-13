# 🏠 IoT-платформа для умного дома

Распределённая платформа для управления умным домом: сбор телеметрии с датчиков, real-time шина, управление устройствами и дашборды. Написана на Go, Elixir и Rust, с web-интерфейсом на Vue и нативными мобильными клиентами.

<p align="center">
  <img src="https://img.shields.io/github/actions/workflow/status/Leonov792/iot-platform/ci.yml?branch=master&label=CI&logo=github&style=flat-square" alt="CI">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white&style=flat-square" alt="Go">
  <img src="https://img.shields.io/badge/Elixir-1.20-4B275F?logo=elixir&logoColor=white&style=flat-square" alt="Elixir">
  <img src="https://img.shields.io/badge/Rust-1.96-DEA584?logo=rust&logoColor=white&style=flat-square" alt="Rust">
  <img src="https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vuedotjs&logoColor=white&style=flat-square" alt="Vue">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/platforms-linux%20%7C%20macOS%20%7C%20windows%20%7C%20raspberry%20pi-blue?style=flat-square" alt="Platforms">
</p>

## Архитектура

```
                    ┌─────────────┐   SQL    ┌──────────┐
 REST / JWT / CRUD  │   Go API    ├──────────┤ Postgres │
 (web + mobile)     └──────┬──────┘          └──────────┘
                           │ HTTP (команды)
 датчик ──binary WS──▶ ┌───▼────────┐  subprocess  ┌──────────┐
 эмулятор              │Elixir гейтвей├─────────────▶│Rust парсер│
                       │  (PubSub)   │◀────JSON────└──────────┘
                       └──────┬──────┘
                              │ broadcast (JSON)
                              ▼
                    ┌──────────────────┐
                    │ Vue / iOS / Android │
                    └──────────────────┘
```

| Сервис | Стек | Что делает |
|---|---|---|
| `api/` | Go, chi, pgx | REST API, JWT-авторизация, CRUD устройств, история телеметрии, команды |
| `gateway/` | Elixir, Bandit | WebSocket-шина, PubSub, держит тысячи соединений датчиков |
| `parser/` | Rust | Разбор тяжёлых бинарных пакетов телеметрии |
| `web/` | Vue 3, Tailwind, ECharts | Дашборд: логин, устройства, real-time графики |
| `mobile/ios` · `mobile/android` | SwiftUI · Kotlin/Compose | Нативные мобильные клиенты |
| `tools/sensor-emu` | Go | Эмулятор датчика для живой демки |

## Фичи

- Регистрация/логин с JWT (bcrypt)
- Устройства умного дома: лампа, розетка, термостат, датчик — с комнатами и состоянием
- Управление в реальном времени: вкл/выкл, яркость, целевая температура
- Live-графики телеметрии (температура, влажность, заряд)
- История телеметрии в Postgres
- Бинарный протокол телеметрии с CRC, парсинг на Rust

## Быстрый старт (docker)

```bash
docker compose up --build
```

| Сервис | Адрес |
|---|---|
| Фронтенд | http://localhost |
| Go API | http://localhost:8080 |
| Гейтвей (WS) | ws://localhost:4000 |

Затем зарегистрируйся на фронте, создай устройство (`sensor`) и запусти эмулятор:

```bash
cd tools/sensor-emu
go run . -id sensor-1 -type sensor -url ws://localhost:4000/ws/device/
```

## Разработка (вручную)

Требуется: Go 1.25+, Elixir 1.20+, Rust, Node 22, Postgres 16.

```bash
# 1. база
docker run -d --name iot-pg -e POSTGRES_USER=iot -e POSTGRES_PASSWORD=iot -e POSTGRES_DB=iot -p 5432:5432 postgres:16

# 2. go api
cd api && go run ./cmd/api

# 3. rust парсер
cd parser && cargo build --release

# 4. elixir гейтвей
cd gateway && mix deps.get
PARSER_PATH=../parser/target/release/iot-parser mix run --no-halt

# 5. фронт
cd web && npm install && npm run dev
```

Или коротко через `make`: `make api` / `make gw` / `make web` / `make emu` / `make test`.

## Тесты

```bash
make test
# или по отдельности:
cd api && go test ./...
cd parser && cargo test
cd gateway && mix test
cd web && npm test
```

CI гоняет всё это на каждый push и PR (см. `.github/workflows/ci.yml`): `go vet`, `cargo fmt` + `clippy`, `mix format --check`, `npm test` + build.

## API

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/auth/register` | Регистрация → JWT |
| POST | `/api/v1/auth/login` | Вход → JWT |
| GET/POST | `/api/v1/devices` | Список / создание устройства (JWT) |
| PUT/DELETE | `/api/v1/devices/{id}` | Обновление / удаление (JWT) |
| POST | `/api/v1/devices/{id}/command` | Команда (вкл/выкл/яркость/температура) |
| GET | `/api/v1/devices/{id}/telemetry` | История для графика |
| POST | `/api/v1/telemetry` | Ингест телеметрии (заголовок `X-Ingest-Token`) |

WebSocket гейтвея:

| Путь | Назначение |
|---|---|
| `GET /ws/device/{id}` | Сенсор шлёт сюда бинарные кадры |
| `GET /ws/dashboard` | Фронт/мобилка подписываются на телеметрию |

## Бинарный протокол

```
[magic:2 = 0xAB 0xCD][ver:1][device_id:16][kind:1][payload_len:2 BE][payload:N][crc:1]
kind = 0x01 → телеметрия: temp(f32 LE), humidity(f32 LE), battery(u8)
```

## Платформы и релизы

Нативные бинарники собираются под **Windows / Linux / macOS / Raspberry Pi**:

| Платформа | Цели |
|---|---|
| Linux | `amd64`, `arm64`, `armv7` (RPi 2/Zero 2) |
| macOS | `amd64` (Intel), `arm64` (Apple Silicon) |
| Windows | `amd64` |

Сборка релизов — по тегу `v*` (см. `.github/workflows/release.yml`): goreleaser для Go, `cargo-zigbuild` для Rust, multi-arch docker-образ гейтвея.

## Структура репозитория

```
iot-platform/
├── api/            # Go: REST API + авторизация + БД
├── gateway/        # Elixir: WebSocket-шина + PubSub
├── parser/         # Rust: парсер бинарной телеметрии
├── web/            # Vue 3: дашборд
├── mobile/
│   ├── ios/        # SwiftUI-клиент
│   └── android/    # Kotlin/Compose-клиент
├── tools/sensor-emu/  # эмулятор датчика
├── docker-compose.yml
└── .github/workflows/  # CI + Release
```

## Лицензия

[MIT](./LICENSE)
