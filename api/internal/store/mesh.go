package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HomeMesh — модель дома для X-Ray AR.
type HomeMesh struct {
	Mesh    any       `json:"mesh"`
	Anchors any       `json:"anchors"`
	Zones   any       `json:"zones"`
	Updated time.Time `json:"updated_at"`
}

// MeshStore — хранение 3D-модели дома и якорей.
type MeshStore struct {
	db *pgxpool.Pool
}

func NewMeshStore(db *pgxpool.Pool) *MeshStore {
	return &MeshStore{db: db}
}

// Put сохраняет (upsert) модель дома владельца.
func (s *MeshStore) Put(ctx context.Context, ownerID string, mesh, anchors, zones any) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO home_mesh (owner_id, mesh, anchors, zones, updated_at)
		 VALUES ($1,$2,$3,$4, now())
		 ON CONFLICT (owner_id) DO UPDATE SET mesh=$2, anchors=$3, zones=$4, updated_at=now()`,
		ownerID, mesh, anchors, zones)
	return err
}

// Get возвращает модель дома (ErrNotFound, если ещё не сохранена).
func (s *MeshStore) Get(ctx context.Context, ownerID string) (HomeMesh, error) {
	var m HomeMesh
	err := s.db.QueryRow(ctx,
		`SELECT mesh, anchors, zones, updated_at FROM home_mesh WHERE owner_id=$1`,
		ownerID).Scan(&m.Mesh, &m.Anchors, &m.Zones, &m.Updated)
	if err != nil {
		return m, err
	}
	return m, nil
}
