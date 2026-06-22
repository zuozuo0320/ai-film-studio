import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  App as AntdApp,
  Button,
  Card,
  Empty,
  Form,
  Input,
  Layout,
  Modal,
  Popconfirm,
  Space,
  Tag,
  Typography,
} from "antd";
import {
  DeleteOutlined,
  PlusOutlined,
  ProjectOutlined,
  VideoCameraOutlined,
} from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { api } from "../../shared/api/client";

export function ProjectListPage() {
  const { message } = AntdApp.useApp();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<{ title: string; script?: string }>();

  const { data: projects, isLoading } = useQuery({
    queryKey: ["projects"],
    queryFn: api.listProjects,
  });
  const projectList = projects ?? [];
  const totalShots = projectList.reduce((sum, p) => sum + p.shots.length, 0);
  const totalCharacters = projectList.reduce((sum, p) => sum + p.characters.length, 0);

  const createMut = useMutation({
    mutationFn: (v: { title: string; script?: string }) => api.createProject(v.title, v.script ?? ""),
    onSuccess: (p) => {
      qc.invalidateQueries({ queryKey: ["projects"] });
      setOpen(false);
      form.resetFields();
      navigate(`/projects/${p.id}`);
    },
    onError: (e: Error) => message.error(e.message),
  });

  const deleteMut = useMutation({
    mutationFn: api.deleteProject,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects"] }),
    onError: (e: Error) => message.error(e.message),
  });

  return (
    <Layout className="studio-shell">
      <Layout.Header className="studio-header">
        <div className="studio-brand">
          <span className="studio-logo">
            <VideoCameraOutlined style={{ fontSize: 21 }} />
          </span>
          <div>
            <Typography.Title level={4} className="studio-title">
              AI 影视创作平台
            </Typography.Title>
            <span className="studio-kicker">剧本拆解 · 角色一致性 · 分镜出图 · 图生视频</span>
          </div>
        </div>
        <div className="studio-header-actions">
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
            新建项目
          </Button>
        </div>
      </Layout.Header>
      <Layout.Content className="studio-page">
        <section className="project-hero">
          <div>
            <Typography.Title level={1}>把剧本推进成可审片的分镜资产</Typography.Title>
            <Typography.Paragraph>
              从项目入口管理剧本、角色、图片和视频任务。这里保持轻量但信息完整，方便你快速进入正在制作的片段。
            </Typography.Paragraph>
          </div>
          <Button size="large" type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
            新建项目
          </Button>
        </section>

        <section className="project-metrics" aria-label="项目概览">
          <div className="metric-tile">
            <span className="metric-value">{projectList.length}</span>
            <span className="metric-label">项目</span>
          </div>
          <div className="metric-tile">
            <span className="metric-value">{totalShots}</span>
            <span className="metric-label">分镜</span>
          </div>
          <div className="metric-tile">
            <span className="metric-value">{totalCharacters}</span>
            <span className="metric-label">角色资产</span>
          </div>
        </section>

        {isLoading ? (
          <div className="project-grid">
            {[0, 1, 2].map((i) => (
              <Card key={i} loading className="project-card" />
            ))}
          </div>
        ) : projectList.length === 0 ? (
          <div className="studio-empty">
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="还没有项目。先创建一个项目，把剧本放进工作台。"
            >
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
                新建第一个项目
              </Button>
            </Empty>
          </div>
        ) : (
          <div className="project-grid">
            {projectList.map((p) => (
              <Card
                key={p.id}
                hoverable
                className="project-card"
                title={p.title}
                onClick={() => navigate(`/projects/${p.id}`)}
                extra={
                  <Popconfirm
                    title="确认删除该项目？"
                    onConfirm={(e) => {
                      e?.stopPropagation();
                      deleteMut.mutate(p.id);
                    }}
                    onCancel={(e) => e?.stopPropagation()}
                  >
                    <Button
                      aria-label={`删除项目 ${p.title}`}
                      icon={<DeleteOutlined />}
                      size="small"
                      danger
                      type="text"
                      onClick={(e) => e.stopPropagation()}
                    >
                      删除
                    </Button>
                  </Popconfirm>
                }
              >
                <Typography.Paragraph className="project-card-description">
                  {p.script ? `${p.script.slice(0, 72)}${p.script.length > 72 ? "..." : ""}` : "还没有剧本内容。"}
                </Typography.Paragraph>
                <div className="project-card-meta">
                  <Tag icon={<ProjectOutlined />} color="default">
                    {p.shots.length} 个分镜
                  </Tag>
                  <Tag color="cyan">{p.characters.length} 个角色</Tag>
                  {p.shots.some((s) => s.status === "done") && <Tag color="success">有成片</Tag>}
                </div>
              </Card>
            ))}
          </div>
        )}
      </Layout.Content>

      <Modal
        title="新建项目"
        open={open}
        confirmLoading={createMut.isPending}
        onOk={() => form.validateFields().then((v) => createMut.mutate(v))}
        onCancel={() => setOpen(false)}
        okText="创建"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item name="title" label="项目标题" rules={[{ required: true, message: "请输入标题" }]}>
            <Input placeholder="如：咖啡馆的重逢" autoFocus />
          </Form.Item>
          <Form.Item name="script" label="剧本（可选，也可稍后在工作台上传/粘贴）">
            <Input.TextArea rows={6} placeholder="在此粘贴剧本，用空行分隔不同场景或镜头…" />
          </Form.Item>
          <Space direction="vertical" size={4}>
            <Typography.Text type="secondary">
              创建后会进入工作台，你可以继续添加角色参考并拆解分镜。
            </Typography.Text>
          </Space>
        </Form>
      </Modal>
    </Layout>
  );
}
