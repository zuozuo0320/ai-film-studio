import { useMutation, useQueryClient } from "@tanstack/react-query";
import { App as AntdApp, Badge, Button, Card, Space, Tag, Tooltip, Typography } from "antd";
import {
  CheckOutlined,
  LoadingOutlined,
  PictureOutlined,
  PlayCircleOutlined,
} from "@ant-design/icons";
import { api, mediaUrl } from "../../shared/api/client";
import type { Character, Project, Shot, ShotStatus } from "../../shared/api/types";

const STATUS_META: Record<ShotStatus, { text: string; color: string; tagColor: string }> = {
  pending: { text: "待出图", color: "default", tagColor: "default" },
  queued: { text: "已入队", color: "blue", tagColor: "blue" },
  image_running: { text: "出图中", color: "processing", tagColor: "processing" },
  image_review: { text: "待确认", color: "gold", tagColor: "gold" },
  video_running: { text: "视频中", color: "processing", tagColor: "processing" },
  done: { text: "完成", color: "success", tagColor: "success" },
  failed: { text: "失败", color: "error", tagColor: "error" },
};

export function ShotCard({ project, shot }: { project: Project; shot: Shot }) {
  const { message } = AntdApp.useApp();
  const qc = useQueryClient();
  const meta = STATUS_META[shot.status];
  const invalidate = () => qc.invalidateQueries({ queryKey: ["project", project.id] });

  const enqueueMut = useMutation({
    mutationFn: () => api.enqueueImage(project.id, shot.id),
    onSuccess: () => {
      invalidate();
      message.success("已加入下个出图批次");
    },
    onError: (e: Error) => message.error(e.message),
  });

  const approveMut = useMutation({
    mutationFn: async () => {
      await api.approveShot(project.id, shot.id);
      return api.generateVideo(project.id, shot.id);
    },
    onSuccess: () => {
      invalidate();
      message.success("图片已确认，开始生成视频");
    },
    onError: (e: Error) => message.error(e.message),
  });

  const chars = shot.character_ids
    .map((id) => project.characters.find((c) => c.id === id))
    .filter((c): c is Character => Boolean(c));

  const running = shot.status === "image_running" || shot.status === "video_running";

  return (
    <Badge.Ribbon text={meta.text} color={meta.color}>
      <Card
        className="shot-card"
        size="small"
        title={
          <span className="shot-card-title">
            <span className="shot-index">#{shot.index + 1}</span>
            <span className="shot-summary">{shot.summary}</span>
          </span>
        }
      >
        <div className="shot-image-wrap">
          {shot.video_path ? (
            <video src={mediaUrl(shot.video_path)} controls preload="metadata" />
          ) : shot.image_path ? (
            <img src={mediaUrl(shot.image_path)} alt={shot.summary} loading="lazy" />
          ) : running ? (
            <span className="shot-placeholder">
              <LoadingOutlined style={{ fontSize: 28 }} />
              {shot.status === "video_running" ? "视频生成中" : "图片生成中"}
            </span>
          ) : (
            <span className="shot-placeholder">
              <PictureOutlined style={{ fontSize: 28 }} />
              等待生成画面
            </span>
          )}
        </div>

        {/* 关键词 chips：图片下方展示 */}
        <Space className="shot-tags" size={[4, 4]} wrap>
          <Tag color={meta.tagColor}>{meta.text}</Tag>
          {shot.keywords.map((k) => (
            <Tag key={k} color="geekblue">
              {k}
            </Tag>
          ))}
          {chars.map((c) => (
            <Tag key={c.id} color="cyan">
              @{c.name}
            </Tag>
          ))}
        </Space>

        {shot.error && (
          <Typography.Paragraph className="shot-error" type="danger">
            {shot.error}
          </Typography.Paragraph>
        )}

        <div className="shot-actions">
          <Tooltip title="加入批次队列，等下个批次窗口统一出图">
            <Button
              icon={<PictureOutlined />}
              loading={enqueueMut.isPending}
              disabled={shot.status === "queued" || running}
              onClick={() => enqueueMut.mutate()}
            >
              {shot.image_path ? "重新出图" : "出图"}
            </Button>
          </Tooltip>
          <Tooltip title="确认图片（质量关卡）后生成视频">
            <Button
              type="primary"
              icon={shot.status === "image_review" ? <CheckOutlined /> : <PlayCircleOutlined />}
              loading={approveMut.isPending || shot.status === "video_running"}
              disabled={!shot.image_path || running || shot.status === "queued"}
              onClick={() => approveMut.mutate()}
            >
              {shot.status === "done" ? "重新生成视频" : "确认图片 → 生成视频"}
            </Button>
          </Tooltip>
        </div>
      </Card>
    </Badge.Ribbon>
  );
}
