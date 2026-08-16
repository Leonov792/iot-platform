defmodule Gateway.Discovery do
  use GenServer

  require Logger

  # Сканер локальной подсети: раз в DISCOVERY_INTERVAL_MS (дефолт 5 мин) пробует
  # подключиться к стандартным портам (Modbus 502, MQTT 1883) и шлёт найденные
  # устройства в go api. Включается через DISCOVERY_ENABLED=true.
  #
  # MAC-адрес не снимаем (для этого нужен nmap/root) — уникальность по ip+port.

  @ports [{"modbus", 502}, {"mqtt", 1883}]
  @max_hosts 1024

  def start_link(opts) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @impl true
  def init(_opts) do
    enabled = Application.get_env(:gateway, :discovery_enabled, false)
    interval = Application.get_env(:gateway, :discovery_interval_ms, 300_000)

    if enabled do
      send(self(), :scan)
    end

    {:ok, %{enabled: enabled, interval: interval}}
  end

  @impl true
  def handle_info(:scan, %{enabled: true} = state) do
    subnet = Application.get_env(:gateway, :discovery_subnet, "192.168.1.0/24")

    case ips(subnet) do
      {:ok, hosts} ->
        hosts
        |> Task.async_stream(&check_host/1, max_concurrency: 50, timeout: 5_000)
        |> Stream.run()

      {:error, reason} ->
        Logger.warning("discovery: не разобрал подсеть #{subnet}: #{reason}")
    end

    Process.send_after(self(), :scan, state.interval)
    {:noreply, state}
  end

  def handle_info(_msg, state), do: {:noreply, state}

  # проверить один хост на все порты
  defp check_host(ip) do
    Enum.each(@ports, fn {service, port} ->
      case :gen_tcp.connect(String.to_charlist(ip), port, [:inet], 300) do
        {:ok, sock} ->
          :gen_tcp.close(sock)
          report(ip, port, service)

        {:error, _} ->
          :ok
      end
    end)
  end

  # отправить найденное устройство в go api (upsert, не критично при ошибке)
  defp report(ip, port, service) do
    api_url = Application.get_env(:gateway, :api_url, "http://localhost:8080")
    token = Application.get_env(:gateway, :ingest_token, "dev-ingest-token")
    body = Jason.encode!(%{"ip" => ip, "port" => port, "service" => service})

    req =
      Finch.build(
        :post,
        api_url <> "/internal/discovered",
        [{"content-type", "application/json"}, {"x-ingest-token", token}],
        body
      )

    _ = Finch.request(req, Gateway.Finch)
  end

  @doc "CIDR -> список IPv4 хостов (без адреса сети и бродкаста)."
  def ips(cidr) do
    with [base, prefix_s] <- String.split(cidr, "/"),
         {prefix, ""} <- Integer.parse(prefix_s),
         true <- prefix >= 0 and prefix <= 30,
         {:ok, base_int} <- ip_to_int(base) do
      host_bits = 32 - prefix
      host_count = Bitwise.bsl(1, host_bits)
      net_mask = Bitwise.band(Bitwise.bnot(host_count - 1), 0xFFFF_FFFF)
      start = Bitwise.band(base_int, net_mask)
      count = host_count - 2

      cond do
        count <= 0 -> {:error, "подсеть слишком мала"}
        count > @max_hosts -> {:error, "подсеть слишком велика (#{count} хостов)"}
        true -> {:ok, Enum.map(1..count, fn i -> int_to_ip(start + i) end)}
      end
    else
      _ -> {:error, "не CIDR"}
    end
  end

  defp ip_to_int(ip) do
    case String.split(ip, ".") do
      [a, b, c, d] ->
        with {na, ""} <- Integer.parse(a),
             {nb, ""} <- Integer.parse(b),
             {nc, ""} <- Integer.parse(c),
             {nd, ""} <- Integer.parse(d),
             true <- Enum.all?([na, nb, nc, nd], &(&1 >= 0 and &1 <= 255)) do
          {:ok, Bitwise.bsl(na, 24) + Bitwise.bsl(nb, 16) + Bitwise.bsl(nc, 8) + nd}
        else
          _ -> {:error, "не ipv4"}
        end

      _ ->
        {:error, "не ipv4"}
    end
  end

  defp int_to_ip(n) do
    a = Bitwise.band(Bitwise.bsr(n, 24), 0xFF)
    b = Bitwise.band(Bitwise.bsr(n, 16), 0xFF)
    c = Bitwise.band(Bitwise.bsr(n, 8), 0xFF)
    d = Bitwise.band(n, 0xFF)
    "#{a}.#{b}.#{c}.#{d}"
  end
end
