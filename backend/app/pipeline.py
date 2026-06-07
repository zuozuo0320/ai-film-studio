"""核心流水线：剧本 -> 分镜 -> 出图 -> 图生视频。

每一步都会即时把结果写回 storage，前端轮询 status 即可看到进度。
单镜失败不影响其他镜（标记 failed 并记录 error）。
"""
from __future__ import annotations

from pathlib import Path

from .config import OUTPUTS_DIR, DATA_DIR, get_settings
from .models import Project, Shot, ShotStatus
from .providers import registry
from . import storage


def _rel(path: Path) -> str:
    """转成相对 data/ 的路径，便于前端通过 /media/ 访问。"""
    return str(path.relative_to(DATA_DIR))


def analyze_script(project: Project) -> Project:
    """用 LLM（或 Mock）把剧本拆成分镜，写回 project.shots。"""
    settings = get_settings()
    llm = registry.get_llm(settings)
    drafts = llm.analyze_script(project.script, project.characters)

    name_to_id = {c.name: c.id for c in project.characters}
    shots: list[Shot] = []
    for i, d in enumerate(drafts):
        char_ids = [name_to_id[n] for n in d.character_names if n in name_to_id]
        shots.append(
            Shot(
                index=i,
                summary=d.summary,
                image_prompt=d.image_prompt,
                video_prompt=d.video_prompt,
                character_ids=char_ids,
            )
        )
    project.shots = shots
    return storage.save_project(project)


def _ref_paths(project: Project, shot: Shot) -> list[Path]:
    paths: list[Path] = []
    for c in project.characters:
        if c.id in shot.character_ids and c.ref_image:
            p = DATA_DIR / c.ref_image
            if p.exists():
                paths.append(p)
    return paths


def generate_image(project_id: str, shot_id: str) -> Shot:
    settings = get_settings()
    provider = registry.get_image(settings)
    project = storage.get_project(project_id)
    if project is None:
        raise ValueError("project not found")
    shot = _find_shot(project, shot_id)

    shot.status = ShotStatus.image_running
    shot.error = None
    storage.save_project(project)
    try:
        data = provider.generate(shot.image_prompt, _ref_paths(project, shot))
        out = OUTPUTS_DIR / f"{shot.id}.png"
        out.write_bytes(data)
        shot.image_path = _rel(out)
        shot.status = ShotStatus.image_done
    except Exception as exc:  # noqa: BLE001 - 单镜失败需记录而非中断
        shot.status = ShotStatus.failed
        shot.error = f"出图失败: {exc}"
    storage.save_project(project)
    return shot


def generate_video(project_id: str, shot_id: str) -> Shot:
    settings = get_settings()
    provider = registry.get_video(settings)
    project = storage.get_project(project_id)
    if project is None:
        raise ValueError("project not found")
    shot = _find_shot(project, shot_id)

    if not shot.image_path:
        shot.status = ShotStatus.failed
        shot.error = "请先出图再生成视频"
        storage.save_project(project)
        return shot

    shot.status = ShotStatus.video_running
    shot.error = None
    storage.save_project(project)
    try:
        image_path = DATA_DIR / shot.image_path
        data = provider.image_to_video(image_path, shot.video_prompt, settings.video_duration)
        out = OUTPUTS_DIR / f"{shot.id}.mp4"
        out.write_bytes(data)
        shot.video_path = _rel(out)
        shot.status = ShotStatus.done
    except Exception as exc:  # noqa: BLE001
        shot.status = ShotStatus.failed
        shot.error = f"图生视频失败: {exc}"
    storage.save_project(project)
    return shot


def generate_all(project_id: str) -> None:
    """一键生成：对每个分镜依次出图 + 图生视频。供后台任务调用。"""
    project = storage.get_project(project_id)
    if project is None:
        return
    for shot in project.shots:
        s = generate_image(project_id, shot.id)
        if s.status == ShotStatus.failed:
            continue
        generate_video(project_id, shot.id)


def _find_shot(project: Project, shot_id: str) -> Shot:
    for s in project.shots:
        if s.id == shot_id:
            return s
    raise ValueError("shot not found")
