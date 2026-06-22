import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Badge, Button, Layout, Result, Skeleton, Space, Typography } from "antd";
import { ArrowLeftOutlined, HistoryOutlined } from "@ant-design/icons";
import { Link, useParams } from "react-router-dom";
import { api } from "../../shared/api/client";
import { useProjectEvents } from "../../shared/ws/useProjectEvents";
import { ScriptPanel } from "../script/ScriptPanel";
import { CharacterPanel } from "../character/CharacterPanel";
import { Storyboard } from "../storyboard/Storyboard";
import { BatchDrawer } from "../generation/BatchDrawer";

// 项目工作台：三栏布局（左剧本 / 中故事板 / 右角色），顶栏批次进度入口
export function WorkbenchPage() {
  const { id } = useParams<{ id: string }>();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const { connected } = useProjectEvents(id);

  const hasRunning = (shots?: { status: string }[]) =>
    shots?.some((s) => ["queued", "image_running", "video_running"].includes(s.status)) ?? false;

  const { data: project, isLoading, error } = useQuery({
    queryKey: ["project", id],
    queryFn: () => api.getProject(id!),
    enabled: Boolean(id),
    staleTime: 0,
    // WS 断开时降级 5s 轮询；连接正常但有任务进行中时 15s 兜底轮询
    refetchInterval: (q) =>
      !connected ? 5000 : hasRunning(q.state.data?.shots) ? 15000 : false,
  });

  if (isLoading) {
    return (
      <Layout className="workbench-page">
        <Layout.Header className="workbench-header">
          <Skeleton.Button active style={{ width: 40 }} />
          <Skeleton.Input active style={{ width: 260 }} />
        </Layout.Header>
        <Layout.Content className="workbench-content">
          <div className="workbench-grid">
            <Skeleton active paragraph={{ rows: 8 }} />
            <Skeleton active paragraph={{ rows: 10 }} />
            <Skeleton active paragraph={{ rows: 8 }} />
          </div>
        </Layout.Content>
      </Layout>
    );
  }
  if (error || !project) {
    return (
      <Result
        status="404"
        title="项目不存在"
        extra={
          <Link to="/">
            <Button type="primary">返回项目列表</Button>
          </Link>
        }
      />
    );
  }
  const doneCount = project.shots.filter((s) => s.status === "done").length;
  const runningCount = project.shots.filter((s) =>
    ["queued", "image_running", "video_running"].includes(s.status),
  ).length;

  return (
    <Layout className="workbench-page">
      <Layout.Header className="workbench-header">
        <Link to="/">
          <Button
            aria-label="返回项目列表"
            type="text"
            icon={<ArrowLeftOutlined />}
            style={{ color: "#fff" }}
          />
        </Link>
        <div className="workbench-title-block">
          <div className="workbench-title-row">
            <Typography.Title level={5} className="workbench-title">
              {project.title}
            </Typography.Title>
          </div>
          <div className="workbench-subtitle">
            {project.shots.length} 个分镜 · {project.characters.length} 个角色 · {doneCount} 个已完成
          </div>
        </div>
        <div className="workbench-actions">
          <Badge
            status={connected ? "success" : "warning"}
            text={
              <span style={{ color: "rgba(255,255,255,.75)" }}>
                {connected ? "实时连接" : "轮询模式"}
              </span>
            }
          />
          <Space size={8} wrap>
            {runningCount > 0 && <Badge count={runningCount} style={{ backgroundColor: "#e5b547" }} />}
            <Button icon={<HistoryOutlined />} onClick={() => setDrawerOpen(true)}>
              批次进度
            </Button>
          </Space>
        </div>
      </Layout.Header>

      <Layout.Content className="workbench-content">
        <div className="workbench-grid">
          <ScriptPanel project={project} />
          <Storyboard project={project} />
          <CharacterPanel project={project} />
        </div>
      </Layout.Content>

      <BatchDrawer projectId={project.id} open={drawerOpen} onClose={() => setDrawerOpen(false)} />
    </Layout>
  );
}
