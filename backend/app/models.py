from __future__ import annotations

import time
import uuid
from enum import Enum

from pydantic import BaseModel, Field


def _id(prefix: str) -> str:
    return f"{prefix}_{uuid.uuid4().hex[:12]}"


def _now() -> float:
    return time.time()


class ShotStatus(str, Enum):
    pending = "pending"
    image_running = "image_running"
    image_done = "image_done"
    video_running = "video_running"
    done = "done"
    failed = "failed"


class Character(BaseModel):
    id: str = Field(default_factory=lambda: _id("char"))
    name: str
    description: str = ""
    # 参考图相对路径（相对 data/），用于出图时保持人物一致性
    ref_image: str | None = None


class Shot(BaseModel):
    id: str = Field(default_factory=lambda: _id("shot"))
    index: int
    summary: str = ""              # 这一镜的剧情概述
    image_prompt: str = ""         # 出图用提示词
    video_prompt: str = ""         # 图生视频用的运动/镜头提示词
    character_ids: list[str] = Field(default_factory=list)
    image_path: str | None = None  # 相对 data/ 的图片路径
    video_path: str | None = None  # 相对 data/ 的视频路径，或外部 url
    status: ShotStatus = ShotStatus.pending
    error: str | None = None


class Project(BaseModel):
    id: str = Field(default_factory=lambda: _id("proj"))
    title: str
    script: str = ""
    characters: list[Character] = Field(default_factory=list)
    shots: list[Shot] = Field(default_factory=list)
    created_at: float = Field(default_factory=_now)
    updated_at: float = Field(default_factory=_now)


# ---------- 请求 / 响应 ----------

class ProjectCreate(BaseModel):
    title: str
    script: str = ""


class ProjectUpdate(BaseModel):
    title: str | None = None
    script: str | None = None


class ShotUpdate(BaseModel):
    summary: str | None = None
    image_prompt: str | None = None
    video_prompt: str | None = None
    character_ids: list[str] | None = None


class ShotDraft(BaseModel):
    """LLM 拆解剧本后产出的单镜草稿。"""

    summary: str = ""
    image_prompt: str = ""
    video_prompt: str = ""
    character_names: list[str] = Field(default_factory=list)
