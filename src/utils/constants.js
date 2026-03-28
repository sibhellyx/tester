export const STATUS_META = {
  pending:  { color: "#f59e0b", bg: "rgba(245,158,11,0.10)",  dot: "#f59e0b" },
  running:  { color: "#22d3ee", bg: "rgba(34,211,238,0.10)",  dot: "#22d3ee" },
  stopped:  { color: "#94a3b8", bg: "rgba(148,163,184,0.10)", dot: "#94a3b8" },
  finished: { color: "#4ade80", bg: "rgba(74,222,128,0.10)",  dot: "#4ade80" },
  failed:   { color: "#f87171", bg: "rgba(248,113,113,0.10)", dot: "#f87171" },
};

export const STAGE_COLORS = { ramp_up: "#22d3ee", steady: "#4ade80", ramp_down: "#f59e0b" };

export const CHAOS_COLORS = {
  component_shutdown: "#f87171",
  network_delay:      "#f59e0b",
  packet_loss:        "#fb7185",
  resource_limit:     "#a78bfa",
};

export const CHART_COLORS = ["#22d3ee","#4ade80","#f59e0b","#f87171","#a78bfa","#fb7185"];

export const EMPTY_SCENARIO = {
  name: "", base_url: "", total_duration: 0,
  stages: [
    { id: 1, type: "ramp_up",   duration: 30, target_users: 10, requests: [], chaos_events: [] },
    { id: 2, type: "steady",    duration: 60, target_users: 10, requests: [], chaos_events: [] },
    { id: 3, type: "ramp_down", duration: 30, target_users: 1,  requests: [], chaos_events: [] },
  ],
  stop_conditions: null,
};