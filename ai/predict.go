package main

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Recommendation — рекомендация привычки для push-уведомления.
type Recommendation struct {
	DeviceID string `json:"device_id"`
	Action   string `json:"action"`
	Weekday  int    `json:"weekday"` // 1..7 (ISO: понедельник=1)
	Count    int    `json:"count"`
	Text     string `json:"text"`
}

var weekdayNames = map[int]string{
	1: "понедельникам",
	2: "вторникам",
	3: "средам",
	4: "четвергам",
	5: "пятницам",
	6: "субботам",
	7: "воскресеньям",
}

// Predictor ищет повторяющиеся действия хозяина в логе команд.
type Predictor struct {
	db *pgxpool.Pool
}

func NewPredictor(db *pgxpool.Pool) *Predictor {
	return &Predictor{db: db}
}

// Analyze находит действия, которые повторяются в один и тот же день недели
// минимум minRepetitions раз за последние days дней.
func (p *Predictor) Analyze(ctx context.Context, days int, minRepetitions int) ([]Recommendation, error) {
	if days <= 0 {
		days = 30
	}
	if minRepetitions <= 0 {
		minRepetitions = 3
	}

	rows, err := p.db.Query(ctx, `
		SELECT device_id, action, EXTRACT(ISODOW FROM ts)::int AS dow, COUNT(*) AS cnt
		FROM device_commands
		WHERE ts > now() - make_interval(days => $1)
		GROUP BY device_id, action, EXTRACT(ISODOW FROM ts)
		HAVING COUNT(*) >= $2
		ORDER BY cnt DESC`, days, minRepetitions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Recommendation, 0)
	for rows.Next() {
		var r Recommendation
		if err := rows.Scan(&r.DeviceID, &r.Action, &r.Weekday, &r.Count); err != nil {
			return nil, err
		}
		day, ok := weekdayNames[r.Weekday]
		if !ok {
			day = "некоторым дням"
		}
		r.Text = "Вы часто отправляете команду «" + r.Action + "» на " + r.DeviceID +
			" по " + day + " (" + strconv.Itoa(r.Count) + " раз за месяц). Создать правило автоматизации?"
		out = append(out, r)
	}
	return out, rows.Err()
}
