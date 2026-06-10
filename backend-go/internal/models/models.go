// Package models 定义核心业务数据模型（与 docs/architecture.md 对应）。
package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func newID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

// ShotStatus 分镜状态机：
// pending -> queued -> image_running -> image_review -> video_running -> done / failed
type ShotStatus string

const (
	ShotPending      ShotStatus = "pending"
	ShotQueued       ShotStatus = "queued"
	ShotImageRunning ShotStatus = "image_running"
	ShotImageReview  ShotStatus = "image_review" // 出图完成，等待用户确认（质量关卡一）
	ShotVideoRunning ShotStatus = "video_running"
	ShotDone         ShotStatus = "done"
	ShotFailed       ShotStatus = "failed"
)

type BatchStatus string

const (
	BatchPending BatchStatus = "pending"
	BatchRunning BatchStatus = "running"
	BatchDone    BatchStatus = "done"
	BatchFailed  BatchStatus = "failed"
)

// Character 模特主体（角色资产包的最小版本，详见 docs/quality-system.md）。
type Character struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	RefImages   []string `json:"ref_images"` // 多角度参考图 URL/路径
}

// Shot 分镜。keywords 为一等字段：图片下方展示的关键词。
type Shot struct {
	ID           string     `json:"id"`
	Index        int        `json:"index"`
	Summary      string     `json:"summary"`
	ImagePrompt  string     `json:"image_prompt"`
	Keywords     []string   `json:"keywords"`
	VideoPrompt  string     `json:"video_prompt"`
	CharacterIDs []string   `json:"character_ids"`
	ImagePath    string     `json:"image_path,omitempty"`
	VideoPath    string     `json:"video_path,omitempty"`
	BatchID      string     `json:"batch_id,omitempty"`
	Status       ShotStatus `json:"status"`
	Error        string     `json:"error,omitempty"`
}

type Project struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Script     string      `json:"script"`
	Characters []Character `json:"characters"`
	Shots      []Shot      `json:"shots"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// GenerationBatch 出图批次：每个调度窗口（默认 10 分钟）聚合的待出图分镜。
type GenerationBatch struct {
	ID          string      `json:"id"`
	ProjectID   string      `json:"project_id"`
	ShotIDs     []string    `json:"shot_ids"`
	Status      BatchStatus `json:"status"`
	Model       string      `json:"model"`
	WindowStart time.Time   `json:"window_start"`
	CreatedAt   time.Time   `json:"created_at"`
	FinishedAt  *time.Time  `json:"finished_at,omitempty"`
}

func NewProject(title, script string) *Project {
	now := time.Now()
	return &Project{ID: newID("proj"), Title: title, Script: script,
		Characters: []Character{}, Shots: []Shot{}, CreatedAt: now, UpdatedAt: now}
}

func NewCharacter(name, description string) Character {
	return Character{ID: newID("char"), Name: name, Description: description, RefImages: []string{}}
}

func NewShot(index int) Shot {
	return Shot{ID: newID("shot"), Index: index, Keywords: []string{}, CharacterIDs: []string{}, Status: ShotPending}
}

func NewBatch(projectID string, shotIDs []string, model string) *GenerationBatch {
	now := time.Now()
	return &GenerationBatch{ID: newID("batch"), ProjectID: projectID, ShotIDs: shotIDs,
		Status: BatchPending, Model: model, WindowStart: now, CreatedAt: now}
}
