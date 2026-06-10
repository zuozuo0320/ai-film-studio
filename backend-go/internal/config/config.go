// Package config 环境变量配置。
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr               string        // 监听地址
	DataDir            string        // 媒体输出目录（生产为 OSS）
	BatchInterval      time.Duration // 出图批次窗口，默认 10 分钟
	BatchMaxImages     int           // 单批上限
	VideoDurationSec   int
	WanxModel          string // 通义万相模型名
	DashScopeAPIKey    string // 留空则 Mock
	KlingAccessKey     string
	KlingSecretKey     string
	QueueWorkersPerTop int
	DatabaseURL        string // 留空则内存存储
	RedisAddr          string // 留空则进程内 Hub/锁/去重
	QuotaDailyImages   int64  // 每项目每日出图配额（0=不限，需 Redis）
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func Load() Config {
	return Config{
		Addr:               envStr("ADDR", ":8080"),
		DataDir:            envStr("DATA_DIR", "data"),
		BatchInterval:      time.Duration(envInt("BATCH_INTERVAL_MINUTES", 10)) * time.Minute,
		BatchMaxImages:     envInt("BATCH_MAX_IMAGES", 50),
		VideoDurationSec:   envInt("VIDEO_DURATION_SEC", 5),
		WanxModel:          envStr("WANX_MODEL", "wanx2.1-t2i-pro"),
		DashScopeAPIKey:    os.Getenv("DASHSCOPE_API_KEY"),
		KlingAccessKey:     os.Getenv("KLING_ACCESS_KEY"),
		KlingSecretKey:     os.Getenv("KLING_SECRET_KEY"),
		QueueWorkersPerTop: envInt("QUEUE_WORKERS", 4),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		RedisAddr:          os.Getenv("REDIS_ADDR"),
		QuotaDailyImages:   int64(envInt("QUOTA_DAILY_IMAGES", 0)),
	}
}
