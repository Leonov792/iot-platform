defmodule Gateway.Registry do
  use GenServer

  # держим два мапа: sensors (device_id -> pid сенсора) и subs (device_id -> [pid подписчиков]).
  # этс было бы быстрее, но тут связей немного, обычный genserver проще читать

  def start_link(_) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  @impl true
  def init(_), do: {:ok, %{sensors: %{}, subs: %{}}}

  def register_sensor(id, pid), do: GenServer.call(__MODULE__, {:register_sensor, id, pid})
  def subscribe(id, pid), do: GenServer.call(__MODULE__, {:subscribe, id, pid})
  def sensor_pid(id), do: GenServer.call(__MODULE__, {:sensor_pid, id})
  def unregister(pid, sock_state), do: GenServer.call(__MODULE__, {:unregister, pid, sock_state})
  def broadcast(id, msg), do: GenServer.cast(__MODULE__, {:broadcast, id, msg})

  @impl true
  def handle_call({:register_sensor, id, pid}, _from, state) do
    {:reply, :ok, put_in(state, [:sensors, id], pid)}
  end

  def handle_call({:subscribe, id, pid}, _from, state) do
    list = Map.get(state.subs, id, [])
    subs = if pid in list, do: list, else: [pid | list]
    {:reply, :ok, put_in(state, [:subs, id], subs)}
  end

  def handle_call({:sensor_pid, id}, _from, state) do
    {:reply, Map.get(state.sensors, id), state}
  end

  def handle_call({:unregister, pid, sock_state}, _from, state) do
    # снять pid как сенсор, если был
    sensors =
      case sock_state[:role] do
        :sensor ->
          id = sock_state[:device_id]

          if Map.get(state.sensors, id) == pid,
            do: Map.delete(state.sensors, id),
            else: state.sensors

        _ ->
          state.sensors
      end

    # выкинуть pid из всех подписок
    subs =
      Enum.reduce(state.subs, %{}, fn {id, pids}, acc ->
        Map.put(acc, id, Enum.reject(pids, &(&1 == pid)))
      end)

    {:reply, :ok, %{state | sensors: sensors, subs: subs}}
  end

  @impl true
  def handle_cast({:broadcast, id, msg}, state) do
    for pid <- Map.get(state.subs, id, []) do
      send(pid, msg)
    end

    {:noreply, state}
  end
end
