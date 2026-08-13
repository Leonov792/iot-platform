defmodule Gateway.Socket do
  @behaviour WebSock

  @impl true
  def init(%{role: :sensor, device_id: id} = state) do
    Gateway.Registry.register_sensor(id, self())
    {:ok, state}
  end

  def init(state), do: {:ok, state}

  @impl true
  def handle_in({data, [opcode: :binary]}, %{role: :sensor, device_id: id} = state) do
    # сырой кадр от датчика -> rust парсит -> json -> раздать подписчикам + в api
    with {:ok, json} <- Gateway.Parser.parse(data),
         {:ok, map} <- Jason.decode(json) do
      Gateway.Registry.broadcast(id, {:telemetry, json})
      Gateway.Ingest.push(map)
    else
      _ -> :ok
    end

    {:ok, state}
  end

  def handle_in({data, [opcode: :text]}, %{role: :dashboard} = state) do
    case Jason.decode(data) do
      {:ok, %{"type" => "subscribe", "device_id" => id}} ->
        Gateway.Registry.subscribe(id, self())
        {:ok, %{state | subs: [id | state.subs]}}

      {:ok, %{"type" => "unsubscribe", "device_id" => id}} ->
        {:ok, %{state | subs: List.delete(state.subs, id)}}

      _ ->
        {:ok, state}
    end
  end

  def handle_in(_frame, state), do: {:ok, state}

  @impl true
  def handle_info({:telemetry, json}, state) do
    {:push, {:text, json}, state}
  end

  def handle_info({:command, payload}, %{role: :sensor} = state) do
    # команда с фронта доехала до сенсора через go api -> гейтвей
    {:push, {:text, payload}, state}
  end

  def handle_info(_msg, state), do: {:ok, state}

  @impl true
  def terminate(_reason, state) do
    Gateway.Registry.unregister(self(), state)
    :ok
  end
end
