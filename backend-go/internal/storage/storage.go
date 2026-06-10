// Package storage 定义存储抽象。生产环境实现 PostgreSQL 版本（见
// docs/backend-architecture-go.md），本骨架提供内存实现便于本地跑通全链路。
package storage

import (
	"errors"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/models"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	CreateProject(p *models.Project) error
	GetProject(id string) (*models.Project, error)
	ListProjects() ([]*models.Project, error)
	UpdateProject(p *models.Project) error
	DeleteProject(id string) error

	CreateBatch(b *models.GenerationBatch) error
	GetBatch(id string) (*models.GenerationBatch, error)
	ListBatches(projectID string) ([]*models.GenerationBatch, error)
	UpdateBatch(b *models.GenerationBatch) error
}
