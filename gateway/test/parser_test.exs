defmodule Gateway.ParserTest do
  use ExUnit.Case, async: true

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

  test "NIF парсит валидный кадр телеметрии" do
    {:ok, json} = Gateway.Parser.parse(telemetry_frame("sensor-1"))
    assert json =~ ~s("device_id":"sensor-1")
    assert json =~ ~s("temp":21.5)
    assert json =~ ~s("battery":87)
  end

  test "NIF возвращает ошибку на битый crc" do
    frame = telemetry_frame("sensor-1")
    n = byte_size(frame)
    corrupted = binary_part(frame, 0, n - 1) <> <<Bitwise.bxor(:binary.at(frame, n - 1), 0xFF)>>

    assert {:error, _} = Gateway.Parser.parse(corrupted)
  end

  test "NIF возвращает ошибку на короткий кадр" do
    assert {:error, _} = Gateway.Parser.parse(<<0xAB, 0xCD>>)
  end
end
