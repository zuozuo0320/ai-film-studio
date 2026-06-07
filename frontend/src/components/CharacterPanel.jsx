import React, { useRef, useState } from "react";
import { mediaUrl } from "../api.js";

export default function CharacterPanel({ characters, onAdd, onDelete }) {
  const [name, setName] = useState("");
  const [desc, setDesc] = useState("");
  const [file, setFile] = useState(null);
  const [busy, setBusy] = useState(false);
  const fileRef = useRef();

  const submit = async () => {
    if (!name.trim()) return;
    setBusy(true);
    try {
      await onAdd({ name: name.trim(), description: desc.trim(), file });
      setName("");
      setDesc("");
      setFile(null);
      if (fileRef.current) fileRef.current.value = "";
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="section">
      <h3>人物设定（参考图用于保持出图一致性）</h3>
      <div className="char-grid">
        {characters.map((c) => (
          <div className="char-card" key={c.id}>
            {c.ref_image ? (
              <img src={mediaUrl(c.ref_image)} alt={c.name} />
            ) : (
              <div className="noimg">无参考图</div>
            )}
            <div className="n">{c.name}</div>
            <div className="d">{c.description}</div>
            <button style={{ marginTop: 6, width: "100%" }} onClick={() => onDelete(c.id)}>
              删除
            </button>
          </div>
        ))}

        <div className="add-char-form">
          <input placeholder="人物名（如 林夏）" value={name} onChange={(e) => setName(e.target.value)} />
          <textarea
            placeholder="人物描述（外貌/服装/气质）"
            value={desc}
            onChange={(e) => setDesc(e.target.value)}
          />
          <input ref={fileRef} type="file" accept="image/*" onChange={(e) => setFile(e.target.files[0])} />
          <button className="primary" disabled={busy || !name.trim()} onClick={submit}>
            {busy ? "添加中…" : "+ 添加人物"}
          </button>
        </div>
      </div>
    </div>
  );
}
