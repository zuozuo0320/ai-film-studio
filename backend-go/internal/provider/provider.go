// Package provider 外部模型适配层（provider-sdk 的最小版本）。
// 生产环境在此实现 DashScope 万相、可灵 Kling、LLM 的真实调用，
// 并叠加限流/熔断/重试/计费埋点（见 docs/backend-architecture-go.md 4.2）。
package provider

import (
	"context"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/models"
)

// ShotDraft LLM 剧本分析产出的单镜草稿。
type ShotDraft struct {
	Summary        string   `json:"summary"`
	ImagePrompt    string   `json:"image_prompt"`
	Keywords       []string `json:"keywords"`
	VideoPrompt    string   `json:"video_prompt"`
	CharacterNames []string `json:"character_names"`
}

// LLM 剧本分析（拆分镜 + 生成提示词/关键词）。
type LLM interface {
	AnalyzeScript(ctx context.Context, script string, characters []models.Character) ([]ShotDraft, error)
}

// Image 出图（通义万相 Pro）。返回图片字节（生产环境直接落 OSS 返回 URL）。
type Image interface {
	Generate(ctx context.Context, prompt string, refImages []string) ([]byte, error)
}

// Video 图生视频（可灵 Kling）。
type Video interface {
	ImageToVideo(ctx context.Context, imagePath string, motionPrompt string, durationSec int) ([]byte, error)
}
