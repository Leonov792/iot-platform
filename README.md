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

Затем зарегистрируйся на фронте, создай устройство (`sensor`), выпусти для него device token
(через `POST /api/v1/devices/{id}/token`) и запусти эмулятор с этим токеном:

```bash
cd tools/sensor-emu
go run . -id sensor-1 -type sensor -token <DEVICE_TOKEN> -url ws://localhost:4000/ws/device/
```

Device token обязателен: гейтвей проверяет его на WebSocket-хендшейке. Для локальной
разработки проверку можно выключить флагом `DEVICE_AUTH_ENABLED=false` (env гейтвея).

## Разработка (вручную)

Требуется: Go 1.25+, Elixir 1.20+, Rust, Node 22, Postgres 16.

```bash
# 1. база
docker run -d --name iot-pg -e POSTGRES_USER=iot -e POSTGRES_PASSWORD=iot -e POSTGRES_DB=iot -p 5432:5432 postgres:16

# 2. go api
cd api && go run ./cmd/api

# 3. elixir гейтвей (rustler сам соберёт rust NIF из ../parser при mix compile)
cd gateway && mix deps.get && mix run --no-halt

# 4. фронт
cd web && npm install && npm run dev
```

Или коротко через `make`: `make api` / `make gw` / `make web` / `make emu` / `make test`.

## Боевое развёртывание (On-Premise)

Пошаговый runbook для закрытого контура (дом/бассейн/спортзал). Ядро поднимается
через `docker compose`, промышленные фичи — через конфиги и env.

### 1. Секреты и сеть

Скопируй `.env.example` в `.env` и замени dev-значения на реальные:

- `JWT_SECRET` — длинная случайная строка (подпись токенов);
- `INGEST_TOKEN` — общий секрет между API и гейтвеем (и Modbus/автоматизацией/алертами);
- device token — выпускается отдельно на каждое устройство (см. ниже).

Порты наружу: API `8080`, гейтвей `4000`, web `80`, modbus-poller `8090`,
automation `8091`, ai `8095`, Ollama `11434`, MQTT `1883`.

### 2. База (TimescaleDB)

Поднимается в compose автоматически. Миграции накатывает API при старте
(гипертаблица + сжатие телеметрии старше 7 дней). Для реальных объёмов повесь
бэкап на `pgdata`-том.

### 3. Реальные датчики по Modbus TCP

IP/регистры устройств бассейна задаются в `modbus.json` (см. `modbus.example.json`):

```jsonc
{
  "api_url": "http://api:8080",
  "ingest_token": "<INGEST_TOKEN>",
  "write_token": "<свой-токен-записи>",
  "devices": [
    {
      "id": "pool-pump",
      "host": "192.168.1.50",   // ← IP контроллера
      "port": 502,
      "unit": 1,
      "points": [
        { "name": "ph", "register": "input", "address": 0, "scale": 0.01 },
        { "name": "orp", "register": "input", "address": 1, "scale": 1.0 }
      ],
      "relays": [
        { "name": "fill_valve", "register": "coil", "address": 0 }
      ]
    }
  ]
}
```

Адреса регистров бери из документации контроллера. Проверка соединения:
`curl http://<хост>:8090/health` и `curl http://<хост>:8090/logs`.

### 4. Беспроводные датчики по MQTT

В env гейтвея: `MQTT_ENABLED=true`, `MQTT_HOST=<IP брокера>`, `MQTT_TOPICS=iot/+/telemetry`.
Шлюз (Zigbee/Tuya) должен слать JSON в топик `iot/<device_id>/telemetry` —
гейтвей переложит в шину и сохранит в API.

### 5. Автоматизация (rules engine)

Правила в `automation.json` (см. `automation.example.json`): условия по полям
телеметрии + действия `modbus_write`. Управляется на лету через
`PUT /v1/rules` с заголовком `X-Automation-Token: <rules_token>` (конструктор в
мобилке ходит сюда же).

### 6. Локальный ИИ (Ollama)

В compose `ollama` уже поднят. Вытяни модель и укажи её в env AI-сервиса:

```bash
docker compose exec ollama ollama pull llama3.1
# env: OLLAMA_MODEL=llama3.1, OLLAMA_URL=http://ollama:11434
```

Проверка: `curl http://<хост>:8095/v1/status`. Парсер намерений — `POST /v1/intent`
(`{"text":"подготовь спортзал и синюю подсветку"}`) → JSON-массив команд.

### 7. Алерты (Telegram / FCM)

Секреты задаются через env (переопределяют значение из `alerts.json`):

- Telegram: `TELEGRAM_TOKEN=<бот-токен>`, `TELEGRAM_CHAT=<chat_id>`;
- FCM: `FCM_CREDENTIALS=<путь к service-account.json>`, `FCM_TOPIC=iot-alerts`.

Пороги аварий — в `alerts.json` (`alerts.example.json`).

### 8. Мобилки под реальный IP

По умолчанию в коде зашиты `localhost` (iOS) и `10.0.2.2` (Android). Для боя
поменяй на IP сервера:
- Android: `mobile/android/app/build.gradle.kts` → `API_BASE_URL`, `WS_BASE_URL`;
- iOS: `mobile/ios/IoTHome/ContentView.swift` → `APIClient.base` и `TelemetryStream`.

### 9. Device token для устройств

Гейтвей проверяет token на WS-хендшейке. Выпусти его для каждого устройства
(owner, JWT):

```bash
curl -X POST http://<хост>:8080/api/v1/devices/<device-id>/token \
  -H "Authorization: Bearer <JWT>"
```

Полученный токен передай устройству/эмулятору (`sensor-emu -token <token>`).
Для локальной разработки проверку можно отключить: `DEVICE_AUTH_ENABLED=false`.

### 10. Логирование (обязательное требование)

Во всех Go-сервисах (`api`, `modbus-poller`, `automation`, `alerts`, `ai`) и в
эмуляторе используется **структурный логгер `log/slog`** (стандартная библиотека)
с JSON-выводом:

```go
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
```

Требования:
- Стандартный пакет `log` для продакшена **запрещён** — только `slog` (или `zap`).
- Вывод — структурный JSON (машиночитаемый, для сбора в Loki/ELK и т.п.).
- Ошибки логируются как `slog.Error(..., "err", err)`, предупреждения — `slog.Warn`,
  инфо — `slog.Info`. Фатальные — `slog.Error` + `os.Exit(1)`.
- Поля — именованные пары `ключ=значение`, без сшивания строк в сообщении.

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
| PUT/DELETE | `/api/v1/devices/{id}` | Обновление / удаление (JWT, owner) |
| POST | `/api/v1/devices/{id}/command` | Команда (вкл/выкл/яркость/температура/цвет) |
| GET | `/api/v1/devices/{id}/telemetry` | История для графика |
| POST | `/api/v1/devices/{id}/token` | Выпуск device token (owner) |
| GET/POST | `/api/v1/users` | Члены дома (owner) |
| PUT | `/api/v1/users/{id}/role` | Смена роли (owner) |
| PUT | `/api/v1/users/{id}/schedule` | Расписание доступа (owner) |
| POST | `/api/v1/telemetry` | Ингест телеметрии (заголовок `X-Ingest-Token`) |
| POST | `/internal/device/{id}/verify` | Проверка device token (гейтвей) |
| GET | `/internal/telemetry/{id}/latest` | Последняя телеметрия (автоматизация/ИИ) |

WebSocket гейтвея:

| Путь | Назначение |
|---|---|
| `GET /ws/device/{id}` | Сенсор шлёт сюда бинарные кадры (device token) |
| `GET /ws/dashboard` | Фронт/мобилка подписываются на телеметрию |

## Драйверы протоколов (Modbus TCP / MQTT)

**Modbus TCP** — оборудование бассейна/спортзала (насосы фильтрации, дозаторы хлора/pH,
клапаны долива, вытяжка). Живёт в `api/cmd/modbus-poller`:

- читает Holding/Input Registers и Coils (датчики pH, ORP, уровня воды, влажности);
- пишет реле/клапаны через `POST /internal/write` (`X-Write-Token`);
- отдаёт телеметрию в общую шину (ingest-ручка API);
- отдаёт лог операций на `GET /logs` (для админки).

Конфиг — JSON-файл (`MODBUS_CONFIG`, пример `api/cmd/modbus-poller/modbus.example.json`):
устройства, точки (register/address/scale) и реле.

**MQTT** — беспроводные датчики через сторонние шлюзы (Zigbee/Tuya). Встроен в гейтвей
на базе `tortoise` (`gateway/lib/gateway/mqtt.ex`):

- подписывается на топики `iot/+/telemetry` (настраивается через `MQTT_TOPICS`);
- сообщения шлюза перекладываются в ту же PubSub-шину + сохраняются в API;
- включается через `MQTT_ENABLED=true`, брокер — `MQTT_HOST`/`MQTT_PORT` (EMQX/Mosquitto).

## B2B/Premium сервисы

| Сервис | Стек | Что делает |
|---|---|---|
| `automation/` | Go | Rules Engine: `IF [Condition] THEN [Action]` (долив воды, хлор/pH, климат) |
| `alerts/` | Go | Push-уведомления (FCM + Telegram) о критических авариях |
| `ai/` | Go | Ollama-клиент, парсинг намерений в JSON, предиктивный анализ |
| `api/cmd/modbus-poller` | Go | Драйвер Modbus TCP (датчики химии бассейна, реле/клапаны) |
| `gateway` | Elixir | + MQTT-клиент (tortoise) для Zigbee/Tuya-шлюзов |

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
├── api/            # Go: REST API + RBAC + БД (+ modbus-poller)
├── gateway/        # Elixir: WebSocket-шина + PubSub + Rustler NIF + MQTT
├── parser/         # Rust: парсер бинарной телеметрии (lib для NIF + bin)
├── automation/     # Go: rules engine
├── alerts/         # Go: уведомления (FCM + Telegram)
├── ai/             # Go: локальный ИИ (Ollama)
├── web/            # Vue 3: дашборд + админка
├── mobile/
│   ├── ios/        # SwiftUI-клиент (color wheel, конструктор автоматизаций)
│   └── android/    # Kotlin/Compose-клиент
├── tools/sensor-emu/  # эмулятор датчика
├── docker-compose.yml
└── .github/workflows/  # CI + Release
```

## Лицензия

[MIT](./LICENSE)
