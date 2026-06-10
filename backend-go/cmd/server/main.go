// 开发模式一体化入口：API + worker + 调度器跑在同一进程。
// 配置 DATABASE_URL 启用 PostgreSQL 存储，配置 REDIS_ADDR 启用 Redis
// （进度 Pub/Sub / 调度器分布式锁 / 任务幂等去重 / 配额计数）；
// 两者留空则回退内存实现（零依赖开发模式）。
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/api"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/config"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/pipeline"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/provider"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/queue"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/redisx"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/scheduler"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/storage"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/ws"
)

func main() {
	cfg := config.Load()
	slog.Info("starting ai-film-studio backend (Go)",
		"addr", cfg.Addr, "batch_interval", cfg.BatchInterval.String(), "wanx_model", cfg.WanxModel)

	var store storage.Store = storage.NewMemory()
	if cfg.DatabaseURL != "" {
		pg, err := storage.NewPostgres(context.Background(), cfg.DatabaseURL)
		if err != nil {
			slog.Error("postgres 连接失败", "err", err)
			os.Exit(1)
		}
		defer pg.Close()
		store = pg
		slog.Info("storage: PostgreSQL")
	} else {
		slog.Info("storage: memory (设置 DATABASE_URL 启用 PostgreSQL)")
	}

	q := queue.NewMemory(cfg.QueueWorkersPerTop)
	defer q.Close()

	var hub ws.Bus = ws.NewHub()
	var quota *redisx.Quota
	var lock *redisx.Lock
	if cfg.RedisAddr != "" {
		rdb, err := redisx.NewClient(cfg.RedisAddr)
		if err != nil {
			slog.Error("redis 连接失败", "err", err)
			os.Exit(1)
		}
		bus := ws.NewRedisBus(rdb)
		defer bus.Close()
		hub = bus
		q.WithDeduper(redisx.NewDeduper(rdb, 24*time.Hour))
		quota = redisx.NewQuota(rdb)
		lock = redisx.NewLock(rdb, "lock:scheduler.assemble", 2*time.Minute)
		slog.Info("redis: 已启用 Pub/Sub + 分布式锁 + 幂等去重 + 配额计数")
	} else {
		slog.Info("redis: 未配置，使用进程内实现 (设置 REDIS_ADDR 启用)")
	}

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
	if lock != nil {
		sched.WithLock(lock)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sched.Run(ctx)

	srv := api.NewServer(store, q, hub, sched, cfg.DataDir)
	if quota != nil && cfg.QuotaDailyImages > 0 {
		srv.WithQuota(quota, cfg.QuotaDailyImages)
	}

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
