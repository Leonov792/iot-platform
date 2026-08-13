defmodule Gateway.Application do
  use Application

  @impl true
  def start(_type, _args) do
    port = Application.get_env(:gateway, :port, 4000)

    children = [
      {Finch, name: Gateway.Finch},
      Gateway.Registry,
      Gateway.Parser,
      {Bandit, plug: Gateway.Router, scheme: :http, port: port, ip: {0, 0, 0, 0}}
    ]

    opts = [strategy: :one_for_one, name: Gateway.Supervisor]
    Supervisor.start_link(children, opts)
  end
end
