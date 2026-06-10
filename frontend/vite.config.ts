import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// 开发时 /api /media /ws 代理到 Go 后端（:8080），避免 CORS 与硬编码地址
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:8080",
      "/media": "http://127.0.0.1:8080",
      "/ws": { target: "ws://127.0.0.1:8080", ws: true },
    },
  },
});
