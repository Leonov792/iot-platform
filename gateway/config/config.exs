import Config

config :gateway,
  port: String.to_integer(System.get_env("GATEWAY_PORT", "4000")),
  api_url: System.get_env("API_URL", "http://localhost:8080"),
  ingest_token: System.get_env("INGEST_TOKEN", "dev-ingest-token"),
  device_auth_enabled: System.get_env("DEVICE_AUTH_ENABLED", "true") == "true",
  mqtt_enabled: System.get_env("MQTT_ENABLED", "false") == "true",
  mqtt_host: System.get_env("MQTT_HOST", "localhost"),
  mqtt_port: String.to_integer(System.get_env("MQTT_PORT", "1883")),
  mqtt_topics: String.split(System.get_env("MQTT_TOPICS", "iot/+/telemetry"), ",")
