defmodule Gateway.DeviceAuth do
  # Проверяет device token на WebSocket-хендшейке, прежде чем пускать устройство в шину.
  # Гейтвей не знает самих токенов — он ходит в go api, где лежит только sha256.

  @timeout 5_000

  # включена ли проверка device token (env DEVICE_AUTH_ENABLED, дефолт true)
  def enabled? do
    Application.get_env(:gateway, :device_auth_enabled, true)
  end

  @spec verify(String.t(), String.t() | nil) :: :ok | {:error, String.t()}
  def verify(_device_id, token) when is_nil(token) or token == "",
    do: {:error, "отсутствует device token"}

  def verify(device_id, token) do
    api_url = Application.get_env(:gateway, :api_url, "http://localhost:8080")
    ingest_token = Application.get_env(:gateway, :ingest_token, "dev-ingest-token")

    req =
      Finch.build(
        :post,
        "#{api_url}/internal/device/#{device_id}/verify",
        [
          {"content-type", "application/json"},
          {"x-ingest-token", ingest_token},
          {"x-device-token", token}
        ],
        ""
      )

    case Finch.request(req, Gateway.Finch, receive_timeout: @timeout) do
      {:ok, %Finch.Response{status: 200}} ->
        :ok

      {:ok, %Finch.Response{status: 401}} ->
        {:error, "неверный device token"}

      {:ok, %Finch.Response{status: 404}} ->
        {:error, "устройство не найдено или токен не задан"}

      {:ok, %Finch.Response{status: code}} ->
        {:error, "api ответил #{code}"}

      {:error, reason} ->
        {:error, "api недоступен: #{inspect(reason)}"}
    end
  end

  # токен может прийти либо в заголовке X-Device-Token (нативный клиент),
  # либо в query-параметре ?token= (браузер/тест)
  @spec token_from_conn(Plug.Conn.t()) :: String.t() | nil
  def token_from_conn(conn) do
    case Plug.Conn.get_req_header(conn, "x-device-token") do
      [token | _] when token != "" -> token
      _ -> conn.query_params["token"]
    end
  end
end
