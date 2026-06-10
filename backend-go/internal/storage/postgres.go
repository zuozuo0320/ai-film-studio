package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/models"
)

// Postgres 生产存储实现。characters/shots 以 JSONB 存储（半结构化、schema 演进灵活），
// 项目级读写模式与 API 形态一致；量级上来后可将 shots 拆独立分区表。
type Postgres struct {
	pool *pgxpool.Pool
}

const schema = `
CREATE TABLE IF NOT EXISTS projects (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    script     TEXT NOT NULL DEFAULT '',
    characters JSONB NOT NULL DEFAULT '[]',
    shots      JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS generation_batches (
    id           TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    shot_ids     JSONB NOT NULL DEFAULT '[]',
    status       TEXT NOT NULL,
    model        TEXT NOT NULL DEFAULT '',
    window_start TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    finished_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_batches_project ON generation_batches(project_id, created_at DESC);
`

func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() { p.pool.Close() }

func (p *Postgres) CreateProject(m *models.Project) error {
	chars, _ := json.Marshal(m.Characters)
	shots, _ := json.Marshal(m.Shots)
	_, err := p.pool.Exec(context.Background(),
		`INSERT INTO projects (id, title, script, characters, shots, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		m.ID, m.Title, m.Script, chars, shots, m.CreatedAt, m.UpdatedAt)
	return err
}

func scanProject(row pgx.Row) (*models.Project, error) {
	var m models.Project
	var chars, shots []byte
	if err := row.Scan(&m.ID, &m.Title, &m.Script, &chars, &shots, &m.CreatedAt, &m.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(chars, &m.Characters); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(shots, &m.Shots); err != nil {
		return nil, err
	}
	return &m, nil
}

func (p *Postgres) GetProject(id string) (*models.Project, error) {
	row := p.pool.QueryRow(context.Background(),
		`SELECT id, title, script, characters, shots, created_at, updated_at FROM projects WHERE id=$1`, id)
	return scanProject(row)
}

func (p *Postgres) ListProjects() ([]*models.Project, error) {
	rows, err := p.pool.Query(context.Background(),
		`SELECT id, title, script, characters, shots, created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*models.Project{}
	for rows.Next() {
		m, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (p *Postgres) UpdateProject(m *models.Project) error {
	chars, _ := json.Marshal(m.Characters)
	shots, _ := json.Marshal(m.Shots)
	m.UpdatedAt = time.Now()
	tag, err := p.pool.Exec(context.Background(),
		`UPDATE projects SET title=$2, script=$3, characters=$4, shots=$5, updated_at=$6 WHERE id=$1`,
		m.ID, m.Title, m.Script, chars, shots, m.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) DeleteProject(id string) error {
	tag, err := p.pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) CreateBatch(b *models.GenerationBatch) error {
	ids, _ := json.Marshal(b.ShotIDs)
	_, err := p.pool.Exec(context.Background(),
		`INSERT INTO generation_batches (id, project_id, shot_ids, status, model, window_start, created_at, finished_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		b.ID, b.ProjectID, ids, b.Status, b.Model, b.WindowStart, b.CreatedAt, b.FinishedAt)
	return err
}

func scanBatch(row pgx.Row) (*models.GenerationBatch, error) {
	var b models.GenerationBatch
	var ids []byte
	if err := row.Scan(&b.ID, &b.ProjectID, &ids, &b.Status, &b.Model, &b.WindowStart, &b.CreatedAt, &b.FinishedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(ids, &b.ShotIDs); err != nil {
		return nil, err
	}
	return &b, nil
}

func (p *Postgres) GetBatch(id string) (*models.GenerationBatch, error) {
	row := p.pool.QueryRow(context.Background(),
		`SELECT id, project_id, shot_ids, status, model, window_start, created_at, finished_at
		 FROM generation_batches WHERE id=$1`, id)
	return scanBatch(row)
}

func (p *Postgres) ListBatches(projectID string) ([]*models.GenerationBatch, error) {
	q := `SELECT id, project_id, shot_ids, status, model, window_start, created_at, finished_at
	      FROM generation_batches`
	args := []any{}
	if projectID != "" {
		q += ` WHERE project_id=$1`
		args = append(args, projectID)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := p.pool.Query(context.Background(), q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*models.GenerationBatch{}
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (p *Postgres) UpdateBatch(b *models.GenerationBatch) error {
	ids, _ := json.Marshal(b.ShotIDs)
	tag, err := p.pool.Exec(context.Background(),
		`UPDATE generation_batches SET shot_ids=$2, status=$3, finished_at=$4 WHERE id=$1`,
		b.ID, ids, b.Status, b.FinishedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
