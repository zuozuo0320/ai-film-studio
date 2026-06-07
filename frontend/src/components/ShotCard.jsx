import React, { useEffect, useState } from "react";
import { mediaUrl } from "../api.js";

const STATUS_TEXT = {
  pending: "待生成",
  image_running: "出图中…",
  image_done: "已出图",
  video_running: "生视频中…",
  done: "完成",
  failed: "失败",
};

export default function ShotCard({ shot, characters, onSave, onImage, onVideo }) {
  const [imagePrompt, setImagePrompt] = useState(shot.image_prompt);
  const [videoPrompt, setVideoPrompt] = useState(shot.video_prompt);
  const [busy, setBusy] = useState(false);

  // 后台一键生成会改变 shot，外部更新时同步本地编辑框
  useEffect(() => {
    setImagePrompt(shot.image_prompt);
    setVideoPrompt(shot.video_prompt);
  }, [shot.image_prompt, shot.video_prompt]);

  const names = characters.filter((c) => shot.character_ids.includes(c.id)).map((c) => c.name);
  const running = shot.status === "image_running" || shot.status === "video_running";

  const saveIfChanged = async () => {
    if (imagePrompt !== shot.image_prompt || videoPrompt !== shot.video_prompt) {
      await onSave(shot.id, { image_prompt: imagePrompt, video_prompt: videoPrompt });
    }
  };

  const doImage = async () => {
    setBusy(true);
    try {
      await saveIfChanged();
      await onImage(shot.id);
    } finally {
      setBusy(false);
    }
  };
  const doVideo = async () => {
    setBusy(true);
    try {
      await onVideo(shot.id);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="shot">
      <div className="media">
        {shot.video_path ? (
          <video src={mediaUrl(shot.video_path)} controls loop muted />
        ) : shot.image_path ? (
          <img src={mediaUrl(shot.image_path)} alt={shot.summary} />
        ) : (
          <span className="placeholder">未出图</span>
        )}
      </div>
      <div className="body">
        <div className="row" style={{ justifyContent: "space-between" }}>
          <span className="idx">镜 {shot.index + 1}</span>
          <span className={`status ${shot.status}`}>{STATUS_TEXT[shot.status] || shot.status}</span>
        </div>
        <div className="sum">{shot.summary}</div>
        {names.length > 0 && <div className="label">人物：{names.join("、")}</div>}

        <div className="label">出图提示词</div>
        <textarea value={imagePrompt} onChange={(e) => setImagePrompt(e.target.value)} onBlur={saveIfChanged} />
        <div className="label">运镜提示词（图生视频）</div>
        <textarea value={videoPrompt} onChange={(e) => setVideoPrompt(e.target.value)} onBlur={saveIfChanged} />

        {shot.error && <div className="err">{shot.error}</div>}

        <div className="actions">
          <button disabled={busy || running} onClick={doImage}>
            {shot.image_path ? "重新出图" : "出图"}
          </button>
          <button className="primary" disabled={busy || running || !shot.image_path} onClick={doVideo}>
            图生视频
          </button>
        </div>
      </div>
    </div>
  );
}
