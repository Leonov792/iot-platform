package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

// эмулятор устройства: цепляется к гейтвею по вебсокету, шлёт бинарную
// телеметрию и слушает команды. нужен, чтобы демо выглядело живым

var (
	deviceID = flag.String("id", "sensor-1", "id устройства")
	devType  = flag.String("type", "sensor", "тип: sensor|light|plug|thermostat")
	gateway  = flag.String("url", "ws://localhost:4000/ws/device/", "url гейтвея")
	interval = flag.Duration("interval", 2*time.Second, "как часто слать телеметрию")
)

const (
	magic1 = 0xAB
	magic2 = 0xCD
	kind   = 0x01 // телеметрия
)

type deviceState struct {
	on         bool
	brightness int
	target     float64
	temp       float64
	humidity   float64
	battery    int
}

func main() {
	flag.Parse()

	st := deviceState{on: false, brightness: 100, target: 22, temp: 21.5, humidity: 45, battery: 100}
	url := *gateway + *deviceID

	for {
		if err := run(url, &st); err != nil {
			log.Printf("соединение отвалилось: %v, переподключаюсь через 2с", err)
		}
		time.Sleep(2 * time.Second)
	}
}

func run(url string, st *deviceState) error {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	defer c.Close()
	log.Printf("подключился к %s как %s", url, *devType)

	// читаем команды в фоне
	go func() {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			handleCommand(st, msg)
		}
	}()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for range ticker.C {
		step(st)
		if err := c.WriteMessage(websocket.BinaryMessage, buildFrame(*deviceID, st)); err != nil {
			return err
		}
	}

	return nil
}

func step(st *deviceState) {
	// дрейфуем показания, чтобы график шевелился
	st.temp += (rand.Float64() - 0.5) * 0.3
	if *devType == "thermostat" {
		st.temp += (st.target - st.temp) * 0.1
	}
	st.humidity += (rand.Float64() - 0.5) * 0.8
	st.humidity = math.Max(20, math.Min(80, st.humidity))
	if st.battery > 0 {
		st.battery--
	}
}

func handleCommand(st *deviceState, msg []byte) {
	var cmd struct {
		Action string  `json:"action"`
		Value  float64 `json:"value"`
	}
	if err := json.Unmarshal(msg, &cmd); err != nil {
		log.Printf("не понял команду: %s", msg)
		return
	}

	switch cmd.Action {
	case "on":
		st.on = true
	case "off":
		st.on = false
	case "set_brightness":
		st.brightness = int(cmd.Value)
		st.on = cmd.Value > 0
	case "set_target":
		st.target = cmd.Value
	}

	log.Printf("команда %s (value=%.0f), состояние: on=%v brightness=%d target=%.1f",
		cmd.Action, cmd.Value, st.on, st.brightness, st.target)
}

func buildFrame(id string, st *deviceState) []byte {
	payload := make([]byte, 9)
	binary.LittleEndian.PutUint32(payload[0:4], math.Float32bits(float32(st.temp)))
	binary.LittleEndian.PutUint32(payload[4:8], math.Float32bits(float32(st.humidity)))
	payload[8] = byte(st.battery)

	idBytes := make([]byte, 16)
	copy(idBytes, id)

	frame := make([]byte, 0, 2+1+16+1+2+9+1)
	frame = append(frame, magic1, magic2, 1)
	frame = append(frame, idBytes...)
	frame = append(frame, kind)
	frame = append(frame, byte(len(payload)>>8), byte(len(payload)))
	frame = append(frame, payload...)

	var crc byte
	for _, b := range frame[2:] {
		crc ^= b
	}
	frame = append(frame, crc)

	return frame
}
