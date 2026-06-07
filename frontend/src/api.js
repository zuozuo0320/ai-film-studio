// 后端 API 封装。开发时由 vite 代理到 127.0.0.1:8000。
const J = { "Content-Type": "application/json" };

async function req(url, opts) {
  const res = await fetch(url, opts);
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`${res.status} ${text}`);
  }
  return res.status === 204 ? null : res.json();
}

export const api = {
  config: () => req("/api/config"),
  listProjects: () => req("/api/projects"),
  createProject: (title, script) =>
    req("/api/projects", { method: "POST", headers: J, body: JSON.stringify({ title, script }) }),
  getProject: (id) => req(`/api/projects/${id}`),
  updateProject: (id, body) =>
    req(`/api/projects/${id}`, { method: "PATCH", headers: J, body: JSON.stringify(body) }),
  deleteProject: (id) => req(`/api/projects/${id}`, { method: "DELETE" }),

  addCharacter: (id, { name, description, file }) => {
    const fd = new FormData();
    fd.append("name", name);
    fd.append("description", description || "");
    if (file) fd.append("image", file);
    return req(`/api/projects/${id}/characters`, { method: "POST", body: fd });
  },
  deleteCharacter: (id, charId) =>
    req(`/api/projects/${id}/characters/${charId}`, { method: "DELETE" }),

  analyze: (id) => req(`/api/projects/${id}/analyze`, { method: "POST" }),
  updateShot: (id, shotId, body) =>
    req(`/api/projects/${id}/shots/${shotId}`, { method: "PATCH", headers: J, body: JSON.stringify(body) }),
  genImage: (id, shotId) => req(`/api/projects/${id}/shots/${shotId}/image`, { method: "POST" }),
  genVideo: (id, shotId) => req(`/api/projects/${id}/shots/${shotId}/video`, { method: "POST" }),
  generateAll: (id) => req(`/api/projects/${id}/generate-all`, { method: "POST" }),
};

export const mediaUrl = (path) => (path ? `/media/${path}` : null);
