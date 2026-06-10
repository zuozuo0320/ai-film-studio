package storage

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/models"
)

// Memory 线程安全的内存存储，读写均做深拷贝，模拟数据库行为，
// 避免调用方共享可变指针导致竞态。
type Memory struct {
	mu       sync.RWMutex
	projects map[string]*models.Project
	batches  map[string]*models.GenerationBatch
}

func NewMemory() *Memory {
	return &Memory{
		projects: map[string]*models.Project{},
		batches:  map[string]*models.GenerationBatch{},
	}
}

func deepCopy[T any](src *T) *T {
	b, _ := json.Marshal(src)
	dst := new(T)
	_ = json.Unmarshal(b, dst)
	return dst
}

func (m *Memory) CreateProject(p *models.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[p.ID] = deepCopy(p)
	return nil
}

func (m *Memory) GetProject(id string) (*models.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.projects[id]
	if !ok {
		return nil, ErrNotFound
	}
	return deepCopy(p), nil
}

func (m *Memory) ListProjects() ([]*models.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*models.Project, 0, len(m.projects))
	for _, p := range m.projects {
		out = append(out, deepCopy(p))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *Memory) UpdateProject(p *models.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.projects[p.ID]; !ok {
		return ErrNotFound
	}
	p.UpdatedAt = time.Now()
	m.projects[p.ID] = deepCopy(p)
	return nil
}

func (m *Memory) DeleteProject(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.projects[id]; !ok {
		return ErrNotFound
	}
	delete(m.projects, id)
	return nil
}

func (m *Memory) CreateBatch(b *models.GenerationBatch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batches[b.ID] = deepCopy(b)
	return nil
}

func (m *Memory) GetBatch(id string) (*models.GenerationBatch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.batches[id]
	if !ok {
		return nil, ErrNotFound
	}
	return deepCopy(b), nil
}

func (m *Memory) ListBatches(projectID string) ([]*models.GenerationBatch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []*models.GenerationBatch{}
	for _, b := range m.batches {
		if projectID == "" || b.ProjectID == projectID {
			out = append(out, deepCopy(b))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *Memory) UpdateBatch(b *models.GenerationBatch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.batches[b.ID]; !ok {
		return ErrNotFound
	}
	m.batches[b.ID] = deepCopy(b)
	return nil
}
