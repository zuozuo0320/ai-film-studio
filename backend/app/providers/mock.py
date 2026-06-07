"""Mock providers：无需任何 API key 即可跑通整条流水线。

- MockLLM：把剧本按段落/句子切成分镜，并用关键词粗略匹配人物。
- MockImage：用 Pillow 画一张带镜号/提示词文字的占位图。
- MockVideo：用 ffmpeg 把占位图做成一个带缓慢放大的小视频片段。
"""
from __future__ import annotations

import re
import subprocess
import tempfile
import textwrap
from io import BytesIO
from pathlib import Path

from PIL import Image, ImageDraw

from ..models import Character, ShotDraft
from .base import ImageProvider, LLMProvider, VideoProvider

_PALETTE = [
    (37, 99, 235), (219, 39, 119), (5, 150, 105), (217, 119, 6),
    (124, 58, 237), (8, 145, 178), (190, 18, 60), (15, 118, 110),
]


class MockLLM(LLMProvider):
    name = "mock-llm"

    def analyze_script(self, script: str, characters: list[Character]) -> list[ShotDraft]:
        # 优先按空行分段；否则按中英文句末标点切分
        blocks = [b.strip() for b in re.split(r"\n\s*\n", script.strip()) if b.strip()]
        if len(blocks) <= 1:
            blocks = [s.strip() for s in re.split(r"(?<=[。！？.!?])\s*", script.strip()) if s.strip()]
        if not blocks:
            blocks = ["空场景"]

        drafts: list[ShotDraft] = []
        for block in blocks:
            present = [c.name for c in characters if c.name and c.name in block]
            who = ("，".join(present) + " ") if present else ""
            drafts.append(
                ShotDraft(
                    summary=block[:60],
                    image_prompt=f"电影感分镜：{who}{block[:120]}，写实光影，16:9，高细节",
                    video_prompt="镜头缓慢推进，自然的人物微动作，电影级运镜",
                    character_names=present,
                )
            )
        return drafts


class MockImage(ImageProvider):
    name = "mock-image"

    def generate(self, prompt: str, reference_images: list[Path]) -> bytes:
        w, h = 1024, 576
        color = _PALETTE[abs(hash(prompt)) % len(_PALETTE)]
        img = Image.new("RGB", (w, h), color)
        draw = ImageDraw.Draw(img)
        # 渐变压暗底部，模拟电影画面
        for y in range(h):
            alpha = int(90 * (y / h))
            draw.line([(0, y), (w, y)], fill=(max(color[0] - alpha, 0),
                                              max(color[1] - alpha, 0),
                                              max(color[2] - alpha, 0)))
        draw.rectangle([20, 20, w - 20, h - 20], outline=(255, 255, 255), width=3)
        wrapped = textwrap.fill(prompt[:160], width=34)
        draw.multiline_text((48, 60), "MOCK 出图\n" + wrapped, fill=(255, 255, 255), spacing=8)
        if reference_images:
            draw.text((48, h - 60), f"参考图 x{len(reference_images)} (人物一致性)", fill=(255, 255, 0))
        buf = BytesIO()
        img.save(buf, format="PNG")
        return buf.getvalue()


class MockVideo(VideoProvider):
    name = "mock-video"

    def image_to_video(self, image_path: Path, prompt: str, duration: int) -> bytes:
        out = Path(tempfile.mkstemp(suffix=".mp4")[1])
        # 用 zoompan 做缓慢放大，模拟图生视频的动态效果
        d = max(int(duration), 2)
        frames = d * 25
        vf = (
            f"scale=1024:-2,zoompan=z='min(zoom+0.0010,1.15)':d={frames}"
            f":s=1024x576:fps=25,format=yuv420p"
        )
        cmd = [
            "ffmpeg", "-y", "-loop", "1", "-i", str(image_path),
            "-t", str(d), "-vf", vf, "-c:v", "libx264", "-pix_fmt", "yuv420p",
            str(out),
        ]
        subprocess.run(cmd, check=True, capture_output=True)
        data = out.read_bytes()
        out.unlink(missing_ok=True)
        return data
