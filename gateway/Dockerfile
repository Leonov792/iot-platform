# гейтвей с Rustler NIF: для сборки нужен и rust (NIF), и elixir.
# rustler-компилятор запускает cargo для крейта gateway_parser (лежит в ../parser).

FROM elixir:1.20-slim

# CA-серты для hex.pm + инструменты для сборки нативного кода
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl build-essential \
    && rm -rf /var/lib/apt/lists/* \
    && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal --default-toolchain stable

ENV PATH="/root/.cargo/bin:${PATH}"

WORKDIR /app

# отдельно копируем манифесты, чтобы deps кешировались в слое
COPY gateway/mix.exs gateway/mix.lock ./
RUN mix deps.get

# исходники rust-крейта для NIF (mix compile соберёт его из ../parser)
COPY parser/ /parser/

COPY gateway/ ./
RUN mix compile

ENV MIX_ENV=prod
EXPOSE 4000
CMD ["mix", "run", "--no-halt"]
