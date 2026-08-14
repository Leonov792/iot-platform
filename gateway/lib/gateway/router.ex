defmodule Gateway.Router do
  use Plug.Router

  plug(Plug.Logger)
  plug(:match)
  plug(:dispatch)

  # сенсор цепляется сюда и шлёт бинарные кадры.
  # до апгрейда проверяем device token, чтобы исключить подмену данных.
  # в локальной разработке проверку можно выключить через DEVICE_AUTH_ENABLED=false
  get "/ws/device/:id" do
    id = conn.path_params["id"]

    if Gateway.DeviceAuth.enabled?() do
      case Gateway.DeviceAuth.verify(id, Gateway.DeviceAuth.token_from_conn(conn)) do
        :ok ->
          upgrade_sensor(conn, id)

        {:error, reason} ->
          conn
          |> send_resp(401, Jason.encode!(%{"error" => reason}))
          |> halt()
      end
    else
      upgrade_sensor(conn, id)
    end
  end

  defp upgrade_sensor(conn, id) do
    conn
    |> WebSockAdapter.upgrade(Gateway.Socket, %{role: :sensor, device_id: id, subs: []},
      timeout: 60_000
    )
    |> halt()
  end

  # фронт цепляется сюда, чтобы читать телеметрию в реалтайме
  get "/ws/dashboard" do
    conn
    |> WebSockAdapter.upgrade(Gateway.Socket, %{role: :dashboard, subs: []}, timeout: 60_000)
    |> halt()
  end

  # go api шлёт команды на устройство через этот эндпоинт
  post "/internal/command" do
    {:ok, body, conn} = Plug.Conn.read_body(conn)

    case Jason.decode(body) do
      {:ok, %{"device_id" => id, "action" => action} = cmd} ->
        payload = Jason.encode!(%{"action" => action, "value" => Map.get(cmd, "value")})

        case Gateway.Registry.sensor_pid(id) do
          nil ->
            send_resp(conn, 404, Jason.encode!(%{"error" => "устройство не подключено"}))

          pid ->
            send(pid, {:command, payload})
            send_resp(conn, 200, Jason.encode!(%{"status" => "отправлено"}))
        end

      _ ->
        send_resp(conn, 400, Jason.encode!(%{"error" => "кривой json"}))
    end
  end

  get "/health" do
    send_resp(conn, 200, "ok")
  end

  match _ do
    send_resp(conn, 404, "not found")
  end
end
