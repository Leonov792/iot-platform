package api

import (
	"context"
	"time"

	"iot-platform/api/internal/gateway"
	"iot-platform/api/internal/models"
)

// интерфейсы хранилищ нужны, чтобы хендлеры можно было тестировать без базы.
// контракт повторяет сигнатуры *store.*Store один в один

type deviceStore interface {
	List(ctx context.Context, ownerID string) ([]models.Device, error)
	Create(ctx context.Context, d models.Device) error
	Update(ctx context.Context, ownerID string, d models.Device) error
	Delete(ctx context.Context, ownerID, id string) error
	Get(ctx context.Context, id string) (models.Device, error)
	OwnedBy(ctx context.Context, deviceID, ownerID string) (bool, error)
	Touch(ctx context.Context, id string) error
}

type userStore interface {
	Create(ctx context.Context, u models.User) error
	GetByEmail(ctx context.Context, email string) (models.User, error)
}

type telemetryStore interface {
	Insert(ctx context.Context, deviceID string, payload map[string]any) error
	List(ctx context.Context, deviceID string, since time.Time, limit int) ([]models.Telemetry, error)
}

type commandSender interface {
	SendCommand(ctx context.Context, cmd gateway.Command) error
}
