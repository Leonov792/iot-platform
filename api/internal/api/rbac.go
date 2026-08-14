package api

import (
	"time"

	"iot-platform/api/internal/auth"
	"iot-platform/api/internal/models"
)

// семейство управляет только климатом и светом
var familyAllowedTypes = map[string]bool{
	"light":      true,
	"thermostat": true,
}

// staffAllowedZones — персонал допускается только в бассейн/спортзал
func staffZone(zone string) bool {
	return zone == "pool" || zone == "gym"
}

// authorized решает, может ли роль управлять устройством прямо сейчас.
func authorized(role string, d models.Device, schedule []models.ScheduleEntry, now time.Time) bool {
	switch role {
	case auth.RoleOwner:
		return true
	case auth.RoleFamily:
		return familyAllowedTypes[d.Type]
	case auth.RoleStaff:
		if !staffZone(d.Zone) {
			return false
		}
		return scheduleAllows(schedule, d.Zone, now)
	default:
		return false
	}
}

// canManageSystem — право менять настройки системы (устройства/пользователей).
// только владелец.
func canManageSystem(role string) bool {
	return role == auth.RoleOwner
}

// scheduleAllows проверяет, что now попадает в окно доступа к зоне.
// days — по time.Weekday (0=воскресенье). пустой список дней = все дни.
func scheduleAllows(schedule []models.ScheduleEntry, zone string, now time.Time) bool {
	weekday := int(now.Weekday())
	cur := now.Hour()*60 + now.Minute()

	for _, e := range schedule {
		if e.Zone != zone {
			continue
		}
		if !dayAllowed(e.Days, weekday) {
			continue
		}
		start, ok1 := hhmm(e.Start)
		end, ok2 := hhmm(e.End)
		if !ok1 || !ok2 {
			continue
		}
		// интервал не переваливает через полночь
		if start <= end {
			if cur >= start && cur <= end {
				return true
			}
		} else {
			// переваливает: напр. 22:00-02:00
			if cur >= start || cur <= end {
				return true
			}
		}
	}
	return false
}

func dayAllowed(days []int, weekday int) bool {
	if len(days) == 0 {
		return true
	}
	for _, d := range days {
		if d == weekday {
			return true
		}
	}
	return false
}

// hhmm парсит "08:00" в минуты с начала суток.
func hhmm(s string) (int, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	if h > 23 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
