package ws

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

const progressChannel = "progress.events"

// RedisBus 基于 Redis Pub/Sub 的事件总线，支持多实例部署：
// 任意实例 Publish，所有实例的本地订阅者都能收到。
type RedisBus struct {
	rdb   *redis.Client
	local *Hub // 本机订阅者管理复用 Hub
	stop  context.CancelFunc
}

func NewRedisBus(rdb *redis.Client) *RedisBus {
	ctx, cancel := context.WithCancel(context.Background())
	b := &RedisBus{rdb: rdb, local: NewHub(), stop: cancel}
	sub := rdb.Subscribe(ctx, progressChannel)
	go func() {
		ch := sub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var e Event
				if err := json.Unmarshal([]byte(msg.Payload), &e); err != nil {
					slog.Warn("redis bus: bad event payload", "err", err)
					continue
				}
				b.local.Publish(e)
			case <-ctx.Done():
				_ = sub.Close()
				return
			}
		}
	}()
	return b
}

func (b *RedisBus) Subscribe(projectID string) (<-chan Event, func()) {
	return b.local.Subscribe(projectID)
}

func (b *RedisBus) Publish(e Event) {
	data, _ := json.Marshal(e)
	if err := b.rdb.Publish(context.Background(), progressChannel, data).Err(); err != nil {
		slog.Error("redis bus: publish failed", "err", err)
	}
}

func (b *RedisBus) Close() { b.stop() }
