// Package ws 进度事件 Hub：worker 发布事件，WebSocket/轮询客户端订阅。
// 生产环境此处为 Redis Pub/Sub + 独立 ws-gateway（见架构文档 4.4）。
package ws

import "sync"

// Event 生成进度事件，推送给前端。
type Event struct {
	Type      string `json:"type"` // shot_update / batch_update
	ProjectID string `json:"project_id"`
	ShotID    string `json:"shot_id,omitempty"`
	BatchID   string `json:"batch_id,omitempty"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

type Hub struct {
	mu   sync.Mutex
	subs map[chan Event]string // chan -> projectID 过滤（空串=全部）
}

func NewHub() *Hub {
	return &Hub{subs: map[chan Event]string{}}
}

// Subscribe 返回事件 channel 与取消函数。projectID 为空表示订阅全部。
func (h *Hub) Subscribe(projectID string) (<-chan Event, func()) {
	ch := make(chan Event, 64)
	h.mu.Lock()
	h.subs[ch] = projectID
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
	return ch, cancel
}

// Publish 非阻塞广播；慢消费者直接丢弃事件（前端以服务端状态为准，可轮询补偿）。
func (h *Hub) Publish(e Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch, pid := range h.subs {
		if pid != "" && pid != e.ProjectID {
			continue
		}
		select {
		case ch <- e:
		default:
		}
	}
}
