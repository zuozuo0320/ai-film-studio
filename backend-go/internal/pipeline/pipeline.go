// Package pipeline 生成流水线 worker：剧本分析 / 批量出图 / 图生视频。
// 消费队列消息，调用 provider，结果写回 storage，并通过 ws.Hub 推送进度。
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/models"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/provider"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/queue"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/storage"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/ws"
)

type Pipeline struct {
	store    storage.Store
	hub      *ws.Hub
	llm      provider.LLM
	image    provider.Image
	video    provider.Video
	dataDir  string
	videoSec int
}

func New(store storage.Store, hub *ws.Hub, llm provider.LLM, img provider.Image, vid provider.Video, dataDir string, videoSec int) *Pipeline {
	return &Pipeline{store: store, hub: hub, llm: llm, image: img, video: vid, dataDir: dataDir, videoSec: videoSec}
}

// Register 把三类 worker 挂到队列上。
func (p *Pipeline) Register(q queue.Queue) {
	q.Subscribe(queue.TopicScriptAnalyze, p.handleAnalyze)
	q.Subscribe(queue.TopicImageBatch, p.handleImageBatch)
	q.Subscribe(queue.TopicVideoGenerate, p.handleVideo)
}

// ---------- 剧本分析 ----------

func (p *Pipeline) handleAnalyze(ctx context.Context, msg queue.Message) {
	proj, err := p.store.GetProject(msg.ProjectID)
	if err != nil {
		slog.Error("analyze: project not found", "project_id", msg.ProjectID)
		return
	}
	drafts, err := p.llm.AnalyzeScript(ctx, proj.Script, proj.Characters)
	if err != nil {
		slog.Error("analyze failed", "err", err)
		return
	}
	nameToID := map[string]string{}
	for _, c := range proj.Characters {
		nameToID[c.Name] = c.ID
	}
	shots := make([]models.Shot, 0, len(drafts))
	for i, d := range drafts {
		s := models.NewShot(i)
		s.Summary, s.ImagePrompt, s.Keywords, s.VideoPrompt = d.Summary, d.ImagePrompt, d.Keywords, d.VideoPrompt
		for _, n := range d.CharacterNames {
			if id, ok := nameToID[n]; ok {
				s.CharacterIDs = append(s.CharacterIDs, id)
			}
		}
		shots = append(shots, s)
	}
	proj.Shots = shots
	if err := p.store.UpdateProject(proj); err != nil {
		slog.Error("analyze: save failed", "err", err)
		return
	}
	p.hub.Publish(ws.Event{Type: "project_analyzed", ProjectID: proj.ID, Status: "done"})
}

// ---------- 批量出图 ----------

func (p *Pipeline) handleImageBatch(ctx context.Context, msg queue.Message) {
	batch, err := p.store.GetBatch(msg.BatchID)
	if err != nil {
		slog.Error("image batch not found", "batch_id", msg.BatchID)
		return
	}
	batch.Status = models.BatchRunning
	_ = p.store.UpdateBatch(batch)
	p.publishBatch(batch)

	failed := 0
	for _, shotID := range batch.ShotIDs {
		if err := p.generateImage(ctx, batch.ProjectID, shotID); err != nil {
			failed++
		}
	}
	if len(batch.ShotIDs) > 0 && failed == len(batch.ShotIDs) {
		batch.Status = models.BatchFailed
	} else {
		batch.Status = models.BatchDone
	}
	t := time.Now()
	batch.FinishedAt = &t
	_ = p.store.UpdateBatch(batch)
	p.publishBatch(batch)
}

func (p *Pipeline) generateImage(ctx context.Context, projectID, shotID string) error {
	proj, shot, err := p.findShot(projectID, shotID)
	if err != nil {
		return err
	}
	p.setShotStatus(proj, shot, models.ShotImageRunning, "")

	var refs []string
	for _, c := range proj.Characters {
		for _, id := range shot.CharacterIDs {
			if c.ID == id {
				refs = append(refs, c.RefImages...)
			}
		}
	}
	data, err := p.image.Generate(ctx, shot.ImagePrompt, refs)
	if err != nil {
		p.setShotStatus(proj, shot, models.ShotFailed, fmt.Sprintf("出图失败: %v", err))
		return err
	}
	rel := filepath.Join("outputs", shot.ID+".png")
	if err := p.writeFile(rel, data); err != nil {
		p.setShotStatus(proj, shot, models.ShotFailed, fmt.Sprintf("写文件失败: %v", err))
		return err
	}
	shot.ImagePath = rel
	p.setShotStatus(proj, shot, models.ShotImageReview, "")
	return nil
}

// ---------- 图生视频 ----------

func (p *Pipeline) handleVideo(ctx context.Context, msg queue.Message) {
	proj, shot, err := p.findShot(msg.ProjectID, msg.ShotID)
	if err != nil {
		slog.Error("video: shot not found", "shot_id", msg.ShotID)
		return
	}
	if shot.ImagePath == "" {
		p.setShotStatus(proj, shot, models.ShotFailed, "请先出图再生成视频")
		return
	}
	p.setShotStatus(proj, shot, models.ShotVideoRunning, "")

	data, err := p.video.ImageToVideo(ctx, filepath.Join(p.dataDir, shot.ImagePath), shot.VideoPrompt, p.videoSec)
	if err != nil {
		p.setShotStatus(proj, shot, models.ShotFailed, fmt.Sprintf("图生视频失败: %v", err))
		return
	}
	rel := filepath.Join("outputs", shot.ID+".mp4")
	if err := p.writeFile(rel, data); err != nil {
		p.setShotStatus(proj, shot, models.ShotFailed, fmt.Sprintf("写文件失败: %v", err))
		return
	}
	shot.VideoPath = rel
	p.setShotStatus(proj, shot, models.ShotDone, "")
}

// ---------- helpers ----------

func (p *Pipeline) findShot(projectID, shotID string) (*models.Project, *models.Shot, error) {
	proj, err := p.store.GetProject(projectID)
	if err != nil {
		return nil, nil, err
	}
	for i := range proj.Shots {
		if proj.Shots[i].ID == shotID {
			return proj, &proj.Shots[i], nil
		}
	}
	return nil, nil, storage.ErrNotFound
}

// setShotStatus 更新分镜状态并落库 + 推送事件。
// 注意 shot 是 proj.Shots 内的指针，UpdateProject 持久化整个项目。
func (p *Pipeline) setShotStatus(proj *models.Project, shot *models.Shot, st models.ShotStatus, errMsg string) {
	shot.Status = st
	shot.Error = errMsg
	if err := p.store.UpdateProject(proj); err != nil {
		slog.Error("save shot status failed", "err", err)
	}
	p.hub.Publish(ws.Event{Type: "shot_update", ProjectID: proj.ID, ShotID: shot.ID, Status: string(st), Error: errMsg})
}

func (p *Pipeline) publishBatch(b *models.GenerationBatch) {
	p.hub.Publish(ws.Event{Type: "batch_update", ProjectID: b.ProjectID, BatchID: b.ID, Status: string(b.Status)})
}

func (p *Pipeline) writeFile(rel string, data []byte) error {
	abs := filepath.Join(p.dataDir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, 0o644)
}
