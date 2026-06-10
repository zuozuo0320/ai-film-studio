// Package scheduler 批次调度器：每个窗口（默认 10 分钟）扫描 status=queued
// 的分镜，按项目组装 GenerationBatch 并投递出图任务。
// 生产环境为单活部署（Redis 分布式锁选主），见架构文档。
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/models"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/queue"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/storage"
)

// Locker 分布式锁抽象：多副本部署时保证单活组装批次。
type Locker interface {
	TryAcquire(ctx context.Context) bool
	Release(ctx context.Context)
}

type Scheduler struct {
	store     storage.Store
	q         queue.Queue
	interval  time.Duration
	maxImages int
	model     string
	lock      Locker // 可选
}

func New(store storage.Store, q queue.Queue, interval time.Duration, maxImages int, model string) *Scheduler {
	return &Scheduler{store: store, q: q, interval: interval, maxImages: maxImages, model: model}
}

// WithLock 注入分布式锁（如 Redis）。
func (s *Scheduler) WithLock(l Locker) *Scheduler {
	s.lock = l
	return s
}

// Run 阻塞运行，直到 ctx 取消。
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	slog.Info("scheduler started", "interval", s.interval.String())
	for {
		select {
		case <-ticker.C:
			s.AssembleOnce()
		case <-ctx.Done():
			return
		}
	}
}

// AssembleOnce 立即执行一次批次组装（也暴露给 API 做"立即出图"调试入口）。
func (s *Scheduler) AssembleOnce() {
	if s.lock != nil {
		ctx := context.Background()
		if !s.lock.TryAcquire(ctx) {
			slog.Info("scheduler: another instance holds the lock, skipping window")
			return
		}
		defer s.lock.Release(ctx)
	}
	projects, err := s.store.ListProjects()
	if err != nil {
		slog.Error("scheduler: list projects failed", "err", err)
		return
	}
	budget := s.maxImages
	for _, proj := range projects {
		if budget <= 0 {
			slog.Warn("scheduler: batch budget exhausted, remaining shots deferred to next window")
			return
		}
		shotIDs := []string{}
		for i := range proj.Shots {
			if proj.Shots[i].Status == models.ShotQueued && len(shotIDs) < budget {
				shotIDs = append(shotIDs, proj.Shots[i].ID)
			}
		}
		if len(shotIDs) == 0 {
			continue
		}
		budget -= len(shotIDs)

		batch := models.NewBatch(proj.ID, shotIDs, s.model)
		if err := s.store.CreateBatch(batch); err != nil {
			slog.Error("scheduler: create batch failed", "err", err)
			continue
		}
		for i := range proj.Shots {
			for _, id := range shotIDs {
				if proj.Shots[i].ID == id {
					proj.Shots[i].BatchID = batch.ID
				}
			}
		}
		if err := s.store.UpdateProject(proj); err != nil {
			slog.Error("scheduler: update project failed", "err", err)
			continue
		}
		if err := s.q.Publish(queue.TopicImageBatch, queue.Message{
			TaskID: "imgbatch_" + batch.ID, ProjectID: proj.ID, BatchID: batch.ID,
		}); err != nil {
			slog.Error("scheduler: publish failed", "err", err)
			continue
		}
		slog.Info("batch assembled", "batch_id", batch.ID, "project_id", proj.ID, "shots", len(shotIDs))
	}
}
