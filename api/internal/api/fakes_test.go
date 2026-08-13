package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"iot-platform/api/internal/auth"
	"iot-platform/api/internal/gateway"
	"iot-platform/api/internal/models"
	"iot-platform/api/internal/store"
)

// фейковые хранилища в памяти — чтобы гонять хендлеры без базы

type fakeDeviceStore struct {
	mu      sync.Mutex
	devices map[string]models.Device
}

func newFakeDeviceStore() *fakeDeviceStore {
	return &fakeDeviceStore{devices: map[string]models.Device{}}
}

func (f *fakeDeviceStore) List(_ context.Context, ownerID string) ([]models.Device, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]models.Device, 0)
	for _, d := range f.devices {
		if d.OwnerID == ownerID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeDeviceStore) Create(_ context.Context, d models.Device) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.devices[d.ID] = d
	return nil
}

func (f *fakeDeviceStore) Update(_ context.Context, ownerID string, d models.Device) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if cur, ok := f.devices[d.ID]; ok && cur.OwnerID == ownerID {
		f.devices[d.ID] = d
	}
	return nil
}

func (f *fakeDeviceStore) Delete(_ context.Context, ownerID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if cur, ok := f.devices[id]; ok && cur.OwnerID == ownerID {
		delete(f.devices, id)
	}
	return nil
}

func (f *fakeDeviceStore) Get(_ context.Context, id string) (models.Device, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if d, ok := f.devices[id]; ok {
		return d, nil
	}
	return models.Device{}, store.ErrNotFound
}

func (f *fakeDeviceStore) OwnedBy(_ context.Context, deviceID, ownerID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.devices[deviceID]
	return ok && d.OwnerID == ownerID, nil
}

func (f *fakeDeviceStore) Touch(_ context.Context, id string) error { return nil }

type fakeUserStore struct {
	mu    sync.Mutex
	users map[string]models.User
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{users: map[string]models.User{}}
}

func (f *fakeUserStore) Create(_ context.Context, u models.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[u.Email] = u
	return nil
}

func (f *fakeUserStore) GetByEmail(_ context.Context, email string) (models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	return models.User{}, store.ErrNotFound
}

type fakeTelemetryStore struct {
	mu   sync.Mutex
	rows map[string][]models.Telemetry
}

func newFakeTelemetryStore() *fakeTelemetryStore {
	return &fakeTelemetryStore{rows: map[string][]models.Telemetry{}}
}

func (f *fakeTelemetryStore) Insert(_ context.Context, deviceID string, payload map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[deviceID] = append(f.rows[deviceID], models.Telemetry{DeviceID: deviceID, Payload: payload})
	return nil
}

func (f *fakeTelemetryStore) List(_ context.Context, deviceID string, _ time.Time, _ int) ([]models.Telemetry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rows[deviceID], nil
}

type fakeGateway struct {
	mu   sync.Mutex
	sent []gateway.Command
	err  error
}

func (f *fakeGateway) SendCommand(_ context.Context, cmd gateway.Command) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, cmd)
	return f.err
}

func newTestRouter() (http.Handler, *fakeDeviceStore, *fakeUserStore, *fakeTelemetryStore, *fakeGateway) {
	ds := newFakeDeviceStore()
	us := newFakeUserStore()
	ts := newFakeTelemetryStore()
	gw := &fakeGateway{}

	h := NewHandler(ds, gw)
	ah := NewAuthHandler(us)
	th := NewTelemetryHandler(ts, ds, "test-token")

	return NewRouter(h, ah, th), ds, us, ts, gw
}

func authHeader(userID string) string {
	tok, _ := auth.GenerateToken(userID, time.Hour)
	return "Bearer " + tok
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
