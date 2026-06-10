package queue

import (
	"context"
	"log/slog"
	"sync"
)

// Memory 基于 channel 的进程内队列，带 worker 并发消费与幂等去重。
type Memory struct {
	mu       sync.Mutex
	topics   map[string]chan Message
	seen     map[string]struct{} // task_id 去重（幂等）
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	workers  int
	bufferSz int
}

func NewMemory(workersPerTopic int) *Memory {
	ctx, cancel := context.WithCancel(context.Background())
	return &Memory{
		topics: map[string]chan Message{}, seen: map[string]struct{}{},
		ctx: ctx, cancel: cancel, workers: workersPerTopic, bufferSz: 1024,
	}
}

func (m *Memory) ch(topic string) chan Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.topics[topic]
	if !ok {
		c = make(chan Message, m.bufferSz)
		m.topics[topic] = c
	}
	return c
}

func (m *Memory) Publish(topic string, msg Message) error {
	select {
	case m.ch(topic) <- msg:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

func (m *Memory) Subscribe(topic string, h Handler) {
	c := m.ch(topic)
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			for {
				select {
				case msg := <-c:
					if !m.firstSeen(msg.TaskID) {
						slog.Warn("duplicate task skipped", "task_id", msg.TaskID)
						continue
					}
					h(m.ctx, msg)
				case <-m.ctx.Done():
					return
				}
			}
		}()
	}
}

func (m *Memory) firstSeen(taskID string) bool {
	if taskID == "" {
		return true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.seen[taskID]; ok {
		return false
	}
	m.seen[taskID] = struct{}{}
	return true
}

func (m *Memory) Close() {
	m.cancel()
	m.wg.Wait()
}
