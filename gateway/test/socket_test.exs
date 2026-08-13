defmodule Gateway.SocketTest do
  use ExUnit.Case, async: false

  # приложение стартует в тестах, поэтому Registry живой — подписка отработает

  test "dashboard подписывается на устройство" do
    state = %{role: :dashboard, subs: []}

    {:ok, new_state} =
      Gateway.Socket.handle_in(
        {~s({"type":"subscribe","device_id":"sensor-1"}), [opcode: :text]},
        state
      )

    assert "sensor-1" in new_state.subs
  end

  test "dashboard отписывается от устройства" do
    state = %{role: :dashboard, subs: ["sensor-1"]}

    {:ok, new_state} =
      Gateway.Socket.handle_in(
        {~s({"type":"unsubscribe","device_id":"sensor-1"}), [opcode: :text]},
        state
      )

    refute "sensor-1" in new_state.subs
  end

  test "телеметрия пушится подписчику" do
    state = %{role: :dashboard, subs: []}

    assert {:push, {:text, "{}"}, ^state} = Gateway.Socket.handle_info({:telemetry, "{}"}, state)
  end

  test "команда уходит только сенсору, не дашборду" do
    sensor = %{role: :sensor, device_id: "sensor-1", subs: []}
    payload = ~s({"action":"on"})

    assert {:push, {:text, ^payload}, ^sensor} = Gateway.Socket.handle_info({:command, payload}, sensor)

    dashboard = %{role: :dashboard, subs: []}
    assert {:ok, ^dashboard} = Gateway.Socket.handle_info({:command, payload}, dashboard)
  end
end
