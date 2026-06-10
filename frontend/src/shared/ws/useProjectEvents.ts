// WebSocket 进度订阅：收到事件即失效对应 query，断线指数退避重连，
// 重连失败期间降级为 5s 轮询（进度状态以服务端为唯一真相）。
import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { ProgressEvent } from "../api/types";

const MAX_BACKOFF_MS = 30_000;

export function useProjectEvents(projectId: string | undefined) {
  const qc = useQueryClient();
  const [connected, setConnected] = useState(false);
  const attemptRef = useRef(0);

  useEffect(() => {
    if (!projectId) return;
    let ws: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let closed = false;

    const connect = () => {
      const proto = location.protocol === "https:" ? "wss" : "ws";
      ws = new WebSocket(`${proto}://${location.host}/ws?project_id=${projectId}`);
      ws.onopen = () => {
        attemptRef.current = 0;
        setConnected(true);
      };
      ws.onmessage = (m) => {
        let e: ProgressEvent;
        try {
          e = JSON.parse(m.data);
        } catch {
          return;
        }
        qc.invalidateQueries({ queryKey: ["project", e.project_id] });
        if (e.type === "batch_update") {
          qc.invalidateQueries({ queryKey: ["batches", e.project_id] });
        }
      };
      ws.onclose = () => {
        setConnected(false);
        if (closed) return;
        const backoff = Math.min(1000 * 2 ** attemptRef.current, MAX_BACKOFF_MS);
        attemptRef.current += 1;
        timer = setTimeout(connect, backoff);
      };
      ws.onerror = () => ws?.close();
    };
    connect();

    return () => {
      closed = true;
      if (timer) clearTimeout(timer);
      ws?.close();
    };
  }, [projectId, qc]);

  return { connected };
}
