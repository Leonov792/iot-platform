defmodule Gateway.Ingest do
  # шлём распарсенную телеметрию в go api, чтобы там была история для графика.
  # асинхронно и без ожидания — не хотим тормозить шину из-за того, что api отвалился

  def push(%{"device_id" => id} = map) do
    payload = Map.drop(map, ["device_id"])
    body = Jason.encode!(%{"device_id" => id, "payload" => payload})

    Task.start(fn ->
      api_url = Application.get_env(:gateway, :api_url, "http://localhost:8080")
      token = Application.get_env(:gateway, :ingest_token, "dev-ingest-token")

      req =
        Finch.build(
          :post,
          api_url <> "/api/v1/telemetry",
          [{"content-type", "application/json"}, {"x-ingest-token", token}],
          body
        )

      _ = Finch.request(req, Gateway.Finch)
    end)
  end

  def push(_), do: :ok
end
