package rules

import (
	"log"
	"sync"
	"time"
)

// Executor исполняет действия правил. Реализация — http-клиент к
// modbus-poller'у и гейтвею (см. main.go).
type Executor interface {
	ModbusWrite(deviceID, relay string, value bool) error
	Command(deviceID, action string, value any) error
}

// Engine хранит правила, отслеживает cooldown и запускает действия.
type Engine struct {
	rules    []Rule
	executor Executor

	mu       sync.Mutex
	lastFire map[string]time.Time
}

func NewEngine(rules []Rule, executor Executor) *Engine {
	return &Engine{
		rules:    rules,
		executor: executor,
		lastFire: map[string]time.Time{},
	}
}

// Rules возвращает копию текущего набора правил.
func (e *Engine) Rules() []Rule {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Rule(nil), e.rules...)
}

// SetRules заменяет правила на лету (из конструктора автоматизаций).
// cooldown сбрасывается, чтобы новые правила срабатывали сразу.
func (e *Engine) SetRules(rules []Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = rules
	e.lastFire = map[string]time.Time{}
}

// Devices — уникальные device_id из условий (кого опрашивать).
func (e *Engine) Devices() []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, r := range e.Rules() {
		if !seen[r.Condition.DeviceID] {
			seen[r.Condition.DeviceID] = true
			out = append(out, r.Condition.DeviceID)
		}
	}
	return out
}

// Evaluate принимает снимок состояний: map[deviceID]map[field]float64.
// Для каждого правила, чьё условие выполняется, запускает действия с учётом cooldown.
func (e *Engine) Evaluate(states map[string]map[string]float64, now time.Time) {
	for _, r := range e.Rules() {
		payload, ok := states[r.Condition.DeviceID]
		if !ok {
			continue
		}
		val, ok := payload[r.Condition.Field]
		if !ok {
			continue
		}
		if !Match(r.Condition.Op, val, r.Condition.Value) {
			continue
		}
		if !e.cooldownOK(r, now) {
			continue
		}
		e.fire(r, now)
	}
}

func (e *Engine) fire(r Rule, now time.Time) {
	e.mu.Lock()
	e.lastFire[r.ID] = now
	e.mu.Unlock()

	log.Printf("[automation] правило %q (%s) сработало", r.Name, r.ID)
	for _, a := range r.Actions {
		e.exec(a)
	}
}

func (e *Engine) exec(a Action) {
	switch a.Type {
	case "modbus_write":
		if err := e.executor.ModbusWrite(a.DeviceID, a.Relay, a.ValueBool()); err != nil {
			log.Printf("[automation] modbus_write %s/%s: %v", a.DeviceID, a.Relay, err)
			return
		}
		if a.OffAfter != "" {
			if d, err := time.ParseDuration(a.OffAfter); err == nil {
				// дозатор/клапан: включили, через N секунд выключаем
				deviceID, relay := a.DeviceID, a.Relay
				time.AfterFunc(d, func() {
					if err := e.executor.ModbusWrite(deviceID, relay, false); err != nil {
						log.Printf("[automation] modbus_write OFF %s/%s: %v", deviceID, relay, err)
					}
				})
			}
		}

	case "command":
		if err := e.executor.Command(a.DeviceID, a.Action, a.Value); err != nil {
			log.Printf("[automation] command %s/%s: %v", a.DeviceID, a.Action, err)
		}

	default:
		log.Printf("[automation] неизвестный тип действия %q", a.Type)
	}
}

func (e *Engine) cooldownOK(r Rule, now time.Time) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	last, ok := e.lastFire[r.ID]
	if !ok {
		return true
	}
	d := time.Duration(0)
	if r.Cooldown != "" {
		if parsed, err := time.ParseDuration(r.Cooldown); err == nil {
			d = parsed
		}
	}
	return now.Sub(last) >= d
}

// Match сравнивает фактическое значение с порогом.
func Match(op string, got, want float64) bool {
	switch op {
	case "gt":
		return got > want
	case "lt":
		return got < want
	case "gte":
		return got >= want
	case "lte":
		return got <= want
	case "eq":
		return got == want
	case "neq":
		return got != want
	default:
		return false
	}
}
