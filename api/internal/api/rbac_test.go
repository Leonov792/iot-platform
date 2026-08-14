package api

import (
	"testing"
	"time"

	"iot-platform/api/internal/models"
)

func TestScheduleAllows(t *testing.T) {
	// пн..пт 08:00-20:00
	sched := []models.ScheduleEntry{
		{Zone: "pool", Days: []int{1, 2, 3, 4, 5}, Start: "08:00", End: "20:00"},
	}

	// вторник 14:00
	now := time.Date(2026, time.August, 11, 14, 0, 0, 0, time.UTC) // вторник
	if !scheduleAllows(sched, "pool", now) {
		t.Fatal("в рабочие часы должен пускать")
	}

	// вторник 07:00 — рано
	early := time.Date(2026, time.August, 11, 7, 0, 0, 0, time.UTC)
	if scheduleAllows(sched, "pool", early) {
		t.Fatal("вне окна не должен пускать")
	}

	// суббота — выходной
	sat := time.Date(2026, time.August, 8, 14, 0, 0, 0, time.UTC) // суббота
	if scheduleAllows(sched, "pool", sat) {
		t.Fatal("по выходным не должен пускать")
	}

	// другая зона
	if scheduleAllows(sched, "gym", now) {
		t.Fatal("в чужую зону не должен пускать")
	}
}

func TestScheduleAllowsOvernight(t *testing.T) {
	sched := []models.ScheduleEntry{
		{Zone: "gym", Days: []int{1, 2, 3, 4, 5, 6, 7}, Start: "22:00", End: "02:00"},
	}
	night := time.Date(2026, time.August, 11, 23, 30, 0, 0, time.UTC)
	if !scheduleAllows(sched, "gym", night) {
		t.Fatal("ночное окно 22:00-02:00 должно пускать в 23:30")
	}
	afterMidnight := time.Date(2026, time.August, 12, 1, 0, 0, 0, time.UTC)
	if !scheduleAllows(sched, "gym", afterMidnight) {
		t.Fatal("ночное окно должно пускать в 01:00")
	}
}

func TestAuthorizedRoles(t *testing.T) {
	now := time.Date(2026, time.August, 11, 14, 0, 0, 0, time.UTC)
	sched := []models.ScheduleEntry{
		{Zone: "pool", Days: []int{1, 2, 3, 4, 5}, Start: "08:00", End: "20:00"},
	}

	light := models.Device{Type: "light", Zone: "home"}
	pool := models.Device{Type: "sensor", Zone: "pool"}
	gym := models.Device{Type: "plug", Zone: "gym"}

	if !authorized("owner", gym, nil, now) {
		t.Fatal("owner управляет всем")
	}
	if !authorized("family", light, nil, now) {
		t.Fatal("family управляет светом")
	}
	if authorized("family", pool, nil, now) {
		t.Fatal("family не должен лезть в бассейн")
	}
	if !authorized("staff", pool, sched, now) {
		t.Fatal("staff в часы работы должен пускаться в бассейн")
	}
	if authorized("staff", light, sched, now) {
		t.Fatal("staff не должен трогать дом")
	}
	if authorized("staff", pool, nil, now) {
		t.Fatal("staff без расписания не должен пускаться")
	}
}
