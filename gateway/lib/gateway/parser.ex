defmodule Gateway.Parser do
  use GenServer

  # держим один долгоживущий Port к rust-бинари. на каждый кадр шлём байты в stdin,
  # читаем одну строку json в ответ. очередь нужна, чтобы сопоставить ответы с запросами,
  # если сенсоры шлют пачкой

  @timeout 10_000

  def start_link(_) do
    GenServer.start_link(__MODULE__, nil, name: __MODULE__)
  end

  def parse(binary) do
    GenServer.call(__MODULE__, {:parse, binary}, @timeout)
  end

  @impl true
  def init(_) do
    exe = parser_path()

    port =
      if File.regular?(exe) do
        Port.open({:spawn_executable, exe}, [:binary, :exit_status])
      else
        nil
      end

    {:ok, %{port: port, buf: "", pending: :queue.new()}}
  end

  @impl true
  def handle_call({:parse, _binary}, _from, %{port: nil} = state) do
    {:reply, {:error, "парсер не найден, собери его (cargo build --release)"}, state}
  end

  def handle_call({:parse, binary}, from, state) do
    Port.command(state.port, binary)
    {:noreply, %{state | pending: :queue.in(from, state.pending)}}
  end

  @impl true
  def handle_info({_port, {:data, data}}, state) do
    buf = state.buf <> data
    {rest, pending} = drain(buf, state.pending)
    {:noreply, %{state | buf: rest, pending: pending}}
  end

  def handle_info({_port, {:exit_status, _status}}, state) do
    # парсер умер — отвечаем ошибкой всем, кто ждал, и поднимаем новый порт
    Enum.each(:queue.to_list(state.pending), fn from ->
      GenServer.reply(from, {:error, "парсер упал"})
    end)

    exe = parser_path()

    port =
      if File.regular?(exe) do
        Port.open({:spawn_executable, exe}, [:binary, :exit_status])
      else
        nil
      end

    {:noreply, %{port: port, buf: "", pending: :queue.new()}}
  end

  # из буфера достаём полные строки (по \n) и отвечаем на соответствующее число запросов.
  # хвост без \n оставляем ждать следующего куска
  defp drain(buf, pending) do
    case :binary.split(buf, "\n", [:global]) do
      [single] ->
        {single, pending}

      parts ->
        {rest, lines} = List.pop_at(parts, -1)
        froms = :queue.to_list(pending)

        Enum.zip(lines, froms)
        |> Enum.each(fn {line, from} -> GenServer.reply(from, {:ok, line}) end)

        remaining = :queue.from_list(Enum.drop(froms, length(lines)))
        {rest, remaining}
    end
  end

  defp parser_path do
    base = Application.get_env(:gateway, :parser_path, "iot-parser")

    cond do
      File.regular?(base) -> base
      # на винде собранный бинарь лежит с .exe, а в конфиге путь без расширения
      File.regular?(base <> ".exe") -> base <> ".exe"
      true -> base
    end
  end
end
