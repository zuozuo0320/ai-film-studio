// Package queue 任务队列抽象。生产环境实现 Kafka 版本（见
// docs/backend-architecture-go.md），本骨架提供内存 channel 实现。
package queue

import "context"

// 任务 topic（与架构文档一致）。
const (
	TopicScriptAnalyze = "script.analyze"
	TopicImageBatch    = "image.batch"
	TopicVideoGenerate = "video.generate"
)

// Message 队列消息。TaskID 用于幂等去重。
type Message struct {
	TaskID    string `json:"task_id"`
	ProjectID string `json:"project_id"`
	ShotID    string `json:"shot_id,omitempty"`
	BatchID   string `json:"batch_id,omitempty"`
}

type Handler func(ctx context.Context, msg Message)

type Queue interface {
	Publish(topic string, msg Message) error
	// Subscribe 注册 handler，由实现负责并发消费。
	Subscribe(topic string, h Handler)
	Close()
}
