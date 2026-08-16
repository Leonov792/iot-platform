defmodule Gateway.DiscoveryTest do
  use ExUnit.Case, async: true

  test "разбирает /24" do
    assert {:ok, ips} = Gateway.Discovery.ips("192.168.1.0/24")
    assert length(ips) == 254
    assert "192.168.1.1" in ips
    refute "192.168.1.0" in ips
    refute "192.168.1.255" in ips
  end

  test "разбирает /30" do
    assert {:ok, ips} = Gateway.Discovery.ips("10.0.0.0/30")
    assert ips == ["10.0.0.1", "10.0.0.2"]
  end

  test "отклоняет не CIDR" do
    assert {:error, _} = Gateway.Discovery.ips("192.168.1.0")
    assert {:error, _} = Gateway.Discovery.ips("хрень")
  end

  test "отклоняет слишком большую подсеть" do
    assert {:error, _} = Gateway.Discovery.ips("10.0.0.0/8")
  end

  test "отклоняет /32" do
    assert {:error, _} = Gateway.Discovery.ips("192.168.1.5/32")
  end
end
