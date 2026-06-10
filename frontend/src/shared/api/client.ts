// Go 后端 API client（/api/v1）。开发时由 vite 代理到 127.0.0.1:8080。
import type { Character, GenerationBatch, Project, Shot, TaskAccepted } from "./types";

const J = { "Content-Type": "application/json" };

async function req<T>(url: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(url, opts);
  if (!res.ok) {
    let msg = `${res.status}`;
    try {
      const body = await res.json();
      if (body?.error) msg = body.error;
    } catch {
      /* 非 JSON 错误体 */
    }
    throw new Error(msg);
  }
  return res.status === 204 ? (null as T) : ((await res.json()) as T);
}

export const api = {
  listProjects: () => req<Project[]>("/api/v1/projects"),
  createProject: (title: string, script = "") =>
    req<Project>("/api/v1/projects", {
      method: "POST",
      headers: J,
      body: JSON.stringify({ title, script }),
    }),
  getProject: (id: string) => req<Project>(`/api/v1/projects/${id}`),
  updateProject: (id: string, body: { title?: string; script?: string }) =>
    req<Project>(`/api/v1/projects/${id}`, {
      method: "PATCH",
      headers: J,
      body: JSON.stringify(body),
    }),
  deleteProject: (id: string) => req<null>(`/api/v1/projects/${id}`, { method: "DELETE" }),

  addCharacter: (id: string, body: { name: string; description?: string; ref_images?: string[] }) =>
    req<Character>(`/api/v1/projects/${id}/characters`, {
      method: "POST",
      headers: J,
      body: JSON.stringify(body),
    }),

  analyze: (id: string) => req<TaskAccepted>(`/api/v1/projects/${id}/analyze`, { method: "POST" }),

  enqueueImage: (id: string, shotID: string) =>
    req<Shot>(`/api/v1/projects/${id}/shots/${shotID}/enqueue-image`, { method: "POST" }),
  approveShot: (id: string, shotID: string) =>
    req<Shot>(`/api/v1/projects/${id}/shots/${shotID}/approve`, { method: "POST" }),
  generateVideo: (id: string, shotID: string) =>
    req<TaskAccepted>(`/api/v1/projects/${id}/shots/${shotID}/video`, { method: "POST" }),

  listBatches: (id: string) => req<GenerationBatch[]>(`/api/v1/projects/${id}/batches`),
  assembleBatchNow: () => req<{ ok: boolean }>("/api/v1/debug/assemble-batch", { method: "POST" }),
};

export const mediaUrl = (path?: string) => (path ? `/media/${path}` : undefined);
