# шпаргалка, чтобы не помнить все команды. tabs в make — зло, но что поделать

.PHONY: up down api gw web emu parser modbus automation alerts ai test

up:
	docker compose up --build

down:
	docker compose down

api:
	cd api && go run ./cmd/api

gw:
	cd gateway && mix run --no-halt

web:
	cd web && npm run dev

emu:
	cd tools/sensor-emu && go run . -id sensor-1 -type sensor

parser:
	cd parser && cargo build --release

modbus:
	cd api && go run ./cmd/modbus-poller

automation:
	cd automation && go run .

alerts:
	cd alerts && go run .

ai:
	cd ai && go run .

test:
	cd api && go test ./...
	cd parser && cargo test
	cd gateway && mix test
	cd automation && go test ./...
	cd alerts && go vet ./...
	cd ai && go test ./...
	cd web && npm test
