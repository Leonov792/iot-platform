package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client шлёт команды в эликсир-гейтвей, а тот уже доставляет их на устройство
// по вебсокету. Прямо с фронта в гейтвей не ходим — всё через api, чтобы был один
// слой с авторизацией.
type Client struct {
	url    string
	client *http.Client
}

// defaultTimeout — таймаут, если клиент не передали (nil), чтобы не зависнуть навсегда.
const defaultTimeout = 5 * time.Second

// NewClient собирает клиент гейтвея. client внедряется снаружи (DI); при nil
// подставляется безопасный дефолтный *http.Client с явным таймаутом.
func NewClient(url string, client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		url:    url,
		client: client,
	}
}

type Command struct {
	DeviceID string `json:"device_id"`
	Action   string `json:"action"`
	Value    any    `json:"value,omitempty"`
}

func (c *Client) SendCommand(ctx context.Context, cmd Command) error {
	body, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/internal/command", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("гейтвей вернул %d", resp.StatusCode)
	}
	return nil
}
