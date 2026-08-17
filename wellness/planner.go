package main

import (
	"time"
)

// Sample — отсчёт HRV (вариабельность сердечного ритма, мс).
type Sample struct {
	Value float64   `json:"value"`
	TS    time.Time `json:"ts"`
}

// slope — коэффициент наклона линейной регрессии (HRV по индексу).
func slope(samples []Sample) float64 {
	n := float64(len(samples))
	if n < 2 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2 float64
	for i, s := range samples {
		x := float64(i)
		y := s.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}
	return (n*sumXY - sumX*sumY) / denom
}

// HRVDropping — резкое падение HRV (признак выхода из глубокой фазы сна).
func HRVDropping(samples []Sample, threshold float64) bool {
	const minPoints = 6
	if len(samples) < minPoints {
		return false
	}
	recent := samples[len(samples)-minPoints:]
	return slope(recent) < threshold // threshold отрицательный (напр. -1.0)
}

// ShouldTriggerSoftDawn — HRV падает в окне [alarm-window, alarm] до будильника.
func ShouldTriggerSoftDawn(samples []Sample, alarm, now time.Time, window time.Duration, threshold float64) bool {
	if now.Before(alarm.Add(-window)) || now.After(alarm) {
		return false
	}
	inWindow := make([]Sample, 0)
	for _, s := range samples {
		if s.TS.After(alarm.Add(-window)) {
			inWindow = append(inWindow, s)
		}
	}
	return HRVDropping(inWindow, threshold)
}
