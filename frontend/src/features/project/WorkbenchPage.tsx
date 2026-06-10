import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Badge, Button, Layout, Result, Spin, Typography } from "antd";
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
    return <Spin size="large" style={{ display: "block", margin: "120px auto" }} />;
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

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Layout.Header style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <Link to="/">
          <Button type="text" icon={<ArrowLeftOutlined />} style={{ color: "#fff" }} />
        </Link>
        <Typography.Title level={5} style={{ color: "#fff", margin: 0, flex: 1 }}>
          {project.title}
        </Typography.Title>
        <Badge
          status={connected ? "success" : "warning"}
          text={
            <span style={{ color: "rgba(255,255,255,.75)" }}>
              {connected ? "实时连接" : "轮询模式"}
            </span>
          }
        />
        <Button icon={<HistoryOutlined />} onClick={() => setDrawerOpen(true)}>
          批次进度
        </Button>
      </Layout.Header>

      <Layout.Content>
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
