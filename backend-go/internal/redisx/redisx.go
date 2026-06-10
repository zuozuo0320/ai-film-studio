// Package redisx Redis 基础设施：分布式锁、任务幂等去重、配额计数。
package redisx

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewClient(addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return rdb, nil
}

// Lock 简单分布式锁（SET NX EX）。scheduler 多副本时保证单活组装批次。
type Lock struct {
	rdb *redis.Client
	key string
	ttl time.Duration
}

func NewLock(rdb *redis.Client, key string, ttl time.Duration) *Lock {
	return &Lock{rdb: rdb, key: key, ttl: ttl}
}

// TryAcquire 抢锁成功返回 true。锁到期自动释放，不做续期（批次组装耗时远小于 TTL）。
func (l *Lock) TryAcquire(ctx context.Context) bool {
	ok, err := l.rdb.SetNX(ctx, l.key, "1", l.ttl).Result()
	return err == nil && ok
}

func (l *Lock) Release(ctx context.Context) {
	_ = l.rdb.Del(ctx, l.key).Err()
}

// Deduper 任务幂等去重：task_id 首次出现返回 true，跨实例共享。
type Deduper struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewDeduper(rdb *redis.Client, ttl time.Duration) *Deduper {
	return &Deduper{rdb: rdb, ttl: ttl}
}

func (d *Deduper) FirstSeen(taskID string) bool {
	if taskID == "" {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ok, err := d.rdb.SetNX(ctx, "task.seen:"+taskID, "1", d.ttl).Result()
	if err != nil {
		// Redis 异常时放行（宁可重复执行，不可丢任务）
		return true
	}
	return ok
}

// Quota 每日配额计数器（原子 INCR，午夜过期）。
type Quota struct {
	rdb *redis.Client
}

func NewQuota(rdb *redis.Client) *Quota {
	return &Quota{rdb: rdb}
}

// Incr 计数 +n 并返回当日累计值。
func (q *Quota) Incr(ctx context.Context, scope, id string, n int64) (int64, error) {
	key := fmt.Sprintf("quota:%s:%s:%s", scope, id, time.Now().Format("20060102"))
	pipe := q.rdb.TxPipeline()
	incr := pipe.IncrBy(ctx, key, n)
	pipe.Expire(ctx, key, 48*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
