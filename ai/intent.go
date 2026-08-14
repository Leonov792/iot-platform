package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// Command — команда устройства, которую понимает go api.
type Command struct {
	DeviceID string `json:"device_id"`
	Command  string `json:"command"`
	Value    any    `json:"value"`
}

// systemPrompt заставляет модель работать строгим детерминированным парсером намерений:
// на вход — текст/голос, на выход — только валидный JSON-массив команд.
const systemPrompt = `Ты — парсер намерений для системы умного дома. Пользователь пишет голосом или текстом, что нужно сделать.

Твоя задача — вернуть СТРОГО валидный JSON-массив команд, БЕЗ markdown-разметки, БЕЗ пояснений, БЕЗ текста вокруг JSON.

Каждая команда — объект с полями:
- "device_id": строка (id устройства)
- "command": одна из: "on", "off", "set_temp", "set_brightness", "set_color"
- "value": число или строка HEX (для set_color), число (для set_temp/set_brightness)

Правила:
- set_temp: value — число градусов Цельсия (например 18)
- set_color: value — строка HEX без решётки, 6 символов (например "0000FF" — синий)
- set_brightness: value — число 0..100
- on/off: value можно опустить или true/false

Известные устройства:
- gym-climat (климат спортзала)
- pool-light (свет бассейна)
- pool-heater (подогрев бассейна)
- light-living (свет в гостиной)
- thermostat-home (термостат)

Если устройство не указано, подставь наиболее подходящее из списка. Если намерение непонятно — верни пустой массив [].

Примеры:
Вход: "Подготовь спортзал к тренировке и сделай синюю подсветку в бассейне"
Выход:
[{"device_id":"gym-climat","command":"set_temp","value":18},{"device_id":"pool-light","command":"set_color","value":"0000FF"}]

Вход: "Выключи весь свет"
Выход:
[{"device_id":"light-living","command":"off","value":false}]

Верни только JSON.`

// parseIntent гонит текст через модель и распарсивает ответ в []Command.
func parseIntent(ctx context.Context, client *OllamaClient, text string) ([]Command, error) {
	raw, err := client.Chat(ctx, systemPrompt, text)
	if err != nil {
		return nil, err
	}
	return parseCommandJSON(raw)
}

// parseCommandJSON достаёт валидный JSON-массив команд из ответа модели.
// модель иногда оборачивает ответ в markdown-блоки или добавляет пояснения.
func parseCommandJSON(raw string) ([]Command, error) {
	cleaned := stripFences(raw)

	var cmds []Command
	if err := json.Unmarshal([]byte(cleaned), &cmds); err == nil {
		return cmds, nil
	}

	// фолбэк: вытащить первый JSON-массив из ответа
	if start := strings.Index(cleaned, "["); start >= 0 {
		if end := strings.LastIndex(cleaned, "]"); end > start {
			if err2 := json.Unmarshal([]byte(cleaned[start:end+1]), &cmds); err2 == nil {
				return cmds, nil
			}
		}
	}
	return nil, errors.New("модель вернула не-JSON")
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
