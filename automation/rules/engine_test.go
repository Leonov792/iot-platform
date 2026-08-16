package rules

import (
	"testing"
	"time"
)

func TestMatch(t *testing.T) {
	cases := []struct {
		op        string
		got, want float64
		expected  bool
	}{
		{"gt", 10, 5, true},
		{"lt", 10, 5, false},
		{"gte", 5, 5, true},
		{"lte", 6, 5, false},
		{"eq", 5, 5, true},
		{"neq", 5, 6, true},
		{"bogus", 5, 6, false},
	}
	for _, c := range cases {
		if Match(c.op, c.got, c.want) != c.expected {
			t.Fatalf("Match(%s,%v,%v) != %v", c.op, c.got, c.want, c.expected)
		}
	}
}

type recordingExecutor struct {
	writes []modbusWrite
	cmds   []command
}

type modbusWrite struct {
	device, relay string
	value         bool
}

type command struct {
	device, action string
	value          any
}

func (r *recordingExecutor) ModbusWrite(device, relay string, value bool) error {
	r.writes = append(r.writes, modbusWrite{device, relay, value})
	return nil
}

func (r *recordingExecutor) Command(device, action string, value any) error {
	r.cmds = append(r.cmds, command{device, action, value})
	return nil
}

func TestEngineFiresOnCondition(t *testing.T) {
	ex := &recordingExecutor{}
	eng := NewEngine([]Rule{
		{
			ID:        "r1",
			Name:      "влажность",
			Condition: Condition{DeviceID: "gym-climat", Field: "humidity", Op: "gt", Value: 70},
			Actions:   []Action{{Type: "modbus_write", DeviceID: "gym-climat", Relay: "exhaust_fan", Value: true}},
			Cooldown:  "1m",
		},
	}, ex)

	now := time.Now()
	states := map[string]map[string]float64{"gym-climat": {"humidity": 75}}
	eng.Evaluate(states, now)

	if len(ex.writes) != 1 || !ex.writes[0].value {
		t.Fatalf("вентиляция должна была включиться: %+v", ex.writes)
	}

	// в пределах cooldown не повторяем
	eng.Evaluate(states, now.Add(10*time.Second))
	if len(ex.writes) != 1 {
		t.Fatalf("в пределах cooldown не должно быть повторов: %+v", ex.writes)
	}

	// после cooldown снова срабатывает
	eng.Evaluate(states, now.Add(2*time.Minute))
	if len(ex.writes) != 2 {
		t.Fatalf("после cooldown должно сработать повторно: %+v", ex.writes)
	}
}

func TestEngineNoFireBelowThreshold(t *testing.T) {
	ex := &recordingExecutor{}
	eng := NewEngine([]Rule{
		{
			ID:        "r1",
			Condition: Condition{DeviceID: "pool-pump", Field: "water_level", Op: "lt", Value: 20},
			Actions:   []Action{{Type: "modbus_write", DeviceID: "pool-pump", Relay: "fill_valve", Value: true}},
			Cooldown:  "0s",
		},
	}, ex)

	states := map[string]map[string]float64{"pool-pump": {"water_level": 80}}
	eng.Evaluate(states, time.Now())

	if len(ex.writes) != 0 {
		t.Fatalf("выше порога не должно срабатывать: %+v", ex.writes)
	}
}

func TestDeviceIDs(t *testing.T) {
	eng := NewEngine([]Rule{
		{Condition: Condition{DeviceID: "a", Field: "x"}},
		{Condition: Condition{DeviceID: "b", Field: "y"}},
		{Condition: Condition{DeviceID: "a", Field: "z"}},
	}, &recordingExecutor{})

	ids := eng.Devices()
	if len(ids) != 2 {
		t.Fatalf("ждём 2 уникальных устройства, пришло %v", ids)
	}
}
