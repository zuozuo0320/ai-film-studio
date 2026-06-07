# 🎬 AI 影视创作平台（MVP）

上传**剧本** + 定义**人物** → LLM 自动拆解**分镜** → 每个分镜 **GPT 出图**（gpt-image-1，带人物参考保持一致性）→ **可灵 Kling 图生视频** → 在故事板里查看图片与视频。

> **零依赖即可体验**：不配置任何 API key 时，平台自动运行在 **Mock 模式**——用占位图 + ffmpeg 合成的短视频跑通整条流水线，方便先体验完整 UI 与流程，再接真实模型。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | FastAPI (Python 3.11+) |
| 前端 | React 18 + Vite |
| 存储 | 本地 JSON 文件 + 文件目录（`backend/data/`） |
| LLM（剧本拆解） | OpenAI（可配 base_url，兼容第三方网关） |
| 图片 | OpenAI **gpt-image-1**（文生图 / 参考图编辑保持人物一致性） |
| 视频 | **可灵 Kling** 图生视频（官方 JWT API，model 可配，支持 3.0） |

## 架构

```
backend/app/
  main.py            # FastAPI 路由
  config.py          # 配置 + Mock 自动降级
  models.py          # Project / Character / Shot 数据模型
  storage.py         # JSON 文件存储
  pipeline.py        # 剧本 -> 分镜 -> 出图 -> 图生视频 编排
  providers/
    base.py          # LLM / Image / Video 三个抽象接口
    registry.py      # 按配置选真实 provider 或 Mock
    openai_provider.py  # OpenAI LLM + gpt-image-1
    kling_provider.py   # 可灵 Kling 图生视频
    mock.py             # Mock（占位图 + ffmpeg 假视频）
frontend/src/
  App.jsx            # 主界面（项目/剧本/人物/故事板）
  components/        # CharacterPanel, ShotCard
  api.js             # 后端 API 封装
```

**Provider 适配器模式**：新增/升级模型（如换成可灵 3.0、Seedance、Runway，或用 DALL·E/FLUX 出图）只需实现 `base.py` 里对应接口并在 `registry.py` 注册，上层流水线无需改动。

## 快速开始

需要本机有 `ffmpeg`（Mock 视频用）。

### 后端

```bash
cd backend
cp .env.example .env          # 填 key 则用真实模型；留空则 Mock 模式
uv venv && source .venv/bin/activate
uv pip install fastapi "uvicorn[standard]" pydantic pydantic-settings \
  python-multipart httpx pyjwt pillow openai python-dotenv
uvicorn app.main:app --reload --port 8000
```

### 前端

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173 （已配置代理到后端 8000）
```

## 接入真实模型

编辑 `backend/.env`：

```env
# GPT 出图 + 剧本拆解
OPENAI_API_KEY=sk-...
LLM_MODEL=gpt-4o-mini
IMAGE_MODEL=gpt-image-1

# 可灵 Kling 图生视频
KLING_ACCESS_KEY=...
KLING_SECRET_KEY=...
KLING_MODEL=kling-v2-master      # 改成可灵 3.0 对应的官方 model_name
```

任一能力缺 key 时，该步骤自动降级到 Mock，其余仍走真实模型。

## 工作流

1. 新建项目，粘贴剧本（用空行分隔场景/镜头）。
2. 添加人物（名字 + 描述 + 可选参考图，参考图用于出图时保持人物一致性）。
3. 点「拆解剧本 → 生成分镜」，LLM 把剧本拆成分镜并自动匹配人物、生成出图/运镜提示词。
4. 单镜点「出图」「图生视频」，或点「⚡ 一键生成全部」自动跑完整条链路。

## 说明 / Roadmap

- 当前为 MVP：本地 JSON 存储、单用户、串行生成。
- 后续可扩展：数据库 + 多用户、分镜视频合成成片、配音/字幕/BGM、任务队列与并发、更强的人物一致性（多参考图 / 角色 LoRA）。
