"""零依赖的 JSON 文件存储。MVP 阶段用单个 db.json 保存全部项目数据。

线程安全：所有读写都用一把进程内锁串行化，足够支撑单机 MVP。
后续要上多用户/高并发时，把这里换成数据库即可，上层接口不变。
"""
from __future__ import annotations

import json
import threading
from pathlib import Path

from .config import DATA_DIR
from .models import Project

_DB_PATH: Path = DATA_DIR / "db.json"
_lock = threading.RLock()


def _load() -> dict[str, dict]:
    if not _DB_PATH.exists():
        return {}
    with _DB_PATH.open("r", encoding="utf-8") as f:
        return json.load(f)


def _dump(raw: dict[str, dict]) -> None:
    tmp = _DB_PATH.with_suffix(".json.tmp")
    with tmp.open("w", encoding="utf-8") as f:
        json.dump(raw, f, ensure_ascii=False, indent=2)
    tmp.replace(_DB_PATH)


def list_projects() -> list[Project]:
    with _lock:
        raw = _load()
    projects = [Project.model_validate(p) for p in raw.values()]
    return sorted(projects, key=lambda p: p.created_at, reverse=True)


def get_project(project_id: str) -> Project | None:
    with _lock:
        raw = _load()
    data = raw.get(project_id)
    return Project.model_validate(data) if data else None


def save_project(project: Project) -> Project:
    import time

    project.updated_at = time.time()
    with _lock:
        raw = _load()
        raw[project.id] = project.model_dump(mode="json")
        _dump(raw)
    return project


def delete_project(project_id: str) -> bool:
    with _lock:
        raw = _load()
        existed = raw.pop(project_id, None) is not None
        if existed:
            _dump(raw)
    return existed
