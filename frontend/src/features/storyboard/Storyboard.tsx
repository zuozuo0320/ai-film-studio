import { useMutation, useQueryClient } from "@tanstack/react-query";
import { App as AntdApp, Button, Card, Empty, Space, Typography } from "antd";
import { PictureOutlined } from "@ant-design/icons";
import { api } from "../../shared/api/client";
import type { Project } from "../../shared/api/types";
import { ShotCard } from "./ShotCard";

export function Storyboard({ project }: { project: Project }) {
  const { message } = AntdApp.useApp();
  const qc = useQueryClient();
  const doneCount = project.shots.filter((s) => s.status === "done").length;

  const enqueueAllMut = useMutation({
    mutationFn: async () => {
      const targets = project.shots.filter(
        (s) => s.status === "pending" || s.status === "failed",
      );
      for (const s of targets) {
        await api.enqueueImage(project.id, s.id);
      }
      return targets.length;
    },
    onSuccess: (n) => {
      qc.invalidateQueries({ queryKey: ["project", project.id] });
      message.success(`已把 ${n} 个分镜加入下个出图批次`);
    },
    onError: (e: Error) => message.error(e.message),
  });

  return (
    <Card
      size="small"
      title={`分镜故事板（${doneCount}/${project.shots.length} 完成）`}
      extra={
        project.shots.length > 0 && (
          <Space>
            <Button
              size="small"
              icon={<PictureOutlined />}
              loading={enqueueAllMut.isPending}
              onClick={() => enqueueAllMut.mutate()}
            >
              全部入队出图
            </Button>
          </Space>
        )
      }
    >
      {project.shots.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <Typography.Text type="secondary">
              先在左侧上传/粘贴剧本并触发 AI 分析，自动拆解分镜与关键词
            </Typography.Text>
          }
        />
      ) : (
        <div className="shot-grid">
          {project.shots.map((s) => (
            <ShotCard key={s.id} project={project} shot={s} />
          ))}
        </div>
      )}
    </Card>
  );
}
