import Config

config :gateway,
  port: String.to_integer(System.get_env("GATEWAY_PORT", "4000")),
  api_url: System.get_env("API_URL", "http://localhost:8080"),
  ingest_token: System.get_env("INGEST_TOKEN", "dev-ingest-token"),
  parser_path:
    System.get_env("PARSER_PATH", Path.expand("../../parser/target/release/iot-parser", __DIR__))
