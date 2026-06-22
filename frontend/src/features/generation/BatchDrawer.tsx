import { useQuery } from "@tanstack/react-query";
import { ClockCircleOutlined } from "@ant-design/icons";
import { Drawer, Empty, List, Space, Tag, Typography } from "antd";
import { api } from "../../shared/api/client";
import type { BatchStatus } from "../../shared/api/types";

const BATCH_COLOR: Record<BatchStatus, string> = {
  pending: "default",
  running: "processing",
  done: "success",
  failed: "error",
};

// 批次进度中心：每个调度窗口（默认 10 分钟）聚合的出图批次
export function BatchDrawer({
  projectId,
  open,
  onClose,
}: {
  projectId: string;
  open: boolean;
  onClose: () => void;
}) {
  const { data: batches } = useQuery({
    queryKey: ["batches", projectId],
    queryFn: () => api.listBatches(projectId),
    enabled: open,
    refetchInterval: open ? 5000 : false,
  });

  return (
    <Drawer title="批次进度中心" open={open} onClose={onClose} width={460}>
      <Typography.Paragraph type="secondary">
        入队的分镜会在下个批次窗口（默认 10 分钟）统一出图。
      </Typography.Paragraph>
      {!batches?.length ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无出图批次" />
      ) : (
        <List
          className="batch-list"
          dataSource={batches}
          renderItem={(b) => (
            <List.Item>
              <List.Item.Meta
                title={
                  <Space size={8} wrap>
                    <Tag color={BATCH_COLOR[b.status]}>{b.status}</Tag>
                    <Typography.Text strong>{b.shot_ids.length} 个分镜</Typography.Text>
                    <Typography.Text type="secondary">{b.model}</Typography.Text>
                  </Space>
                }
                description={
                  <Space direction="vertical" size={2}>
                    <Typography.Text type="secondary">
                      <ClockCircleOutlined /> 创建：{new Date(b.created_at).toLocaleString()}
                    </Typography.Text>
                    {b.finished_at && (
                      <Typography.Text type="secondary">
                        完成：{new Date(b.finished_at).toLocaleString()}
                      </Typography.Text>
                    )}
                  </Space>
                }
              />
            </List.Item>
          )}
        />
      )}
    </Drawer>
  );
}
