# этап 1: собираем rust-парсер
FROM rust:1.96-slim AS parser
WORKDIR /b
COPY parser/ ./parser/
RUN cd parser && cargo build --release

# этап 2: эликсир-гейтвей
FROM elixir:1.20-slim
WORKDIR /app

# отдельно копируем манифесты, чтобы deps кешировались в слое
COPY gateway/mix.exs gateway/mix.lock ./
RUN mix deps.get

COPY gateway/ ./
RUN mix compile

COPY --from=parser /b/parser/target/release/iot-parser /app/iot-parser

ENV MIX_ENV=prod
EXPOSE 4000
CMD ["mix", "run", "--no-halt"]
