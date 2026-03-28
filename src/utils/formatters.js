export const fmt = {
  ms:    (v) => v == null ? "—" : `${v} ms`,
  pct:   (v) => v == null ? "—" : `${(v * 100).toFixed(1)}%`,
  bytes: (v) => {
    if (v == null) return "—";
    if (v > 1e6) return `${(v / 1e6).toFixed(2)} MB`;
    if (v > 1e3) return `${(v / 1e3).toFixed(1)} KB`;
    return `${v} B`;
  },
  rps:   (v) => v == null ? "—" : `${v.toFixed(1)} rps`,
  time:  (s) => !s ? "—" : new Date(s).toLocaleString("ru-RU", { dateStyle: "short", timeStyle: "medium" }),
  dur:   (s) => !s ? "—" : s < 60 ? `${s}с` : `${Math.floor(s/60)}м ${s%60}с`,
};