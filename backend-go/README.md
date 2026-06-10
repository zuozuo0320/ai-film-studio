# backend-go

Go 后端（按 [docs/backend-architecture-go.md](../docs/backend-architecture-go.md) 实现）。
当前为**可运行骨架**：内存存储/队列 + Mock provider，单进程跑通
「剧本分析 → 批次出图（默认 10 分钟窗口）→ 图片确认 → 图生视频」全链路。

## 运行

```bash
cd backend-go
go run ./cmd/server          # 默认 :8080，无需任何 API key（Mock 模式）
```

环境变量：`ADDR` `DATA_DIR` `BATCH_INTERVAL_MINUTES`(默认10) `BATCH_MAX_IMAGES`
`WANX_MODEL` `DASHSCOPE_API_KEY` `KLING_ACCESS_KEY/SECRET_KEY` `VIDEO_DURATION_SEC`。

## API（/api/v1）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | /projects | 创建项目（title, script） |
| GET | /projects, /projects/:id | 列表 / 详情 |
| PATCH/DELETE | /projects/:id | 更新 / 删除 |
| POST | /projects/:id/characters | 添加模特主体 |
| POST | /projects/:id/analyze | AI 剧本分析（异步，拆分镜+关键词） |
| POST | /projects/:id/shots/:shotID/enqueue-image | 分镜入队，等下个批次窗口出图 |
| POST | /projects/:id/shots/:shotID/approve | 质量关卡一：确认图片 |
| POST | /projects/:id/shots/:shotID/video | 图生视频（异步） |
| GET | /projects/:id/batches | 出图批次列表 |
| POST | /debug/assemble-batch | 调试：立即组装批次（不等窗口） |
| GET | /ws?project_id=… | WebSocket 进度推送 |
| GET | /media/* | 生成的图片/视频 |

## 架构映射

| 包 | 对应架构组件 | 生产替换 |
| --- | --- | --- |
| internal/storage | PostgreSQL | 实现 Store 接口的 PG 版本 |
| internal/queue | Kafka | 实现 Queue 接口的 Kafka 版本 |
| internal/ws | Redis Pub/Sub + ws-gateway | Hub 换 Redis 实现 |
| internal/provider | provider-sdk（万相/可灵/LLM） | 实现真实 API + 限流熔断 |
| internal/scheduler | scheduler 服务（10 分钟批次） | 加分布式锁选主 |
| internal/pipeline | 三类 worker | 独立部署，按队列积压扩缩 |

接口已抽象，替换实现不改业务代码。

## 测试

```bash
go test ./...
go vet ./...
```
