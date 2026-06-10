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
  List,
  Modal,
  Popconfirm,
  Typography,
} from "antd";
import { PlusOutlined, VideoCameraOutlined } from "@ant-design/icons";
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
    <Layout style={{ minHeight: "100vh" }}>
      <Layout.Header style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <VideoCameraOutlined style={{ color: "#fff", fontSize: 20 }} />
        <Typography.Title level={4} style={{ color: "#fff", margin: 0, flex: 1 }}>
          AI 影视创作平台
        </Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          新建项目
        </Button>
      </Layout.Header>
      <Layout.Content style={{ padding: 24, maxWidth: 960, width: "100%", margin: "0 auto" }}>
        <List
          loading={isLoading}
          grid={{ gutter: 16, column: 3 }}
          dataSource={projects ?? []}
          locale={{ emptyText: <Empty description="还没有项目，点右上角新建" /> }}
          renderItem={(p) => (
            <List.Item>
              <Card
                hoverable
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
                    <Button size="small" danger type="text" onClick={(e) => e.stopPropagation()}>
                      删除
                    </Button>
                  </Popconfirm>
                }
              >
                {p.shots.length} 个分镜 · {p.characters.length} 个角色
              </Card>
            </List.Item>
          )}
        />
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
        </Form>
      </Modal>
    </Layout>
  );
}
