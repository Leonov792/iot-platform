package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
)

// binding — характеристики устройства для синка состояния из API.
type binding struct {
	deviceType  string
	on          *characteristic.On
	brightness  *characteristic.Brightness
	currentTemp *characteristic.CurrentTemperature
	targetTemp  *characteristic.TargetTemperature
	humidity    *characteristic.CurrentRelativeHumidity
}

// buildBridge собирает HAP-мост + саб-аксессуары из списка устройств.
func buildBridge(api *APIClient, devices []Device) (*accessory.A, []*accessory.A, map[string]*binding) {
	bridge := accessory.NewBridge(accessory.Info{
		Name:         "IoT Bridge",
		Manufacturer: "Leonov792",
		Model:        "iot-platform",
	})

	accs := make([]*accessory.A, 0, len(devices))
	bindings := make(map[string]*binding, len(devices))

	for _, d := range devices {
		a, b := makeAccessory(api, d)
		if a == nil {
			continue
		}
		accs = append(accs, a)
		bindings[d.ID] = b
	}

	return bridge.A, accs, bindings
}

func makeAccessory(api *APIClient, d Device) (*accessory.A, *binding) {
	info := accessory.Info{Name: d.Name, Manufacturer: "Leonov792", Model: "iot-platform"}

	switch d.Type {
	case "light":
		l := accessory.NewLightbulb(info)

		brightness := characteristic.NewBrightness()
		brightness.SetMinValue(0)
		brightness.SetMaxValue(100)
		brightness.SetStepValue(1)
		l.Lightbulb.AddC(brightness.C)

		l.Lightbulb.On.OnValueRemoteUpdate(func(on bool) {
			action := "off"
			if on {
				action = "on"
			}
			if err := command(api, d.ID, action, nil); err != nil {
				slog.Warn("команда света не ушла", "device", d.ID, "err", err)
			}
		})
		brightness.OnValueRemoteUpdate(func(v int) {
			if err := command(api, d.ID, "set_brightness", float64(v)); err != nil {
				slog.Warn("яркость не ушла", "device", d.ID, "err", err)
			}
		})

		return l.A, &binding{deviceType: "light", on: l.Lightbulb.On, brightness: brightness}

	case "plug":
		o := accessory.NewOutlet(info)
		o.Outlet.On.OnValueRemoteUpdate(func(on bool) {
			action := "off"
			if on {
				action = "on"
			}
			if err := command(api, d.ID, action, nil); err != nil {
				slog.Warn("команда розетки не ушла", "device", d.ID, "err", err)
			}
		})

		return o.A, &binding{deviceType: "plug", on: o.Outlet.On}

	case "thermostat":
		t := accessory.NewThermostat(info)
		t.Thermostat.TargetTemperature.SetMinValue(5)
		t.Thermostat.TargetTemperature.SetMaxValue(35)
		t.Thermostat.TargetTemperature.SetStepValue(1)

		t.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(v float64) {
			if err := command(api, d.ID, "set_target", v); err != nil {
				slog.Warn("температура не ушла", "device", d.ID, "err", err)
			}
		})

		return t.A, &binding{
			deviceType:  "thermostat",
			currentTemp: t.Thermostat.CurrentTemperature,
			targetTemp:  t.Thermostat.TargetTemperature,
		}

	case "sensor":
		a := accessory.New(info, accessory.TypeSensor)
		temp := service.NewTemperatureSensor()
		hum := service.NewHumiditySensor()
		a.AddS(temp.S)
		a.AddS(hum.S)

		return a, &binding{
			deviceType:  "sensor",
			currentTemp: temp.CurrentTemperature,
			humidity:    hum.CurrentRelativeHumidity,
		}

	default:
		return nil, nil
	}
}

// command отправляет команду в api с коротким таймаутом.
func command(api *APIClient, deviceID, action string, value any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return api.Command(ctx, deviceID, action, value)
}

// syncDevices тянет состояние устройств и телеметрию, обновляет характеристики.
func syncDevices(api *APIClient, bindings map[string]*binding) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	devices, err := api.ListDevices(ctx)
	if err != nil {
		slog.Warn("не вытащил устройства для синка", "err", err)
		return
	}

	byID := make(map[string]Device, len(devices))
	for _, d := range devices {
		byID[d.ID] = d
	}

	for id, b := range bindings {
		d, ok := byID[id]
		if !ok {
			continue
		}

		switch b.deviceType {
		case "light":
			if on, ok := d.State["on"].(bool); ok {
				b.on.SetValue(on)
			}
			if br, ok := d.State["brightness"].(float64); ok {
				_ = b.brightness.SetValue(int(br))
			}

		case "plug":
			if on, ok := d.State["on"].(bool); ok {
				b.on.SetValue(on)
			}

		case "thermostat":
			if target, ok := d.State["target_temp"].(float64); ok {
				b.targetTemp.SetValue(target)
			}
			if p, err := api.LatestTelemetry(ctx, id); err == nil {
				if v, ok := p["temp"].(float64); ok {
					b.currentTemp.SetValue(v)
				}
			}

		case "sensor":
			if p, err := api.LatestTelemetry(ctx, id); err == nil {
				if v, ok := p["temp"].(float64); ok {
					b.currentTemp.SetValue(v)
				}
				if v, ok := p["humidity"].(float64); ok {
					b.humidity.SetValue(v)
				}
			}
		}
	}
}
