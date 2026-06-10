# AI 影视创作平台 系统架构设计文档

> 面向 2C 的 AI 影视创作平台。核心链路：**上传剧本 → 创建模特主体（人物） → AI 自动分析剧本 → 定时（每 10 分钟）批量调用通义万相 Pro 出图（每张图附关键词） → 调用可灵（Kling）API 图生视频片段 → 故事板预览/导出**。
>
> 本文档给出从当前 MVP（本仓库代码）演进到可上线 2C 产品的完整架构设计。

---

## 1. 产品核心逻辑

1. 用户上传/粘贴**剧本**（支持 txt / docx / pdf，或直接粘贴）。
2. 用户创建**模特主体**：人物名字、外貌描述、参考图（用于人物形象一致性）。
3. 平台调用 **LLM 自动分析剧本**：拆分场景与分镜（Shot），为每个分镜生成：
   - 剧情概述（演什么）
   - **出图提示词 + 关键词列表**（图片下方展示的关键词即来自这里）
   - 运镜/动作提示词（供图生视频用）
   - 关联的模特主体
4. **批量出图调度器**：每 **10 分钟**为一个生成窗口，将待出图的分镜打包成一个批次（Batch），调用**通义万相 2.x Pro**（用户指定 wanx 2.7 Pro，模型名做成配置项）文生图/参考图生图，按批输出一组图片，每张图片下展示对应关键词。
5. 用户确认/微调图片后，调用**可灵 Kling API** 图生视频，产出每个分镜的视频片段。
6. 故事板中按分镜顺序查看图片与视频，后续可拼接导出成片。

---

## 2. 总体架构

```
                ┌─────────────────────────────────────────────────────┐
                │                      用户（2C）                       │
                └───────────────┬─────────────────────────────────────┘
                                │ HTTPS / WebSocket
                ┌───────────────▼─────────────────┐
                │     前端 Web（React + Vite）      │
                │  剧本上传 / 模特管理 / 故事板 / 批次进度 │
                └───────────────┬─────────────────┘
                                │
                ┌───────────────▼─────────────────┐
                │       API 网关 / BFF（FastAPI）    │
                │  鉴权(JWT)、限流、配额、文件上传      │
                └──────┬──────────────┬───────────┘
                       │              │ 投递任务
              ┌────────▼───────┐  ┌───▼──────────────────────────┐
              │  业务服务层      │  │   任务队列（Redis + Celery/   │
              │ 项目/剧本/模特/  │  │   Arq；或 RabbitMQ）          │
              │ 分镜 CRUD       │  └───┬───────────┬─────────────┘
              └────────┬───────┘      │           │
                       │        ┌─────▼─────┐ ┌───▼─────────┐
              ┌────────▼──────┐ │ 剧本分析    │ │ 批量出图 Worker│
              │ PostgreSQL    │ │ Worker(LLM)│ │ (通义万相 Pro) │
              │ (业务数据)      │ └───────────┘ └───┬──────────┘
              └───────────────┘                    │
              ┌───────────────┐               ┌────▼──────────┐
              │ 对象存储 OSS/S3 │◄──────────────┤ 视频生成 Worker │
              │ (图片/视频/剧本) │               │ (可灵 Kling)   │
              └───────────────┘               └───────────────┘

              ┌─────────────────────────────────────────────┐
              │ 批次调度器（Scheduler，每 10 分钟触发一次出图批次）│
              └─────────────────────────────────────────────┘
```

### 分层说明

| 层 | 职责 | 技术选型 |
| --- | --- | --- |
| 前端 | 剧本上传、模特管理、故事板、批次进度实时展示 | React 18 + Vite（沿用现有） |
| API 网关/BFF | 鉴权、限流、用户配额、文件上传签名 | FastAPI（沿用现有） + Nginx |
| 业务服务 | 项目/剧本/模特/分镜/批次的 CRUD 与状态机 | FastAPI + SQLAlchemy |
| 异步任务 | 剧本分析、批量出图、图生视频，全部异步化 | Redis + Celery（或 Arq） |
| 调度器 | 每 10 分钟聚合待出图分镜为批次并入队 | Celery Beat / APScheduler |
| 存储 | 业务数据：PostgreSQL；媒体文件：OSS/S3 + CDN | PostgreSQL、阿里云 OSS |
| 外部模型 | LLM 剧本分析、通义万相出图、可灵图生视频 | DashScope API、Kling API |

### Provider 适配器模式（沿用现有设计）

`providers/base.py` 定义三个抽象接口，新增模型只需实现接口并在 `registry.py` 注册：

```python
class LLMProvider:        # 剧本分析（拆分镜 + 生成提示词/关键词）
    def analyze_script(script, characters) -> list[ShotDraft]: ...

class ImageProvider:      # 出图。新增 WanxProvider（通义万相 2.x Pro，DashScope）
    def generate(prompt, ref_images) -> bytes: ...
    def batch_generate(items: list[ImageTask]) -> list[ImageResult]: ...  # 新增批量接口

class VideoProvider:      # 图生视频。现有 KlingProvider
    def image_to_video(image, motion_prompt, duration) -> bytes | url: ...
```

新增 `providers/wanx_provider.py`：调用阿里云 DashScope 的万相文生图/图生图异步 API（`model` 配置为 `wan2.x-t2i-pro`，可随官方版本升级，如用户指定的 2.7 Pro），轮询任务结果后落盘到 OSS。

---

## 3. 核心数据模型

在现有 `Project / Character / Shot` 基础上扩展（迁移到 PostgreSQL）：

```
User           用户（2C 账号、套餐、配额）
Project        作品项目（属于某个 User）
Script         剧本（原文、解析后的场景结构、版本）
Character      模特主体（名字、描述、参考图集、可选 LoRA/形象 ID）
Shot           分镜（剧情概述、image_prompt、keywords[]、video_prompt、
               character_ids、image_url、video_url、status、batch_id）
GenerationBatch  出图批次（每 10 分钟一批）
  - id, project_id, window_start, window_end
  - shot_ids[]            # 本批包含的分镜
  - status: pending → running → partial_done → done / failed
  - provider, model       # wanx 2.x pro
VideoJob       可灵图生视频任务（shot_id、kling_task_id、status、回调结果）
```

关键点：

- **关键词（keywords）是 Shot 的一等字段**：LLM 分析剧本时同时产出 `image_prompt`（给模型）与 `keywords[]`（给用户看，展示在图片下方），二者同源但分开存储，用户可单独编辑关键词后重新出图。
- **Shot 状态机**：`pending → queued(进入批次) → image_running → image_done → video_running → done / failed`，失败单镜重试，不阻塞批次内其他分镜（沿用现有 MVP 思路）。

---

## 4. 关键流程设计

### 4.1 剧本上传与 AI 分析

```
用户上传剧本 ──► OSS 存原文 ──► 入队 analyze_script 任务
   Worker：LLM（结构化输出 JSON Schema）
     ├─ 切分场景/分镜（长剧本按场景分块、多轮调用、再合并）
     ├─ 每镜生成：summary / image_prompt / keywords[] / video_prompt
     ├─ 人物名 → 模特主体 ID 匹配
     └─ 写库，前端通过 WebSocket/轮询拿到分镜列表
```

- 长剧本（电影/电视剧集）**分块分析 + 全局上下文摘要**，避免超出 LLM 上下文。
- LLM 输出用 JSON Schema 校验，失败自动重试 + 降级提示。

### 4.2 每 10 分钟批量出图（核心调度）

```
Celery Beat 每 10 分钟触发 ──► 批次组装器
  1. 扫描所有 status=queued 的 Shot（按项目/用户分组）
  2. 组装 GenerationBatch（受配额与并发上限约束，超出部分顺延到下一批次）
  3. 入队 batch_generate 任务
Worker 执行批次：
  4. 并发调用通义万相 Pro（DashScope 异步任务 API，受 QPS 限流，
     带参考图时走图生图/人物一致性通道）
  5. 轮询任务结果 → 下载图片 → 存 OSS → 更新 Shot.image_url
  6. 批次完成后推送通知（WebSocket / 站内信）：
     前端按批展示"本批 N 张图"，每张图下方渲染 keywords[]
```

设计要点：

- **窗口聚合**：10 分钟是产品节奏（每 10 分钟输出一批图），实现上是定时任务聚合窗口内的待出图分镜，**不是**让用户干等——单镜也支持"立即出图"插队（高优队列）。
- **限流与退避**：DashScope/Kling 都有 QPS 与并发任务上限，Worker 内置令牌桶限流 + 指数退避重试（429/5xx）。
- **幂等**：批次与单镜任务都带幂等键（batch_id + shot_id），重试不重复扣费。

### 4.3 可灵图生视频

```
用户确认图片（或自动触发）──► 入队 video_job
  Worker：Kling JWT 鉴权 → 提交 image2video 任务 → 拿 task_id
        → 轮询/回调获取结果 → 视频存 OSS → 更新 Shot.video_url
```

- 可灵任务耗时长（分钟级），必须异步 + 轮询/回调，禁止同步阻塞 API。
- 运镜提示词来自 Shot.video_prompt，支持用户编辑后重跑。

---

## 5. 2C 化必须补齐的能力

| 能力 | 方案 |
| --- | --- |
| 账号体系 | 手机号/微信登录，JWT；User 表 + 套餐/配额 |
| 计费与配额 | 出图/视频按张/秒计量；批次组装时校验余额，余额不足的分镜挂起并提示 |
| 内容安全 | 剧本文本与生成图/视频接入审核（阿里云内容安全），违规拦截 |
| 多租户隔离 | 所有数据按 user_id 隔离；OSS 路径按用户分桶，签名 URL 访问 |
| 实时进度 | WebSocket 推送批次/分镜状态，降级轮询 |
| 可观测性 | 任务成功率、批次时延、模型 API 错误率监控告警（Prometheus + Grafana） |

---

## 6. 从当前 MVP 的演进路线

当前仓库已实现：FastAPI + React、JSON 文件存储、串行流水线（剧本分析 → 出图 → 图生视频）、Provider 适配器、Mock 降级。演进步骤：

1. **Phase 1（接入万相 + 关键词）**：新增 `WanxProvider`（通义万相 Pro）；`Shot` 增加 `keywords[]` 字段，LLM 分析时一并产出，前端图片卡片下方展示关键词。
2. **Phase 2（异步化 + 批次）**：引入 Redis + Celery，把出图/视频改为异步任务；新增 `GenerationBatch` 与 10 分钟调度器；JSON 存储迁移 PostgreSQL，媒体迁移 OSS。
3. **Phase 3（2C 上线）**：账号/配额/计费、内容审核、WebSocket 进度、监控告警、CDN。
4. **Phase 4（增强）**：分镜视频拼接成片、配音/字幕/BGM、人物一致性增强（多参考图 / 角色 LoRA / 万相人物一致性接口）、多模型路由（按成本/质量自动选模型）。

---

## 7. 配置示例

```env
# LLM 剧本分析
LLM_PROVIDER=openai            # 或 dashscope(qwen)
LLM_MODEL=gpt-4o-mini

# 通义万相出图（DashScope）
DASHSCOPE_API_KEY=sk-...
IMAGE_PROVIDER=wanx
WANX_MODEL=wan2.x-t2i-pro      # 按官方最新版本配置（如 2.x Pro）
BATCH_INTERVAL_MINUTES=10      # 批量出图窗口
BATCH_MAX_IMAGES=50            # 单批上限（受配额/QPS 约束）

# 可灵 Kling 图生视频
KLING_ACCESS_KEY=...
KLING_SECRET_KEY=...
KLING_MODEL=kling-v2-master
```

任一能力缺 key 自动降级 Mock（沿用现有机制），便于本地开发与演示。
