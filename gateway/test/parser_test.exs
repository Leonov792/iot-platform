defmodule Gateway.ParserTest do
  use ExUnit.Case, async: false

  defp telemetry_frame(id) do
    idb = String.pad_trailing(id, 16, <<0>>)
    payload = <<21.5::little-float-32, 40.0::little-float-32, 87>>
    frame = <<0xAB, 0xCD, 1, idb::binary, 0x01, 9::16-big, payload::binary>>

    crc =
      frame
      |> binary_part(2, byte_size(frame) - 2)
      |> :binary.bin_to_list()
      |> Enum.reduce(0, &Bitwise.bxor/2)

    frame <> <<crc>>
  end

  defp binary? do
    base = Application.get_env(:gateway, :parser_path, "iot-parser")
    File.regular?(base) or File.regular?(base <> ".exe")
  end

  test "парсит валидный кадр телеметрии" do
    if binary?() do
      {:ok, json} = Gateway.Parser.parse(telemetry_frame("sensor-1"))
      assert json =~ ~s("device_id":"sensor-1")
      assert json =~ ~s("temp":21.5)
    else
      IO.puts("пропускаю parser test: бинарник не собран")
    end
  end
end
