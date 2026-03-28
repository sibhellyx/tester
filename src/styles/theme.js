export const S = {
  input: {
    background: "#0a0e17", border: "1px solid #1e2a3a", borderRadius: 4,
    color: "#e2e8f0", padding: "7px 10px", fontFamily: "'JetBrains Mono', 'Fira Mono', monospace",
    fontSize: 12, outline: "none", width: "100%", boxSizing: "border-box",
  },
  label: {
    fontSize: 10, color: "#3d5068", fontFamily: "monospace",
    letterSpacing: "0.1em", display: "block", marginBottom: 4, textTransform: "uppercase",
  },
  card: {
    background: "#0b0f18", border: "1px solid #1a2535", borderRadius: 8,
    padding: "16px 20px",
  },
  btn: (variant = "ghost") => ({
    background: variant === "primary" ? "#22d3ee" : variant === "danger" ? "rgba(248,113,113,0.12)" : "rgba(255,255,255,0.04)",
    border: variant === "primary" ? "none" : `1px solid ${variant === "danger" ? "#f8717140" : "#1e2a3a"}`,
    borderRadius: 4, color: variant === "primary" ? "#070a10" : variant === "danger" ? "#f87171" : "#64748b",
    padding: "6px 14px", cursor: "pointer", fontSize: 11, fontWeight: variant === "primary" ? 700 : 500,
    fontFamily: "monospace", letterSpacing: "0.06em",
  }),
};