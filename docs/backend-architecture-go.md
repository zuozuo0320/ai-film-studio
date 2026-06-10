# Go 后端架构设计（面向百万级用户）

> 技术决策：Web 优先路线，后端采用 **Go**。本文档定义生产级后端架构，目标支撑百万注册用户 / 十万级 DAU 的 2C AI 影视创作平台。
>
> 与 [architecture.md](./architecture.md)（整体业务架构）、[quality-system.md](./quality-system.md)（质量体系）配套；本文专注后端工程实现。

---

## 1. 设计目标与容量假设

| 指标 | 目标 |
| --- | --- |
| 注册用户 | 100 万+ |
| DAU | 5–10 万 |
| API 峰值 QPS | 5,000+（读多写少，读写比约 9:1） |
| 并发生成任务 | 出图数千/小时、视频数百/小时（受上游模型 API 配额约束，瓶颈在外部） |
| 可用性 | 核心 API 99.9% |
| 延迟 | API P99 < 300ms（生成类全部异步，不计入） |

核心特征：**业务逻辑轻、I/O 重、强依赖外部模型 API**。架构重点不在算力，而在**任务编排、限流配额、状态一致性、媒体分发**。

---

## 2. 总体架构

```
                         ┌──────────────────────────────┐
                         │   CDN（图片/视频/前端静态资源）   │
                         └──────────────┬───────────────┘
 用户 ──HTTPS──► SLB/负载均衡 ──► API 网关层（Nginx/APISIX：TLS、WAF、全局限流）
                                        │
                  ┌─────────────────────┼─────────────────────────┐
                  ▼                     ▼                         ▼
          ┌──────────────┐      ┌──────────────┐         ┌──────────────────┐
          │ api-gateway   │      │ ws-gateway   │         │ media-svc        │
          │ (Go) REST API │      │ (Go) WebSocket│        │ (Go) 上传凭证/转码 │
          │ 鉴权/配额/CRUD │      │ 进度实时推送    │         │ 回调/签名 URL     │
          └──────┬───────┘      └──────┬───────┘         └────────┬─────────┘
                 │                     │                          │
                 │              ┌──────▼───────┐                  │
                 │              │ Redis Cluster │◄────────────────┤
                 │              │ 缓存/会话/Pub-Sub│                │
                 │              └──────┬───────┘                  │
                 ▼                     │                          ▼
          ┌──────────────┐             │                  ┌──────────────┐
          │ PostgreSQL    │             │                  │ OSS 对象存储   │
          │ 主从 + PgBouncer│            │                  │ (剧本/图/视频) │
          └──────────────┘             │                  └──────────────┘
                 ▲                     │
                 │              ┌──────▼──────────────────────────────┐
                 │              │ 消息队列 Kafka（或 Pulsar）            │
                 │              │ topics: script.analyze / image.batch │
                 │              │         video.generate / qc.check    │
                 │              └──────┬──────────────────────────────┘
                 │                     │
        ┌────────┴─────────────────────┼────────────────────────────┐
        ▼                              ▼                            ▼
┌───────────────┐            ┌─────────────────┐          ┌─────────────────┐
│ analyze-worker │            │ image-worker     │          │ video-worker     │
│ (Go) LLM 剧本   │            │ (Go) 万相批量出图  │          │ (Go) 可灵图生视频  │
│ 分析编排        │            │ + 一致性校验/评分  │          │ + 视频质检        │
└───────────────┘            └─────────────────┘          └─────────────────┘
        │                              │                            │
        └──────────────► 外部模型 API（LLM / DashScope 万相 / 可灵 Kling）
                          统一经 provider-sdk（限流/熔断/重试/计费埋点）

┌────────────────────┐  ┌─────────────────────┐  ┌──────────────────────┐
│ scheduler (Go)      │  │ billing-svc (Go)     │  │ 可观测性               │
│ 10分钟批次组装/超时回收│  │ 配额/计量/订单/支付    │  │ Prometheus + Grafana  │
│ (分布式锁选主)        │  │                     │  │ + OpenTelemetry + 日志 │
└────────────────────┘  └─────────────────────┘  └──────────────────────┘
```

---

## 3. 服务拆分（适度微服务，避免过度拆分）

百万级用户不需要几十个微服务。按**变更频率与扩缩容特征**拆为 6 个 Go 服务：

| 服务 | 职责 | 扩缩容特征 |
| --- | --- | --- |
| `api-gateway` | REST API：用户/项目/剧本/角色/分镜 CRUD、鉴权、配额校验、任务投递 | 无状态，按 QPS 水平扩 |
| `ws-gateway` | WebSocket 长连接，订阅 Redis Pub/Sub 推送生成进度 | 按连接数扩，需会话粘性 |
| `media-svc` | OSS 直传凭证签发、上传回调、签名 URL、缩略图/转码触发 | 无状态 |
| `worker`（3 类） | analyze / image / video 消费者，调用外部模型 API | 按队列积压(lag)自动扩缩 |
| `scheduler` | 10 分钟批次组装、超时任务回收、对账 | 单活（分布式锁选主），冷备 |
| `billing-svc` | 配额、计量、订单、支付回调 | 无状态，强一致性要求 |

**Go 技术选型**：

- HTTP 框架：`gin`（或 `echo`）+ `protovalidate`；内部服务间通信 gRPC（`connect-go`）
- ORM：`sqlc`（编译期生成类型安全 SQL，性能好、无魔法）或 `ent`
- 配置/服务发现：Kubernetes + ConfigMap；服务网格暂不需要
- 任务队列消费：`franz-go`（Kafka 客户端）
- 定时与分布式锁：`redsync`（Redis 红锁）选主
- 部署：全部容器化，K8s + HPA（api 按 CPU/QPS，worker 按 Kafka lag）

---

## 4. 关键设计

### 4.1 异步任务编排（核心）

所有生成类操作（剧本分析/出图/视频）一律异步，Kafka 作为任务总线：

```
api-gateway：写 DB（任务记录，状态 queued）→ 发 Kafka 消息（事务性发件箱模式 outbox，
             保证 DB 与 MQ 一致）→ 立即返回 task_id
worker：     消费消息 → 幂等检查（task_id 去重）→ 调外部 API → 结果写 OSS + DB
             → 发进度事件到 Redis Pub/Sub → ws-gateway 推送给前端
scheduler：  扫描超时任务（worker 崩溃/外部 API 挂起）→ 重置为 queued 重新入队
```

要点：

- **Outbox 模式**保证"DB 状态与队列消息"不会不一致（先写本地 outbox 表，独立 relay 进程投递 Kafka）。
- **幂等**：所有任务带唯一 `task_id`，worker 处理前 `SETNX`；外部 API 调用带幂等键，重试不重复扣费。
- **优先级**：Kafka 按 topic 分高/普通优先级（用户"立即出图"走高优 topic，10 分钟批次走普通 topic）。

### 4.2 外部模型 API 治理（provider-sdk）

统一的 Go SDK 封装 LLM / 万相 / 可灵，所有 worker 通过它调用：

- **限流**：每个 provider 全局令牌桶（Redis 实现，跨 worker 实例共享），贴着官方 QPS/并发上限跑
- **熔断**：`sony/gobreaker`，上游持续 5xx 时快速失败并降级排队
- **重试**：指数退避 + 抖动，仅对幂等操作重试；429 读 Retry-After
- **计费埋点**：每次调用记录 tokens/张数/秒数 → billing-svc 计量
- **多账号池**：单账号配额不够时，多 API 账号轮转（按剩余配额加权）

### 4.3 数据层

**PostgreSQL（主存储）**

- 一主多从，读写分离（读多写少场景从库扛读流量）；PgBouncer 连接池
- 百万用户量级**单库足够**（核心表行数：users 1M、projects ~5M、shots ~100M）；shots 表按 `project_id` HASH 分区预留扩展性，**不要过早分库分表**
- 任务/事件类高写入表（generation_events、billing_records）按月分区，定期归档

**Redis Cluster**

- 缓存：用户会话、项目/分镜热数据（cache-aside，TTL + 主动失效）
- 配额计数器：用户日/月配额用 Redis 原子计数，异步对账落 DB
- Pub/Sub：生成进度事件分发给 ws-gateway
- 分布式锁：scheduler 选主、任务幂等

**OSS + CDN**

- 客户端**直传 OSS**（media-svc 签发 STS 临时凭证），不经过后端转发，节省带宽
- 所有媒体走 CDN，私有内容用 CDN 鉴权签名 URL（防盗链 + 按用户隔离）
- 生命周期：候选图（未选中）30 天降冷存储，成品长期保存

### 4.4 实时进度推送

```
worker 产生进度事件 ──► Redis Pub/Sub（channel: progress.{user_id}）
ws-gateway：用户连接时订阅自己的 channel，事件直推浏览器
降级：WebSocket 不可用时前端自动降级轮询 GET /tasks/{id}
```

ws-gateway 水平扩展：连接分布在多实例，每实例只订阅本机在线用户的 channel（订阅管理用本地表 + Redis 在线状态）。

### 4.5 鉴权与安全

- 登录：手机号验证码 + 微信扫码；JWT（短期 access + 长期 refresh），refresh token 存 Redis 可吊销
- 全局限流：网关层按 IP + 用户双维度；注册/验证码接口加图形验证 + 频控（2C 防刷必备）
- 内容安全：剧本文本、生成图/视频送审（阿里云内容安全），异步审核流水线，违规拦截并标记

### 4.6 可观测性

- 指标：Prometheus + Grafana（API QPS/延迟、Kafka lag、外部 API 成功率与配额水位、批次时延）
- 链路：OpenTelemetry 全链路 trace（一次"出图"从 API → Kafka → worker → 万相 → OSS 全程可追）
- 日志：结构化日志（zap）→ Loki/ELK；关键告警：队列积压、外部 API 错误率、配额耗尽预警

---

## 5. 从 MVP（Python）到 Go 的迁移路径

现有 FastAPI MVP 不必推倒重来，按"绞杀者模式"渐进迁移：

1. **Phase 1**：Go 实现 `api-gateway`（用户/项目/CRUD）+ PostgreSQL，替换 JSON 文件存储；Python 流水线暂保留，作为内部 worker 被 Go 调用
2. **Phase 2**：Go 实现 worker + provider-sdk（万相/可灵/LLM），引入 Kafka 与 scheduler，下线 Python
3. **Phase 3**：ws-gateway、billing-svc、内容安全、多账号池、全链路可观测

每阶段都可独立上线，前端 API 契约（OpenAPI 定义）保持稳定。

---

## 6. 容量与成本要点

- 后端本身极轻：Go 服务 + PG + Redis + Kafka 支撑 10 万 DAU 约需 10–20 台中配容器节点，**成本大头是模型 API 调用费与 OSS/CDN 流量**，约占总成本 80%+
- 因此架构上的"省钱设计"优先级：质量关卡拦截差图再生视频（见 quality-system.md）> 候选图冷存储 > CDN 缓存命中率 > 服务器规格
