"""OpenAI providers：GPT 文本拆解剧本 + gpt-image-1 出图。

出图策略（人物一致性）：
- 若该镜关联了人物且人物有参考图，用 images.edit + input_fidelity=high，把参考图作为
  输入，让 gpt-image-1 在保持人物形象的前提下生成新画面。
- 否则用 images.generate 纯文生图。
"""
from __future__ import annotations

import base64
import json
from pathlib import Path

from openai import OpenAI

from ..config import Settings
from ..models import Character, ShotDraft
from .base import ImageProvider, LLMProvider

_SYSTEM = (
    "你是专业的影视分镜师。把用户的剧本拆解成有序的分镜(shot)列表。"
    "每个分镜输出：summary(该镜剧情, 简短中文)、image_prompt(出图提示词, 含人物/场景/"
    "构图/光影/景别, 适合文生图)、video_prompt(图生视频的运镜与动作提示)、"
    "character_names(本镜出现的人物名, 必须从给定人物列表里选)。"
    '只输出 JSON: {"shots":[{"summary":"","image_prompt":"","video_prompt":"",'
    '"character_names":[]}]}'
)


class OpenAILLM(LLMProvider):
    name = "openai-llm"

    def __init__(self, settings: Settings):
        self._client = OpenAI(api_key=settings.openai_api_key, base_url=settings.openai_base_url)
        self._model = settings.llm_model

    def analyze_script(self, script: str, characters: list[Character]) -> list[ShotDraft]:
        char_block = "\n".join(f"- {c.name}: {c.description}" for c in characters) or "（无指定人物）"
        user = f"人物列表:\n{char_block}\n\n剧本:\n{script}"
        resp = self._client.chat.completions.create(
            model=self._model,
            response_format={"type": "json_object"},
            messages=[
                {"role": "system", "content": _SYSTEM},
                {"role": "user", "content": user},
            ],
        )
        content = resp.choices[0].message.content or "{}"
        data = json.loads(content)
        drafts: list[ShotDraft] = []
        for item in data.get("shots", []):
            drafts.append(
                ShotDraft(
                    summary=str(item.get("summary", ""))[:120],
                    image_prompt=str(item.get("image_prompt", "")),
                    video_prompt=str(item.get("video_prompt", "")),
                    character_names=[str(n) for n in item.get("character_names", [])],
                )
            )
        return drafts or [ShotDraft(summary=script[:60], image_prompt=script[:200])]


class OpenAIImage(ImageProvider):
    name = "openai-image"

    def __init__(self, settings: Settings):
        self._client = OpenAI(api_key=settings.openai_api_key, base_url=settings.openai_base_url)
        self._model = settings.image_model
        self._size = settings.image_size

    def generate(self, prompt: str, reference_images: list[Path]) -> bytes:
        if reference_images:
            files = [p.open("rb") for p in reference_images]
            try:
                resp = self._client.images.edit(
                    model=self._model,
                    image=files,
                    prompt=prompt,
                    size=self._size,
                    input_fidelity="high",
                )
            finally:
                for f in files:
                    f.close()
        else:
            resp = self._client.images.generate(
                model=self._model, prompt=prompt, size=self._size,
            )
        b64 = resp.data[0].b64_json
        return base64.b64decode(b64)
