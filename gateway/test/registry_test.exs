defmodule Gateway.RegistryTest do
  use ExUnit.Case, async: false

  # приложение в тестах стартует целиком, поэтому Registry уже запущен.
  # тут не поднимаем его через start_supervised, а юзаем глобальный,
  # для чистоты берём уникальные id на каждый тест

  test "регистрирует сенсор и находит по id" do
    id = "dev-#{System.unique_integer([:positive])}"
    pid = self()

    :ok = Gateway.Registry.register_sensor(id, pid)
    assert Gateway.Registry.sensor_pid(id) == pid
    assert Gateway.Registry.sensor_pid("nope") == nil
  end

  test "подписка и рассылка" do
    id = "dev-#{System.unique_integer([:positive])}"

    :ok = Gateway.Registry.subscribe(id, self())
    Gateway.Registry.broadcast(id, {:telemetry, "{}"})

    assert_receive {:telemetry, "{}"}, 200
  end

  test "unregister снимает сенсор" do
    id = "dev-#{System.unique_integer([:positive])}"
    pid = self()

    :ok = Gateway.Registry.register_sensor(id, pid)
    :ok = Gateway.Registry.subscribe(id, pid)
    :ok = Gateway.Registry.unregister(pid, %{role: :sensor, device_id: id, subs: [id]})

    assert Gateway.Registry.sensor_pid(id) == nil
  end
end
