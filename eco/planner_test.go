package main

import "testing"

func TestBuildPlanTariff(t *testing.T) {
	// 24 часа: ночью (23,0,1,2) дёшево, днём дорого
	prices := make([]float64, 24)
	for i := range prices {
		prices[i] = 100 // дорого днём
	}
	prices[23], prices[0], prices[1], prices[2] = 10, 5, 5, 5 // дешёвая ночь

	p := BuildPlan(prices, 4)
	if p.Mode != "tariff" {
		t.Fatalf("ждём режим tariff, пришло %q", p.Mode)
	}
	// самое дешёвое 4-часовое окно в ночном диапазоне [23,0,1,2,3,4,5]:
	// варианты: [23,0,1,2]=25, [0,1,2,3]=15+100=115? нет [0,1,2,3]=5+5+5+100=115, [23..]=10+5+5+5=25
	// так что лучший старт 23
	if p.StartHour != 23 {
		t.Fatalf("ждём старт 23, пришло %d", p.StartHour)
	}
	if p.EndHour != 3 {
		t.Fatalf("ждём конец 3, пришло %d", p.EndHour)
	}
}

func TestBuildPlanWeatherFallback(t *testing.T) {
	p := BuildPlan(nil, 4)
	if p.Mode != "weather" {
		t.Fatalf("ждём режим weather, пришло %q", p.Mode)
	}
	if p.StartHour != 23 || p.EndHour != 3 {
		t.Fatalf("эвристика должна греть 23:00-03:00, пришло %d-%d", p.StartHour, p.EndHour)
	}
}

func TestBuildPlanClampsDuration(t *testing.T) {
	p := BuildPlan(nil, 100)
	// duration клампится до размера ночного окна (7): 23..(23+7)%24=6
	if p.StartHour != 23 || p.EndHour != 6 {
		t.Fatalf("ждём 23-6, пришло %d-%d", p.StartHour, p.EndHour)
	}
}
