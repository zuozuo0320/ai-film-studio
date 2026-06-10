// 开发模式一体化入口：API + worker + 调度器跑在同一进程（内存存储/队列）。
// 生产环境按 docs/backend-architecture-go.md 拆为独立服务（PG/Kafka/Redis 实现
// 替换 storage/queue/ws 的内存版即可，业务代码不变）。
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/api"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/config"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/pipeline"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/provider"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/queue"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/scheduler"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/storage"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/ws"
)

func main() {
	cfg := config.Load()
	slog.Info("starting ai-film-studio backend (Go)",
		"addr", cfg.Addr, "batch_interval", cfg.BatchInterval.String(), "wanx_model", cfg.WanxModel)

	store := storage.NewMemory()
	q := queue.NewMemory(cfg.QueueWorkersPerTop)
	defer q.Close()
	hub := ws.NewHub()

	// provider 选择：有 key 用真实实现（待接入），否则 Mock（与 Python MVP 行为一致）
	var llm provider.LLM = provider.MockLLM{}
	var img provider.Image = provider.MockImage{}
	var vid provider.Video = provider.MockVideo{}
	if cfg.DashScopeAPIKey != "" {
		slog.Warn("DASHSCOPE_API_KEY 已配置，但万相 provider 尚未实现，暂用 Mock")
	}
	if cfg.KlingAccessKey != "" {
		slog.Warn("KLING_ACCESS_KEY 已配置，但可灵 provider 尚未实现，暂用 Mock")
	}

	pipe := pipeline.New(store, hub, llm, img, vid, cfg.DataDir, cfg.VideoDurationSec)
	pipe.Register(q)

	sched := scheduler.New(store, q, cfg.BatchInterval, cfg.BatchMaxImages, cfg.WanxModel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sched.Run(ctx)

	srv := api.NewServer(store, q, hub, sched, cfg.DataDir)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
		os.Exit(0)
	}()

	if err := srv.Router().Run(cfg.Addr); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}
