defmodule Gateway.MQTT do
  use GenServer

  # Интеграция со стандартным MQTT-брокером (EMQX/Mosquitto). Подписывается на
  # топики сторонних беспроводных шлюзов (Zigbee/Tuya) и перекладывает сообщения
  # в общую шину гейтвея: рассылает дашбордам + шлёт в api на сохранение.
  #
  # Включение/настройка через env: MQTT_ENABLED, MQTT_HOST, MQTT_PORT, MQTT_TOPICS.

  def start_link(opts) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @impl true
  def init(opts) do
    enabled = Keyword.get(opts, :enabled, Application.get_env(:gateway, :mqtt_enabled, false))

    if enabled do
      host = Application.get_env(:gateway, :mqtt_host, "localhost")
      port = Application.get_env(:gateway, :mqtt_port, 1883)
      topics = Application.get_env(:gateway, :mqtt_topics, ["iot/+/telemetry"])

      Tortoise.Supervisor.start_child(
        client_id: "iot-gateway-" <> suffix(),
        handler: {Gateway.MQTT.Handler, []},
        server: {Tortoise.Transport.Tcp, host: host, port: port},
        subscriptions: Enum.map(topics, &{&1, 0})
      )
    end

    {:ok, %{enabled: enabled}}
  end

  # client_id в mqtt должен быть уникальным, иначе брокер рвёт старое соединение
  defp suffix do
    :crypto.strong_rand_bytes(4) |> Base.encode16(case: :lower)
  end
end

defmodule Gateway.MQTT.Handler do
  @behaviour Tortoise.Handler

  @impl true
  def init(args), do: {:ok, args}

  # топик: iot/<device_id>/telemetry. payload — json от шлюза (или сырое значение)
  @impl true
  def handle_message(["iot", device_id, "telemetry"], payload, state) do
    map =
      case Jason.decode(payload) do
        {:ok, m} when is_map(m) -> m
        _ -> %{"value" => payload}
      end

    full = Map.put(map, "device_id", device_id)
    json = Jason.encode!(full)

    Gateway.Registry.broadcast(device_id, {:telemetry, json})
    Gateway.Ingest.push(full)

    {:ok, state}
  end

  def handle_message(_topic, _payload, state), do: {:ok, state}

  @impl true
  def connection(_status, state), do: {:ok, state}

  @impl true
  def subscription(_status, _topic_filter, state), do: {:ok, state}

  @impl true
  def terminate(_reason, _state), do: :ok
end
