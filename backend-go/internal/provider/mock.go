package provider

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/models"
)

// Mock 实现：无任何 API key 即可跑通全链路（与 Python MVP 的 Mock 模式对齐）。

type MockLLM struct{}

// AnalyzeScript 按空行把剧本拆成分镜，并从文本里抽取简易关键词。
func (MockLLM) AnalyzeScript(_ context.Context, script string, characters []models.Character) ([]ShotDraft, error) {
	blocks := []string{}
	for _, b := range strings.Split(strings.ReplaceAll(script, "\r\n", "\n"), "\n\n") {
		if s := strings.TrimSpace(b); s != "" {
			blocks = append(blocks, s)
		}
	}
	if len(blocks) == 0 {
		blocks = []string{"空剧本占位镜头"}
	}
	drafts := make([]ShotDraft, 0, len(blocks))
	for i, b := range blocks {
		names := []string{}
		for _, c := range characters {
			if c.Name != "" && strings.Contains(b, c.Name) {
				names = append(names, c.Name)
			}
		}
		summary := b
		if r := []rune(b); len(r) > 40 {
			summary = string(r[:40]) + "…"
		}
		drafts = append(drafts, ShotDraft{
			Summary:        summary,
			ImagePrompt:    fmt.Sprintf("电影感画面，镜头%d：%s", i+1, summary),
			Keywords:       mockKeywords(b, names),
			VideoPrompt:    "缓慢推镜，自然光影，人物动作流畅",
			CharacterNames: names,
		})
	}
	return drafts, nil
}

func mockKeywords(block string, names []string) []string {
	kws := append([]string{}, names...)
	kws = append(kws, "电影感", "高清", "剧情")
	if len(kws) > 6 {
		kws = kws[:6]
	}
	return kws
}

type MockImage struct{}

// Generate 生成一张纯色占位 PNG（颜色由 prompt 哈希决定，便于肉眼区分分镜）。
func (MockImage) Generate(_ context.Context, prompt string, _ []string) ([]byte, error) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(prompt))
	v := h.Sum32()
	c := color.RGBA{R: uint8(v), G: uint8(v >> 8), B: uint8(v >> 16), A: 255}
	img := image.NewRGBA(image.Rect(0, 0, 512, 288))
	for y := 0; y < 288; y++ {
		for x := 0; x < 512; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	time.Sleep(200 * time.Millisecond) // 模拟生成耗时
	return buf.Bytes(), nil
}

type MockVideo struct{}

// ImageToVideo 返回占位字节（生产环境为可灵生成的 mp4；本地可用 ffmpeg 把图转短视频）。
func (MockVideo) ImageToVideo(_ context.Context, imagePath, motionPrompt string, durationSec int) ([]byte, error) {
	time.Sleep(500 * time.Millisecond) // 模拟生成耗时
	return []byte(fmt.Sprintf("MOCK_VIDEO src=%s prompt=%q duration=%ds", imagePath, motionPrompt, durationSec)), nil
}
