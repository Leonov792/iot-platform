package main

import (
	"testing"
	"time"
)

func mk(start time.Time, vals []float64) []Sample {
	out := make([]Sample, len(vals))
	for i, v := range vals {
		out[i] = Sample{Value: v, TS: start.Add(time.Duration(i) * 10 * time.Second)}
	}
	return out
}

func TestHRVDropping(t *testing.T) {
	// стабильно высокий HRV -> не падает
	flat := mk(time.Now(), []float64{50, 51, 50, 52, 51, 50, 51})
	if HRVDropping(flat, -1.0) {
		t.Fatal("стабильный HRV не должен считаться падением")
	}

	// резкое падение -> true
	drop := mk(time.Now(), []float64{60, 59, 58, 55, 50, 44, 38})
	if !HRVDropping(drop, -1.0) {
		t.Fatal("резкое падение HRV должно детектиться")
	}
}

func TestShouldTriggerSoftDawn(t *testing.T) {
	alarm := time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC)

	// за 10 минут до будильника, HRV падает -> срабатывает
	now := alarm.Add(-10 * time.Minute)
	samples := mk(now.Add(-5*time.Minute), []float64{60, 59, 58, 55, 50, 44, 38})
	if !ShouldTriggerSoftDawn(samples, alarm, now, 15*time.Minute, -1.0) {
		t.Fatal("в окне с падением HRV должно срабатывать")
	}

	// слишком рано (за час) -> не срабатывает
	early := alarm.Add(-1 * time.Hour)
	if ShouldTriggerSoftDawn(samples, alarm, early, 15*time.Minute, -1.0) {
		t.Fatal("вне окна не должно срабатывать")
	}

	// после будильника -> не срабатывает
	if ShouldTriggerSoftDawn(samples, alarm, alarm.Add(5*time.Minute), 15*time.Minute, -1.0) {
		t.Fatal("после будильника не должно срабатывать")
	}

	// в окне, но HRV стабилен -> не срабатывает
	flat := mk(now.Add(-5*time.Minute), []float64{50, 51, 50, 52, 51, 50, 51})
	if ShouldTriggerSoftDawn(flat, alarm, now, 15*time.Minute, -1.0) {
		t.Fatal("без падения HRV не должно срабатывать")
	}
}
