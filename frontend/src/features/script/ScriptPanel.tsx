import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { App as AntdApp, Button, Card, Input, Space, Typography, Upload } from "antd";
import { ThunderboltOutlined, UploadOutlined } from "@ant-design/icons";
import { api } from "../../shared/api/client";
import type { Project } from "../../shared/api/types";

// 剧本区：上传 .txt / 粘贴编辑 / 触发 AI 分析（异步，结果经 WS 推送刷新）
export function ScriptPanel({ project }: { project: Project }) {
  const { message } = AntdApp.useApp();
  const qc = useQueryClient();
  const [script, setScript] = useState(project.script);
  const [analyzing, setAnalyzing] = useState(false);

  useEffect(() => setScript(project.script), [project.id, project.script]);

  // 分析完成（shots 变化）后解除按钮 loading
  useEffect(() => {
    if (project.shots.length > 0) setAnalyzing(false);
  }, [project.shots.length]);

  const saveMut = useMutation({
    mutationFn: (s: string) => api.updateProject(project.id, { script: s }),
    onSuccess: (p) => {
      qc.setQueryData(["project", project.id], p);
      message.success("剧本已保存");
    },
    onError: (e: Error) => message.error(e.message),
  });

  const analyzeMut = useMutation({
    mutationFn: async () => {
      await api.updateProject(project.id, { script });
      return api.analyze(project.id);
    },
    onSuccess: () => {
      setAnalyzing(true);
      message.info("已提交 AI 分析，分镜生成后自动刷新");
    },
    onError: (e: Error) => message.error(e.message),
  });

  return (
    <Card
      title="剧本"
      size="small"
      extra={
        <Upload
          accept=".txt,.md"
          showUploadList={false}
          beforeUpload={(file) => {
            file.text().then(setScript);
            message.success(`已读入「${file.name}」，请点击保存`);
            return false;
          }}
        >
          <Button size="small" icon={<UploadOutlined />}>
            上传剧本
          </Button>
        </Upload>
      }
    >
      <Input.TextArea
        rows={14}
        value={script}
        onChange={(e) => setScript(e.target.value)}
        placeholder="在此粘贴/编写剧本，用空行分隔不同场景或镜头…"
      />
      <Space style={{ marginTop: 12 }}>
        <Button loading={saveMut.isPending} onClick={() => saveMut.mutate(script)}>
          保存剧本
        </Button>
        <Button
          type="primary"
          icon={<ThunderboltOutlined />}
          disabled={!script.trim()}
          loading={analyzeMut.isPending || analyzing}
          onClick={() => analyzeMut.mutate()}
        >
          AI 分析 → 生成分镜
        </Button>
      </Space>
      {project.shots.length > 0 && (
        <Typography.Paragraph type="secondary" style={{ marginTop: 8, marginBottom: 0 }}>
          重新分析会覆盖现有 {project.shots.length} 个分镜。
        </Typography.Paragraph>
      )}
    </Card>
  );
}
