package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Device — устройство из go api.
type Device struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Type  string         `json:"type"` // light | plug | thermostat | sensor
	State map[string]any `json:"state"`
}

// APIClient ходит в go api: логин, список устройств, команды, телеметрия.
type APIClient struct {
	base     string
	http     *http.Client
	email    string
	password string

	mu    sync.Mutex
	token string
}

func NewAPIClient(base, email, password string, client *http.Client) *APIClient {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &APIClient{base: base, http: client, email: email, password: password}
}

// login получает JWT. Возвращает ошибку, если не пустило.
func (a *APIClient) login(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{"email": a.email, "password": a.password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.base+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login: api ответил %d", resp.StatusCode)
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if out.Token == "" {
		return errors.New("login: пустой токен")
	}

	a.mu.Lock()
	a.token = out.Token
	a.mu.Unlock()
	return nil
}

// do выполняет запрос с Bearer-токеном. При 401 — перелогинивается и повторяет один раз.
func (a *APIClient) do(ctx context.Context, method, path string, body any, out any) error {
	for attempt := 0; attempt < 2; attempt++ {
		a.mu.Lock()
		token := a.token
		a.mu.Unlock()

		if token == "" {
			if err := a.login(ctx); err != nil {
				return err
			}
			a.mu.Lock()
			token = a.token
			a.mu.Unlock()
		}

		var buf bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&buf).Encode(body); err != nil {
				return err
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, a.base+path, &buf)
		if err != nil {
			return err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := a.http.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusUnauthorized {
			_ = resp.Body.Close()
			// токен протух — перелогиниваемся и пробуем ещё раз
			if err := a.login(ctx); err != nil {
				return err
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			_ = resp.Body.Close()
			return fmt.Errorf("%s %s: api ответил %d", method, path, resp.StatusCode)
		}
		defer resp.Body.Close()

		if out != nil {
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				return err
			}
		}
		return nil
	}
	return errors.New("не смог авторизоваться")
}

// ListDevices возвращает устройства дома.
func (a *APIClient) ListDevices(ctx context.Context) ([]Device, error) {
	var out []Device
	if err := a.do(ctx, http.MethodGet, "/api/v1/devices", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Command отправляет команду на устройство.
func (a *APIClient) Command(ctx context.Context, deviceID, action string, value any) error {
	return a.do(ctx, http.MethodPost, "/api/v1/devices/"+deviceID+"/command",
		map[string]any{"action": action, "value": value}, nil)
}

// LatestTelemetry возвращает последний payload телеметрии устройства.
func (a *APIClient) LatestTelemetry(ctx context.Context, deviceID string) (map[string]any, error) {
	var out []struct {
		Payload map[string]any `json:"payload"`
	}
	if err := a.do(ctx, http.MethodGet, "/api/v1/devices/"+deviceID+"/telemetry?limit=1", nil, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, errors.New("телеметрии нет")
	}
	return out[len(out)-1].Payload, nil
}
