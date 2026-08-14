defmodule Gateway.Parser do
  # Rustler NIF: парсинг бинарной телеметрии прямо внутри BEAM через DirtyCpu.
  # Никакого subprocess/Port — библиотека загружается как нативный модуль.
  # rustler 0.38 собирает крейт при компиляции модуля (cargo rustc в path).
  use Rustler, otp_app: :gateway, crate: "gateway_parser", path: "../parser"

  # заглушка заменяется нативной функцией parse/1 при загрузке NIF.
  # если NIF не загрузился (не собран) — честно падаем, а не молча глотаем кадры
  def parse(_binary), do: :erlang.nif_error(:nif_not_loaded)
end
