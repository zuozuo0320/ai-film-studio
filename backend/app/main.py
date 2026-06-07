from __future__ import annotations

from pathlib import Path

from fastapi import BackgroundTasks, FastAPI, File, Form, HTTPException, UploadFile
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles

from . import pipeline, storage
from .config import DATA_DIR, UPLOADS_DIR, get_settings
from .models import (
    Character,
    Project,
    ProjectCreate,
    ProjectUpdate,
    ShotUpdate,
)

settings = get_settings()
app = FastAPI(title="AI 影视创作平台 API", version="0.1.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=[o.strip() for o in settings.cors_origins.split(",") if o.strip()],
    allow_methods=["*"],
    allow_headers=["*"],
)

# 生成的图片/视频与上传的参考图，统一通过 /media/<相对data路径> 访问
app.mount("/media", StaticFiles(directory=str(DATA_DIR)), name="media")


@app.get("/api/health")
def health() -> dict:
    return {"status": "ok"}


@app.get("/api/config")
def config() -> dict:
    """前端用来展示当前是真实模式还是 Mock 模式。"""
    return {
        "mock_mode": not (settings.llm_enabled or settings.image_enabled or settings.video_enabled),
        "providers": {
            "llm": "openai" if settings.llm_enabled else "mock",
            "image": "openai-gpt-image-1" if settings.image_enabled else "mock",
            "video": "kling" if settings.video_enabled else "mock",
        },
        "kling_model": settings.kling_model,
        "image_model": settings.image_model,
    }


# ---------------- 项目 ----------------

@app.get("/api/projects", response_model=list[Project])
def list_projects() -> list[Project]:
    return storage.list_projects()


@app.post("/api/projects", response_model=Project)
def create_project(body: ProjectCreate) -> Project:
    return storage.save_project(Project(title=body.title, script=body.script))


@app.get("/api/projects/{project_id}", response_model=Project)
def get_project(project_id: str) -> Project:
    return _require(project_id)


@app.patch("/api/projects/{project_id}", response_model=Project)
def update_project(project_id: str, body: ProjectUpdate) -> Project:
    project = _require(project_id)
    if body.title is not None:
        project.title = body.title
    if body.script is not None:
        project.script = body.script
    return storage.save_project(project)


@app.delete("/api/projects/{project_id}")
def delete_project(project_id: str) -> dict:
    if not storage.delete_project(project_id):
        raise HTTPException(404, "project not found")
    return {"deleted": True}


# ---------------- 人物 ----------------

@app.post("/api/projects/{project_id}/characters", response_model=Project)
async def add_character(
    project_id: str,
    name: str = Form(...),
    description: str = Form(""),
    image: UploadFile | None = File(None),
) -> Project:
    project = _require(project_id)
    char = Character(name=name, description=description)
    if image is not None and image.filename:
        ext = Path(image.filename).suffix or ".png"
        dest = UPLOADS_DIR / f"{char.id}{ext}"
        dest.write_bytes(await image.read())
        char.ref_image = str(dest.relative_to(DATA_DIR))
    project.characters.append(char)
    return storage.save_project(project)


@app.delete("/api/projects/{project_id}/characters/{char_id}", response_model=Project)
def delete_character(project_id: str, char_id: str) -> Project:
    project = _require(project_id)
    project.characters = [c for c in project.characters if c.id != char_id]
    for shot in project.shots:
        shot.character_ids = [cid for cid in shot.character_ids if cid != char_id]
    return storage.save_project(project)


# ---------------- 分镜 / 生成 ----------------

@app.post("/api/projects/{project_id}/analyze", response_model=Project)
def analyze(project_id: str) -> Project:
    project = _require(project_id)
    if not project.script.strip():
        raise HTTPException(400, "剧本为空，无法拆解")
    return pipeline.analyze_script(project)


@app.patch("/api/projects/{project_id}/shots/{shot_id}", response_model=Project)
def update_shot(project_id: str, shot_id: str, body: ShotUpdate) -> Project:
    project = _require(project_id)
    shot = next((s for s in project.shots if s.id == shot_id), None)
    if shot is None:
        raise HTTPException(404, "shot not found")
    if body.summary is not None:
        shot.summary = body.summary
    if body.image_prompt is not None:
        shot.image_prompt = body.image_prompt
    if body.video_prompt is not None:
        shot.video_prompt = body.video_prompt
    if body.character_ids is not None:
        shot.character_ids = body.character_ids
    return storage.save_project(project)


@app.post("/api/projects/{project_id}/shots/{shot_id}/image", response_model=Project)
def shot_image(project_id: str, shot_id: str) -> Project:
    _require(project_id)
    pipeline.generate_image(project_id, shot_id)
    return _require(project_id)


@app.post("/api/projects/{project_id}/shots/{shot_id}/video", response_model=Project)
def shot_video(project_id: str, shot_id: str) -> Project:
    _require(project_id)
    pipeline.generate_video(project_id, shot_id)
    return _require(project_id)


@app.post("/api/projects/{project_id}/generate-all", response_model=Project)
def generate_all(project_id: str, background: BackgroundTasks) -> Project:
    project = _require(project_id)
    if not project.shots:
        raise HTTPException(400, "请先拆解剧本生成分镜")
    background.add_task(pipeline.generate_all, project_id)
    return project


def _require(project_id: str) -> Project:
    project = storage.get_project(project_id)
    if project is None:
        raise HTTPException(404, "project not found")
    return project
