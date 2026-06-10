package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/models"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/queue"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/storage"
)

func TestAssembleOnce(t *testing.T) {
	store := storage.NewMemory()
	q := queue.NewMemory(1)
	defer q.Close()

	p := models.NewProject("t", "s")
	s1 := models.NewShot(0)
	s1.Status = models.ShotQueued
	s2 := models.NewShot(1) // pending，不应入批
	p.Shots = []models.Shot{s1, s2}
	if err := store.CreateProject(p); err != nil {
		t.Fatal(err)
	}

	got := make(chan queue.Message, 1)
	q.Subscribe(queue.TopicImageBatch, func(_ context.Context, m queue.Message) {
		got <- m
	})

	New(store, q, time.Minute, 50, "wanx").AssembleOnce()

	select {
	case m := <-got:
		if m.ProjectID != p.ID || m.BatchID == "" {
			t.Fatalf("unexpected message: %+v", m)
		}
		b, err := store.GetBatch(m.BatchID)
		if err != nil {
			t.Fatal(err)
		}
		if len(b.ShotIDs) != 1 || b.ShotIDs[0] != s1.ID {
			t.Fatalf("batch should contain only queued shot, got %v", b.ShotIDs)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no image.batch message published")
	}

	updated, _ := store.GetProject(p.ID)
	if updated.Shots[0].BatchID == "" {
		t.Fatal("queued shot should be assigned batch_id")
	}
	if updated.Shots[1].BatchID != "" {
		t.Fatal("pending shot should not be assigned batch_id")
	}
}
