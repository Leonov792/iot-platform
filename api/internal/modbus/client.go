package modbus

import (
	"fmt"
	"time"

	"github.com/goburrow/modbus"
)

// Client — обёртка над Modbus TCP для оборудования бассейна/спортзала.
// Один клиент = одно Modbus-устройство (слейв). Читает регистры датчиков
// (pH/ORP/уровень воды/влажность) и пишет реле/клапаны (coils + holding).
type Client struct {
	handler *modbus.TCPClientHandler
	client  modbus.Client
}

// NewClient открывает Modbus TCP-соединение. unit — id слейва (обычно 1).
func NewClient(host string, port int, unit byte, timeout time.Duration) *Client {
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", host, port))
	handler.Timeout = timeout
	handler.SlaveId = unit
	return &Client{handler: handler, client: modbus.NewClient(handler)}
}

// Close закрывает TCP-соединение.
func (c *Client) Close() error {
	return c.handler.Close()
}

// ReadHoldingRegisters читает N holding-регистров (FC 0x03), начиная с addr.
func (c *Client) ReadHoldingRegisters(addr, qty uint16) ([]byte, error) {
	return c.client.ReadHoldingRegisters(addr, qty)
}

// ReadInputRegisters читает N input-регистров (FC 0x04) — датчики обычно тут.
func (c *Client) ReadInputRegisters(addr, qty uint16) ([]byte, error) {
	return c.client.ReadInputRegisters(addr, qty)
}

// ReadCoils читает N дискретных выходов (FC 0x01).
func (c *Client) ReadCoils(addr, qty uint16) ([]byte, error) {
	return c.client.ReadCoils(addr, qty)
}

// WriteSingleCoil включает/выключает реле (FC 0x05). on=true -> 0xFF00, false -> 0x0000.
func (c *Client) WriteSingleCoil(addr uint16, on bool) error {
	val := uint16(0xFF00)
	if !on {
		val = 0x0000
	}
	_, err := c.client.WriteSingleCoil(addr, val)
	return err
}

// WriteSingleRegister пишет значение в holding-регистр (FC 0x06).
// Пригодится для уставок (целевая температура, доза химии, скорость насоса).
func (c *Client) WriteSingleRegister(addr, value uint16) error {
	_, err := c.client.WriteSingleRegister(addr, value)
	return err
}
