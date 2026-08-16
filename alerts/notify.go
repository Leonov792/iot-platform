package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ---- Telegram ----

type telegramNotifier struct {
	token string
	chat  string
	http  *http.Client
}

func newTelegramNotifier(token, chat string, client *http.Client) *telegramNotifier {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &telegramNotifier{token: token, chat: chat, http: client}
}

func (t *telegramNotifier) Send(title, body string) error {
	text := title
	if body != "" {
		text = title + "\n" + body
	}
	payload := map[string]any{
		"chat_id":    t.chat,
		"text":       text,
		"parse_mode": "HTML",
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := t.http.Post("https://api.telegram.org/bot"+t.token+"/sendMessage",
		"application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram ответил %d", resp.StatusCode)
	}
	return nil
}

// ---- Firebase Cloud Messaging (HTTP v1, service account) ----

type fcmNotifier struct {
	project string
	topic   string
	client  *http.Client
}

type serviceAccount struct {
	ProjectID string `json:"project_id"`
}

func newFCMNotifier(credsPath, topic string, client *http.Client) (*fcmNotifier, error) {
	key, err := os.ReadFile(credsPath)
	if err != nil {
		return nil, err
	}

	var sa serviceAccount
	if err := json.Unmarshal(key, &sa); err != nil {
		return nil, err
	}
	if sa.ProjectID == "" {
		return nil, fmt.Errorf("в credentials нет project_id")
	}

	// service account -> oauth2 JWT c scope firebase.messaging
	cfg, err := google.JWTConfigFromJSON(key, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return nil, err
	}

	c := oauth2.NewClient(context.Background(), cfg.TokenSource(context.Background()))
	// явный таймаут: из внедрённого клиента, иначе дефолт 10s (иначе горутина может висеть вечно)
	c.Timeout = timeoutOf(client)

	return &fcmNotifier{
		project: sa.ProjectID,
		topic:   topic,
		client:  c,
	}, nil
}

// timeoutOf возвращает таймаут внедрённого клиента или безопасный дефолт 10s.
func timeoutOf(client *http.Client) time.Duration {
	if client != nil && client.Timeout > 0 {
		return client.Timeout
	}
	return 10 * time.Second
}

func (f *fcmNotifier) Send(title, body string) error {
	payload := map[string]any{
		"message": map[string]any{
			"topic": f.topic,
			"notification": map[string]any{
				"title": title,
				"body":  body,
			},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := "https://fcm.googleapis.com/v1/projects/" + f.project + "/messages:send"
	resp, err := f.client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fcm ответил %d", resp.StatusCode)
	}
	return nil
}
