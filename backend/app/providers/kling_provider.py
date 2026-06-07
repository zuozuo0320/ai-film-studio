"""可灵 Kling 图生视频 provider（官方开发者 API）。

认证：用 AccessKey/SecretKey 现签一个短期 JWT，作为 Bearer token。
流程（任务式）：POST 创建 image2video 任务 -> 轮询任务状态 -> 取回视频 URL -> 下载字节。

模型名通过配置 kling_model 指定（如可灵 3.0 对应的官方 model_name）。
"""
from __future__ import annotations

import base64
import time
from pathlib import Path

import httpx
import jwt

from ..config import Settings
from .base import VideoProvider


class KlingVideo(VideoProvider):
    name = "kling-video"

    def __init__(self, settings: Settings):
        self._ak = settings.kling_access_key
        self._sk = settings.kling_secret_key
        self._base = settings.kling_base_url.rstrip("/")
        self._model = settings.kling_model
        self._mode = settings.kling_mode

    def _token(self) -> str:
        now = int(time.time())
        payload = {"iss": self._ak, "exp": now + 1800, "nbf": now - 5}
        return jwt.encode(payload, self._sk, algorithm="HS256", headers={"alg": "HS256", "typ": "JWT"})

    def _headers(self) -> dict[str, str]:
        return {"Authorization": f"Bearer {self._token()}", "Content-Type": "application/json"}

    def image_to_video(self, image_path: Path, prompt: str, duration: int) -> bytes:
        image_b64 = base64.b64encode(image_path.read_bytes()).decode()
        payload = {
            "model_name": self._model,
            "mode": self._mode,
            "duration": str(duration),
            "image": image_b64,
            "prompt": prompt,
            "cfg_scale": 0.5,
        }
        with httpx.Client(timeout=60) as client:
            r = client.post(
                f"{self._base}/v1/videos/image2video", headers=self._headers(), json=payload
            )
            r.raise_for_status()
            task_id = r.json()["data"]["task_id"]
            video_url = self._poll(client, task_id)
            return client.get(video_url, timeout=120).content

    def _poll(self, client: httpx.Client, task_id: str, timeout: int = 600) -> str:
        deadline = time.time() + timeout
        while time.time() < deadline:
            r = client.get(
                f"{self._base}/v1/videos/image2video/{task_id}", headers=self._headers()
            )
            r.raise_for_status()
            data = r.json()["data"]
            status = data.get("task_status")
            if status == "succeed":
                return data["task_result"]["videos"][0]["url"]
            if status == "failed":
                raise RuntimeError(f"Kling 任务失败: {data.get('task_status_msg')}")
            time.sleep(5)
        raise TimeoutError("Kling 图生视频任务超时")
