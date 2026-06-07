import React, { useCallback, useEffect, useRef, useState } from "react";
import { api } from "./api.js";
import CharacterPanel from "./components/CharacterPanel.jsx";
import ShotCard from "./components/ShotCard.jsx";

export default function App() {
  const [config, setConfig] = useState(null);
  const [projects, setProjects] = useState([]);
  const [activeId, setActiveId] = useState(null);
  const [project, setProject] = useState(null);
  const [script, setScript] = useState("");
  const [title, setTitle] = useState("");
  const [busy, setBusy] = useState(false);
  const [generatingAll, setGeneratingAll] = useState(false);
  const [error, setError] = useState(null);
  const [newTitle, setNewTitle] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [confirmState, setConfirmState] = useState(null); // {message, onYes}
  const pollRef = useRef(null);

  const showErr = (e) => {
    setError(String(e.message || e));
    setTimeout(() => setError(null), 5000);
  };

  const refreshProjects = useCallback(async () => {
    try {
      setProjects(await api.listProjects());
    } catch (e) {
      showErr(e);
    }
  }, []);

  useEffect(() => {
    api.config().then(setConfig).catch(showErr);
    refreshProjects();
  }, [refreshProjects]);

  const openProject = useCallback(async (id) => {
    setActiveId(id);
    try {
      const p = await api.getProject(id);
      setProject(p);
      setScript(p.script);
      setTitle(p.title);
    } catch (e) {
      showErr(e);
    }
  }, []);

  // 一键生成期间轮询项目状态
  useEffect(() => {
    if (!generatingAll || !activeId) return;
    pollRef.current = setInterval(async () => {
      try {
        const p = await api.getProject(activeId);
        setProject(p);
        const done = p.shots.every((s) => s.status === "done" || s.status === "failed");
        if (done) {
          setGeneratingAll(false);
          clearInterval(pollRef.current);
        }
      } catch (e) {
        showErr(e);
      }
    }, 2000);
    return () => clearInterval(pollRef.current);
  }, [generatingAll, activeId]);

  const createProject = async () => {
    const t = (newTitle || "未命名短剧").trim();
    setShowCreate(false);
    setNewTitle("");
    try {
      const p = await api.createProject(t, "");
      await refreshProjects();
      openProject(p.id);
    } catch (e) {
      showErr(e);
    }
  };

  const saveScript = async () => {
    setBusy(true);
    try {
      const p = await api.updateProject(activeId, { title, script });
      setProject(p);
      refreshProjects();
    } catch (e) {
      showErr(e);
    } finally {
      setBusy(false);
    }
  };

  const analyze = async () => {
    setBusy(true);
    try {
      await api.updateProject(activeId, { title, script });
      const p = await api.analyze(activeId);
      setProject(p);
    } catch (e) {
      showErr(e);
    } finally {
      setBusy(false);
    }
  };

  const addCharacter = async (data) => {
    const p = await api.addCharacter(activeId, data);
    setProject(p);
  };
  const deleteCharacter = async (cid) => {
    const p = await api.deleteCharacter(activeId, cid);
    setProject(p);
  };

  const saveShot = async (sid, body) => setProject(await api.updateShot(activeId, sid, body));
  const genImage = async (sid) => setProject(await api.genImage(activeId, sid));
  const genVideo = async (sid) => setProject(await api.genVideo(activeId, sid));

  const generateAll = async () => {
    setGeneratingAll(true);
    try {
      await api.generateAll(activeId);
    } catch (e) {
      showErr(e);
      setGeneratingAll(false);
    }
  };

  const deleteProject = (id) => {
    setConfirmState({
      message: "确认删除该项目？",
      onYes: async () => {
        setConfirmState(null);
        await api.deleteProject(id);
        if (id === activeId) {
          setActiveId(null);
          setProject(null);
        }
        refreshProjects();
      },
    });
  };

  const doneCount = project ? project.shots.filter((s) => s.status === "done").length : 0;

  return (
    <>
      <div className="topbar">
        <h1>🎬 AI 影视创作平台</h1>
        {config && (
          <span className={`badge ${config.mock_mode ? "mock" : "live"}`}>
            {config.mock_mode ? "Mock 模式（无需 key）" : "真实模式"}
          </span>
        )}
        {config && !config.mock_mode && (
          <span className="badge">图: {config.image_model} · 视频: {config.kling_model}</span>
        )}
        <div className="spacer" />
        <button className="primary" onClick={() => setShowCreate(true)}>+ 新建项目</button>
      </div>

      <div className="layout">
        <aside className="sidebar">
          {projects.length === 0 && <div className="muted">还没有项目，点右上角新建。</div>}
          {projects.map((p) => (
            <div
              key={p.id}
              className={`proj-item ${p.id === activeId ? "active" : ""}`}
              onClick={() => openProject(p.id)}
            >
              <div className="t">{p.title}</div>
              <div className="s">{p.shots.length} 个分镜 · {p.characters.length} 人物</div>
            </div>
          ))}
        </aside>

        <main className="main">
          {!project ? (
            <div className="empty">从左侧选择或新建一个项目开始创作。</div>
          ) : (
            <>
              <div className="toolbar">
                <input
                  className="title-input"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  onBlur={saveScript}
                />
                <div className="spacer" />
                <button onClick={() => deleteProject(project.id)}>删除项目</button>
              </div>

              <div className="section">
                <h3>剧本</h3>
                <textarea
                  rows={8}
                  placeholder="在此粘贴/编写剧本，用空行分隔不同场景或镜头…"
                  value={script}
                  onChange={(e) => setScript(e.target.value)}
                />
                <div className="row" style={{ marginTop: 10 }}>
                  <button disabled={busy} onClick={saveScript}>保存剧本</button>
                  <button className="primary" disabled={busy || !script.trim()} onClick={analyze}>
                    {busy ? "处理中…" : "拆解剧本 → 生成分镜"}
                  </button>
                </div>
              </div>

              <CharacterPanel
                characters={project.characters}
                onAdd={addCharacter}
                onDelete={deleteCharacter}
              />

              {project.shots.length > 0 && (
                <div className="section">
                  <div className="row" style={{ justifyContent: "space-between", marginBottom: 10 }}>
                    <h3 style={{ margin: 0 }}>
                      分镜故事板（{doneCount}/{project.shots.length} 完成）
                    </h3>
                    <button className="primary" disabled={generatingAll} onClick={generateAll}>
                      {generatingAll ? "生成中…（自动出图+出视频）" : "⚡ 一键生成全部"}
                    </button>
                  </div>
                  <div className="storyboard">
                    {project.shots.map((s) => (
                      <ShotCard
                        key={s.id}
                        shot={s}
                        characters={project.characters}
                        onSave={saveShot}
                        onImage={genImage}
                        onVideo={genVideo}
                      />
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </main>
      </div>

      {error && <div className="toast">{error}</div>}

      {showCreate && (
        <div className="modal-overlay" onClick={() => setShowCreate(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>新建项目</h3>
            <input
              autoFocus
              placeholder="项目标题（如：咖啡馆的重逢）"
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && createProject()}
            />
            <div className="row" style={{ justifyContent: "flex-end", marginTop: 14 }}>
              <button onClick={() => setShowCreate(false)}>取消</button>
              <button className="primary" onClick={createProject}>创建</button>
            </div>
          </div>
        </div>
      )}

      {confirmState && (
        <div className="modal-overlay" onClick={() => setConfirmState(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>{confirmState.message}</h3>
            <div className="row" style={{ justifyContent: "flex-end", marginTop: 14 }}>
              <button onClick={() => setConfirmState(null)}>取消</button>
              <button className="primary" onClick={confirmState.onYes}>确认</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
