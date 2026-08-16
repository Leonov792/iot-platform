package main

import (
	"math"
)

// Plan — окно нагрева бассейна (часы, 0..23).
type Plan struct {
	StartHour int    `json:"start_hour"`
	EndHour   int    `json:"end_hour"`
	Mode      string `json:"mode"` // tariff | weather
}

// nightWindow — часы ночного окна (23:00–06:00) по порядку, включая переход через полночь.
var nightWindow = []int{23, 0, 1, 2, 3, 4, 5}

// BuildPlan выбирает дешёвое непрерывное окно длиной duration часов внутри ночного окна.
// prices — 24 значения по часам (0 = 00:00). Если цен нет — эвристика «нагрев ночью» с 23:00.
func BuildPlan(prices []float64, duration int) Plan {
	if duration < 1 {
		duration = 1
	}
	if duration > len(nightWindow) {
		duration = len(nightWindow)
	}

	if len(prices) == 0 {
		return Plan{StartHour: 23, EndHour: (23 + duration) % 24, Mode: "weather"}
	}

	bestStart := nightWindow[0]
	bestSum := math.MaxFloat64
	for i := 0; i+duration <= len(nightWindow); i++ {
		sum := 0.0
		for j := i; j < i+duration; j++ {
			h := nightWindow[j]
			if h < len(prices) {
				sum += prices[h]
			}
		}
		if sum < bestSum {
			bestSum = sum
			bestStart = nightWindow[i]
		}
	}

	return Plan{StartHour: bestStart, EndHour: (bestStart + duration) % 24, Mode: "tariff"}
}
