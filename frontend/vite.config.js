import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// 开发时把 /api 和 /media 代理到后端，避免 CORS 与硬编码地址
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:8000",
      "/media": "http://127.0.0.1:8000",
    },
  },
});
