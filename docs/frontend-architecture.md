# 前端架构设计（Web 端）

> 技术决策：Web 优先路线。前端选型 **React 18 + TypeScript + Vite**，与现有 MVP（React + Vite）平滑衔接，生态最成熟、招聘容易、未来可低成本包成 Tauri 桌面端或复用到小程序（Taro）。
>
> 与 [backend-architecture-go.md](./backend-architecture-go.md) 配套。

---

## 1. 技术栈

| 关注点 | 选型 | 理由 |
| --- | --- | --- |
| 框架 | React 18 + TypeScript | 沿用现有代码、生态/人才最优；TS 保证大型协作质量 |
| 构建 | Vite | 现有；快、配置少 |
| 路由 | React Router v6 | 标准方案 |
| 服务端状态 | TanStack Query | 缓存/重试/轮询/失效管理，天然适配"任务进度"场景 |
| 客户端状态 | Zustand | 轻量，避免 Redux 模板代码 |
| UI 组件 | Ant Design 5 | 中后台型重交互界面（故事板/表格/表单）开发效率最高，中文生态好 |
| 样式 | CSS Modules + AntD token 主题 | 简单可控 |
| 表单 | AntD Form | 与组件库一体 |
| 实时通信 | 原生 WebSocket + 自动降级轮询 | 对接 ws-gateway |
| 视频播放 | video.js | HLS/进度控制成熟 |
| 上传 | OSS 直传（STS 凭证）+ 分片断点续传 | 大剧本/参考图不经后端 |
| 国际化 | 暂不做，预留 i18n key 规范 | 2C 国内市场优先 |
| 测试 | Vitest + React Testing Library + Playwright(E2E 关键链路) | |
| 监控 | Sentry（错误）+ Web Vitals 上报 | 2C 必备 |

---

## 2. 工程结构（按 feature 切分）

```
frontend/src/
  app/                  # 应用骨架：路由、Provider、全局布局、错误边界
  shared/               # 跨功能复用
    api/                #   API client（openapi-typescript 从后端 OpenAPI 生成类型）
    ws/                 #   WebSocket 客户端（重连/心跳/降级轮询）
    ui/                 #   通用组件（上传器、视频播放器、空态、确认弹窗）
    hooks/  utils/
  features/             # 按业务功能切分，内部高内聚
    auth/               #   登录注册（手机号/微信扫码）
    project/            #   项目列表/创建/设置
    script/             #   剧本上传与编辑、AI 分析触发与结果
    character/          #   模特主体（角色资产包：多角度参考图/外貌/造型版本）
    storyboard/         #   故事板：分镜列表、候选图挑选、关键词展示/编辑
    generation/         #   批次进度中心：批次列表、实时进度、失败重试
    billing/            #   配额/套餐/订单
  stores/               # Zustand 全局 store（用户会话、全局通知）
```

原则：

- **类型从后端契约生成**：后端 OpenAPI → `openapi-typescript` 自动生成请求/响应类型，前后端契约不漂移。
- **feature 之间不互相 import 内部模块**，只通过 shared 或路由组合。

---

## 3. 关键页面与数据流

### 3.1 核心页面

```
登录 → 项目列表 → 项目工作台（核心，三栏布局）
                   ├─ 左：剧本区（上传/编辑/触发 AI 分析）
                   ├─ 中：故事板（分镜卡片流：候选图九宫格、图下关键词 chips、
                   │        状态徽标、出图/出视频操作、视频预览）
                   └─ 右：角色面板（角色资产包管理）
   全局：批次进度中心（顶栏入口，抽屉式：当前批次倒计时、进行中任务、完成通知）
```

### 3.2 任务进度数据流（核心模式）

```
用户触发生成 → POST 返回 task_id → TanStack Query 写入乐观状态(queued)
WebSocket 收到 progress.{user_id} 事件 → 按 task_id 失效对应 query 缓存
  → 分镜卡片自动刷新状态（queued → running → done，图片渐入）
WS 断开 → 自动指数退避重连；重连失败 → Query 切换为 5s 轮询模式
批次完成 → 全局 toast + 浏览器 Notification（用户授权后）"本批 N 张图已生成"
```

要点：**进度状态以服务端为唯一真相**，前端不自己推演状态机，只渲染推送/轮询结果，避免状态不一致。

### 3.3 候选图挑选（质量体系的前端落点）

- 每镜候选图按综合评分排序展示（美学/对齐度/一致性分以徽标呈现，见 quality-system.md）
- 关键词 chips 可直接编辑 → "用新关键词重新生成"
- 选定图后才解锁"生成视频"按钮（对应后端 image_review 关卡）

---

## 4. 性能（百万级用户的 2C 标准）

- **首屏**：路由级代码分割 + 登录页/工作台分包；目标 FCP < 1.5s（CDN + HTTP/2）
- **图片**：候选图用 OSS 图片处理出缩略图（列表缩略/点击大图），懒加载 + 占位骨架
- **长列表**：分镜数百个时虚拟滚动（`@tanstack/react-virtual`）
- **视频**：列表内只显示封面帧，点击才加载播放器；CDN 分发
- **缓存**：TanStack Query staleTime 分级（项目列表 30s、分镜实时失效）
- **构建**：vendor 分包、AntD 按需、bundle 监控（size-limit 进 CI）

---

## 5. 发布与质量保障

- 环境：dev / staging / prod 三套，静态资源推 OSS + CDN，HTML 不缓存实现秒级发布回滚
- 灰度：按用户百分比灰度新版本（网关层注入版本标）
- CI：lint(eslint+prettier) → typecheck → vitest → 构建 → Playwright 冒烟（登录/上传剧本/出图链路 mock 后端）
- 错误监控：Sentry sourcemap 上传，按 release 聚合；白屏率/接口错误率看板

---

## 6. 从现有 MVP 的演进

1. 现有 `frontend/`（JS）逐步 TS 化：先把 `api.js` 替换为 OpenAPI 生成的 typed client
2. 引入 React Router + TanStack Query，把 App.jsx 的单页状态拆到 features 结构
3. 接入 ws-gateway 实时进度，替换现有轮询
4. 按第 3 节落地三栏工作台与批次进度中心
