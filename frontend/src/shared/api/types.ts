// 与 backend-go/internal/models/models.go 对应的契约类型。
// TODO: 后端提供 OpenAPI 后改用 openapi-typescript 自动生成。

export type ShotStatus =
  | "pending"
  | "queued"
  | "image_running"
  | "image_review"
  | "video_running"
  | "done"
  | "failed";

export type BatchStatus = "pending" | "running" | "done" | "failed";

export interface Character {
  id: string;
  name: string;
  description: string;
  ref_images: string[];
}

export interface Shot {
  id: string;
  index: number;
  summary: string;
  image_prompt: string;
  keywords: string[];
  video_prompt: string;
  character_ids: string[];
  image_path?: string;
  video_path?: string;
  batch_id?: string;
  status: ShotStatus;
  error?: string;
}

export interface Project {
  id: string;
  title: string;
  script: string;
  characters: Character[];
  shots: Shot[];
  created_at: string;
  updated_at: string;
}

export interface GenerationBatch {
  id: string;
  project_id: string;
  shot_ids: string[];
  status: BatchStatus;
  model: string;
  window_start: string;
  created_at: string;
  finished_at?: string;
}

export interface TaskAccepted {
  task_id: string;
}

// WebSocket 进度事件（backend-go/internal/ws/hub.go Event）
export interface ProgressEvent {
  type: "shot_update" | "batch_update" | "project_analyzed";
  project_id: string;
  shot_id?: string;
  batch_id?: string;
  status: string;
  error?: string;
}
