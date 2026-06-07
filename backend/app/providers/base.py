"""Provider 适配器接口。

三类能力各定义一个抽象接口，新增/升级模型只需实现对应接口并在 registry 注册：
- LLMProvider：剧本 -> 分镜草稿
- ImageProvider：提示词(+参考图) -> 图片字节
- VideoProvider：图片(+提示词) -> 视频字节
"""
from __future__ import annotations

from abc import ABC, abstractmethod
from pathlib import Path

from ..models import Character, ShotDraft


class LLMProvider(ABC):
    name: str = "base-llm"

    @abstractmethod
    def analyze_script(self, script: str, characters: list[Character]) -> list[ShotDraft]:
        """把剧本拆解为有序的分镜草稿列表。"""
        raise NotImplementedError


class ImageProvider(ABC):
    name: str = "base-image"

    @abstractmethod
    def generate(self, prompt: str, reference_images: list[Path]) -> bytes:
        """根据提示词出图。若提供参考图（人物形象），应尽量保持一致性。

        返回 PNG 字节。
        """
        raise NotImplementedError


class VideoProvider(ABC):
    name: str = "base-video"

    @abstractmethod
    def image_to_video(self, image_path: Path, prompt: str, duration: int) -> bytes:
        """图生视频。返回 mp4 字节。"""
        raise NotImplementedError
