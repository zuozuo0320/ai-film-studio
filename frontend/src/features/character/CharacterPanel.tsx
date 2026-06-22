import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { App as AntdApp, Avatar, Button, Card, Empty, Form, Input, List, Modal, Typography } from "antd";
import { PlusOutlined, TeamOutlined, UserOutlined } from "@ant-design/icons";
import { api } from "../../shared/api/client";
import type { Project } from "../../shared/api/types";

// 角色面板：角色资产包（名字/外貌描述/参考图 URL）
export function CharacterPanel({ project }: { project: Project }) {
  const { message } = AntdApp.useApp();
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<{ name: string; description?: string; ref_images?: string }>();

  const addMut = useMutation({
    mutationFn: (v: { name: string; description?: string; ref_images?: string }) =>
      api.addCharacter(project.id, {
        name: v.name,
        description: v.description ?? "",
        ref_images: v.ref_images
          ? v.ref_images.split("\n").map((s) => s.trim()).filter(Boolean)
          : [],
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["project", project.id] });
      setOpen(false);
      form.resetFields();
    },
    onError: (e: Error) => message.error(e.message),
  });

  return (
    <Card
      className="studio-panel"
      title={
        <span className="panel-title">
          <TeamOutlined />
          角色
          <span className="panel-count">{project.characters.length}</span>
        </span>
      }
      size="small"
      extra={
        <Button icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          添加
        </Button>
      }
    >
      {project.characters.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="先添加角色，出图时保持人物一致性"
        >
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
            添加角色
          </Button>
        </Empty>
      ) : (
        <List
          className="character-list"
          itemLayout="horizontal"
          dataSource={project.characters}
          renderItem={(c) => (
            <List.Item>
              <List.Item.Meta
                avatar={
                  c.ref_images[0] ? (
                    <Avatar src={c.ref_images[0]} />
                  ) : (
                    <Avatar icon={<UserOutlined />} />
                  )
                }
                title={c.name}
                description={
                  <Typography.Paragraph className="character-description" ellipsis={{ rows: 2 }}>
                    {c.description || "（无描述）"}
                  </Typography.Paragraph>
                }
              />
            </List.Item>
          )}
        />
      )}

      <Modal
        title="添加角色"
        open={open}
        confirmLoading={addMut.isPending}
        onOk={() => form.validateFields().then((v) => addMut.mutate(v))}
        onCancel={() => setOpen(false)}
        okText="添加"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名字" rules={[{ required: true, message: "请输入名字" }]}>
            <Input placeholder="如：林夏" autoFocus />
          </Form.Item>
          <Form.Item name="description" label="外貌 / 设定描述">
            <Input.TextArea rows={3} placeholder="如：25 岁女性，齐肩短发，米色风衣…" />
          </Form.Item>
          <Form.Item name="ref_images" label="参考图 URL（每行一个，可选）">
            <Input.TextArea rows={2} placeholder="https://…/front.png" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
