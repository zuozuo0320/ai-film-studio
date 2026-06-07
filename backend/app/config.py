from functools import lru_cache
from pathlib import Path

from pydantic_settings import BaseSettings, SettingsConfigDict

BASE_DIR = Path(__file__).resolve().parent.parent
DATA_DIR = BASE_DIR / "data"
UPLOADS_DIR = DATA_DIR / "uploads"
OUTPUTS_DIR = DATA_DIR / "outputs"


class Settings(BaseSettings):
    """运行时配置。所有字段都可通过环境变量或 backend/.env 覆盖。

    未配置任何真实 key 时，平台自动运行在 Mock 模式：用占位图 + ffmpeg 合成的
    短视频跑通整条流水线，方便在没有付费 API 的情况下体验完整 UI 与流程。
    """

    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    # 强制 Mock 模式（即使配置了 key 也走假数据），便于演示/离线开发
    force_mock: bool = False

    # ---- LLM（剧本拆解 / 分镜提示词）----
    openai_api_key: str = ""
    openai_base_url: str = "https://api.openai.com/v1"
    llm_model: str = "gpt-4o-mini"

    # ---- 图片（GPT 出图，gpt-image-1）----
    image_model: str = "gpt-image-1"
    image_size: str = "1024x1024"

    # ---- 视频（可灵 Kling 图生视频）----
    kling_access_key: str = ""
    kling_secret_key: str = ""
    kling_base_url: str = "https://api-singapore.klingai.com"
    # 可灵模型名，例如 kling-v2-master / kling-v2.5 / 可灵 3.0 对应的官方模型名
    kling_model: str = "kling-v2-master"
    kling_mode: str = "std"          # std | pro
    video_duration: int = 5          # 秒

    # CORS（前端开发地址）
    cors_origins: str = "http://localhost:5173,http://127.0.0.1:5173"

    @property
    def llm_enabled(self) -> bool:
        return bool(self.openai_api_key) and not self.force_mock

    @property
    def image_enabled(self) -> bool:
        return bool(self.openai_api_key) and not self.force_mock

    @property
    def video_enabled(self) -> bool:
        return bool(self.kling_access_key and self.kling_secret_key) and not self.force_mock


@lru_cache
def get_settings() -> Settings:
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    UPLOADS_DIR.mkdir(parents=True, exist_ok=True)
    OUTPUTS_DIR.mkdir(parents=True, exist_ok=True)
    return Settings()
