import { useState } from "react";
import { STATUS_META } from "../../utils/constants";
import { S } from "../../styles/theme";

export function Badge({ status }) {
  const m = STATUS_META[status] || STATUS_META.pending;
  return (
    <span style={{
      display: "inline-flex", alignItems: "center", gap: 5,
      padding: "6px 10px", borderRadius: 4, fontSize: 11, fontWeight: 700,
      letterSpacing: "0.1em", color: m.color, background: m.bg,
      border: `1px solid ${m.color}30`, fontFamily: "monospace",
    }}>
      <span style={{ width: 5, height: 5, borderRadius: "50%", background: m.dot, boxShadow: status === "running" ? `0 0 6px ${m.dot}` : "none" }} />
      {status?.toUpperCase()}
    </span>
  );
}

export function StatCard({ label, value, accent, sub }) {
  return (
    <div style={{ ...S.card, padding: "12px 16px", minWidth: 100 }}>
      <div style={{ ...S.label, marginBottom: 6 }}>{label}</div>
      <div style={{ fontSize: 20, fontWeight: 700, color: accent || "#e2e8f0", fontFamily: "monospace", lineHeight: 1 }}>{value}</div>
      {sub && <div style={{ fontSize: 10, color: "#3d5068", fontFamily: "monospace", marginTop: 4 }}>{sub}</div>}
    </div>
  );
}

export function Pill({ color, label }) {
  return (
    <span style={{
      display: "inline-block", padding: "1px 7px", borderRadius: 3,
      fontSize: 10, fontWeight: 700, letterSpacing: "0.08em",
      color, border: `1px solid ${color}40`, background: `${color}12`,
      fontFamily: "monospace",
    }}>{label}</span>
  );
}

export function Flash({ msg }) {
  const [expanded, setExpanded] = useState(false);
  if (!msg) return null;
  const ok = msg.startsWith("✓");
  const isLong = msg.length > 120;
  const display = (!ok && isLong && !expanded) ? msg.slice(0, 120) + "…" : msg;
  return (
    <div style={{
      padding: "10px 14px", borderRadius: 5, marginBottom: 14,
      background: ok ? "rgba(74,222,128,0.08)" : "rgba(248,113,113,0.08)",
      border: `1px solid ${ok ? "#4ade8040" : "#f8717140"}`,
      color: ok ? "#4ade80" : "#f87171", fontSize: 12, fontFamily: "monospace",
      lineHeight: 1.6,
    }}>
      <span>{display}</span>
      {!ok && isLong && (
        <button onClick={() => setExpanded(e => !e)} style={{
          background: "none", border: "none", color: "#f87171", cursor: "pointer",
          fontSize: 11, fontFamily: "monospace", marginLeft: 8, textDecoration: "underline",
        }}>{expanded ? "свернуть" : "подробнее"}</button>
      )}
    </div>
  );
}