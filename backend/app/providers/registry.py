"""根据配置选择真实 provider 还是 Mock。任一能力缺少 key 时自动降级到 Mock。"""
from __future__ import annotations

from ..config import Settings
from .base import ImageProvider, LLMProvider, VideoProvider
from .mock import MockImage, MockLLM, MockVideo


def get_llm(settings: Settings) -> LLMProvider:
    if settings.llm_enabled:
        from .openai_provider import OpenAILLM

        return OpenAILLM(settings)
    return MockLLM()


def get_image(settings: Settings) -> ImageProvider:
    if settings.image_enabled:
        from .openai_provider import OpenAIImage

        return OpenAIImage(settings)
    return MockImage()


def get_video(settings: Settings) -> VideoProvider:
    if settings.video_enabled:
        from .kling_provider import KlingVideo

        return KlingVideo(settings)
    return MockVideo()
