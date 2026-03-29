import { useState } from "react";
import { STAGE_COLORS, CHAOS_COLORS, EMPTY_SCENARIO } from "../../utils/constants";
import { fmt } from "../../utils/formatters";
import { S } from "../../styles/theme";
import { Pill } from "../ui";

export function StageTimeline({ stages }) {
  if (!stages?.length) return null;
  const total = stages.reduce((s, st) => s + (st.duration || 0), 0);
  return (
    <div style={{ marginBottom: 16 }}>
      <div style={{ display: "flex", height: 28, borderRadius: 5, overflow: "hidden", gap: 2 }}>
        {stages.map((st, i) => {
          const pct = total ? (st.duration / total) * 100 : 0;
          const col = STAGE_COLORS[st.type] || "#64748b";
          const hasChaos = (st.chaos_events || []).length > 0;
          return (
            <div key={i} title={`${st.type} · ${st.duration}с · ${st.target_users} VU${hasChaos ? " · CHAOS" : ""}`}
              style={{
                width: `${pct}%`, background: `${col}18`, border: `1px solid ${col}40`,
                borderRadius: 3, display: "flex", alignItems: "center", justifyContent: "center",
                fontSize: 9, color: col, fontFamily: "monospace", letterSpacing: "0.06em",
                overflow: "hidden", whiteSpace: "nowrap", cursor: "default", position: "relative",
              }}>
              {hasChaos && <span style={{ position: "absolute", top: 2, right: 3, fontSize: 8, color: "#f87171" }}>⚡</span>}
              {pct > 8 ? `${st.target_users}VU` : ""}
            </div>
          );
        })}
      </div>
      <div style={{ display: "flex", gap: 12, marginTop: 5 }}>
        {stages.map((st, i) => (
          <span key={i} style={{ fontSize: 9, color: STAGE_COLORS[st.type] || "#64748b", fontFamily: "monospace" }}>
            {st.type} {fmt.dur(st.duration)}
          </span>
        ))}
      </div>
    </div>
  );
}

// containers — список доступных Docker-контейнеров, загружается в ScenariosTab
// и передаётся сюда чтобы пользователь мог выбрать контейнер из списка
export function ChaosEventForm({ event, onChange, onRemove, containers }) {
  console.log(containers)
  const iStyle = { ...S.input, fontSize: 11 };
  return (
    <div style={{ background: "#060910", border: `1px solid ${CHAOS_COLORS[event.type] || "#1e2a3a"}30`, borderRadius: 6, padding: "12px 14px", marginBottom: 8 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 10 }}>
        <Pill color={CHAOS_COLORS[event.type] || "#64748b"} label={event.type || "CHAOS"} />
        <button onClick={onRemove} style={{ ...S.btn("danger"), padding: "2px 8px", fontSize: 10 }}>✕</button>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "2fr 2fr 1fr 1fr", gap: 8, marginBottom: 8 }}>
        <div>
          <label style={S.label}>ТИП СБОЯ</label>
          <select style={iStyle} value={event.type} onChange={(e) => onChange("type", e.target.value)}>
            <option value="component_shutdown">component_shutdown — убить контейнер</option>
            <option value="network_delay">network_delay — задержка сети</option>
            <option value="packet_loss">packet_loss — потеря пакетов</option>
            <option value="resource_limit">resource_limit — ограничение CPU/RAM</option>
          </select>
        </div>
        <div>
          <label style={S.label}>КОНТЕЙНЕР</label>
          {containers && containers.length > 0 ? (
            // Если контейнеры загружены — показываем select
            <select
              style={iStyle}
              value={event.target_container || ""}
              onChange={(e) => onChange("target_container", e.target.value)}
            >
              <option value="">— выберите контейнер —</option>
              {containers.map((ct) => (
                <option key={ct.ID} value={ct.ID}>
                  {ct.Name} · {ct.Image} · {ct.Status}
                </option>
              ))}
            </select>
          ) : (
            // Fallback — ручной ввод если контейнеры не загрузились или Docker недоступен
            <input
              style={iStyle}
              value={event.target_container || ""}
              onChange={(e) => onChange("target_container", e.target.value)}
              placeholder="my-service"
            />
          )}
        </div>
        <div>
          <label style={S.label}>СТАРТ (сек)</label>
          <input style={iStyle} type="number" min={0} value={event.start_delay ?? 0} onChange={(e) => onChange("start_delay", +e.target.value)} />
        </div>
        <div>
          <label style={S.label}>ДЛИТ. (сек)</label>
          <input style={iStyle} type="number" min={1} value={event.duration ?? 10} onChange={(e) => onChange("duration", +e.target.value)} />
        </div>
      </div>

      {event.type === "network_delay" && (
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
          <div>
            <label style={S.label}>DELAY (напр. "100ms")</label>
            <input style={iStyle} value={event.delay || ""} onChange={(e) => onChange("delay", e.target.value)} placeholder="100ms" />
          </div>
          <div>
            <label style={S.label}>JITTER (напр. "10ms")</label>
            <input style={iStyle} value={event.jitter || ""} onChange={(e) => onChange("jitter", e.target.value)} placeholder="10ms" />
          </div>
        </div>
      )}
      {event.type === "packet_loss" && (
        <div style={{ maxWidth: 140 }}>
          <label style={S.label}>ПОТЕРЯ ПАКЕТОВ (%)</label>
          <input style={iStyle} type="number" min={0} max={100} value={event.packet_loss ?? 10} onChange={(e) => onChange("packet_loss", +e.target.value)} />
        </div>
      )}
      {event.type === "resource_limit" && (
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
          <div>
            <label style={S.label}>CPU QUOTA (−1..100000)</label>
            <input style={iStyle} type="number" value={event.cpu_quota ?? 50000} onChange={(e) => onChange("cpu_quota", +e.target.value)} />
          </div>
          <div>
            <label style={S.label}>MEMORY LIMIT (bytes)</label>
            <input style={iStyle} type="number" value={event.memory_bytes ?? 134217728} onChange={(e) => onChange("memory_bytes", +e.target.value)} placeholder="134217728 = 128MB" />
          </div>
        </div>
      )}
    </div>
  );
}

export function RequestRow({ req, onChange, onRemove, totalWeight }) {
  const weightPct = totalWeight > 0 ? Math.round(((req.probability ?? 0) / totalWeight) * 100) : 0;
  const METHOD_COLORS = { GET: "#4ade80", POST: "#22d3ee", PUT: "#f59e0b", DELETE: "#f87171", PATCH: "#a78bfa" };
  const iStyle = { ...S.input, fontSize: 11 };
  const sel = (e) => e.target.select();
  const hasBody = ["POST", "PUT", "PATCH"].includes(req.method);

  const [codeInput, setCodeInput] = useState("");
  const [headerKey, setHeaderKey] = useState("");
  const [headerVal, setHeaderVal] = useState("");

  const addCode = (val) => {
    const code = parseInt(val);
    if (!code) return;
    const current = req.expected_codes || [200];
    if (current.includes(code)) { setCodeInput(""); return; }
    onChange("expected_codes", [...current, code]);
    setCodeInput("");
  };

  const removeCode = (code) => {
    onChange("expected_codes", (req.expected_codes || []).filter(c => c !== code));
  };

  return (
    <div style={{ border: "1px solid #1a2535", borderRadius: 6, padding: "12px 14px", marginBottom: 8, background: "#060910" }}>
      <div style={{ display: "grid", gridTemplateColumns: "90px 150px 1fr 90px 28px", gap: 8, alignItems: "end" }}>
        <div>
          <label style={S.label}>МЕТОД</label>
          <select style={{ ...iStyle, color: METHOD_COLORS[req.method] || "#e2e8f0" }} value={req.method}
            onChange={(e) => onChange("method", e.target.value)}>
            {["GET", "POST", "PUT", "DELETE", "PATCH"].map((m) => <option key={m}>{m}</option>)}
          </select>
        </div>
        <div>
          <label style={S.label}>НАЗВАНИЕ</label>
          <input style={iStyle} value={req.name || ""} onChange={(e) => onChange("name", e.target.value)} placeholder="GetUsers" />
        </div>
        <div>
          <label style={S.label}>ENDPOINT</label>
          <input style={iStyle} value={req.endpoint || ""} onChange={(e) => onChange("endpoint", e.target.value)} placeholder="/api/v1/users" />
        </div>
        <div>
          <label style={S.label}>ВЕРОЯТНОСТЬ ({weightPct}%)</label>
          <input style={iStyle} type="number" min={0} value={req.probability || ""}
            placeholder="1" onFocus={sel} onChange={(e) => onChange("probability", +e.target.value)} />
        </div>
        <button onClick={onRemove} style={{ ...S.btn("danger"), padding: "6px 8px", alignSelf: "end" }}>✕</button>
      </div>

      {/* Статус коды — теги + поле ввода нового кода */}
      <div style={{ marginTop: 8 }}>
        <label style={S.label}>ОЖ. СТАТУС КОДЫ (Enter для добавления)</label>
        <div style={{ display: "flex", flexWrap: "wrap", alignItems: "center", gap: 4, minHeight: 28 }}>
          {(req.expected_codes || [200]).map((code) => (
            <span key={code} style={{
              display: "inline-flex", alignItems: "center", gap: 3,
              height: 32, padding: "0 7px", borderRadius: 3, fontSize: 10, fontFamily: "monospace", fontWeight: 700,
              color: code < 400 ? "#4ade80" : "#f87171",
              border: `1px solid ${code < 400 ? "#4ade8040" : "#f8717140"}`,
              background: code < 400 ? "#4ade8012" : "#f8717112",
            }}>
              {code}
              <button
                onClick={() => removeCode(code)}
                style={{ background: "none", border: "none", cursor: "pointer", color: "inherit", fontSize: 9, padding: 0, lineHeight: 1 }}
              >✕</button>
            </span>
          ))}
          <input
            style={{ ...iStyle, width: 60 }}
            type="number"
            value={codeInput}
            placeholder="201"
            onChange={(e) => setCodeInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") { e.preventDefault(); addCode(codeInput); }
            }}
            onBlur={() => { if (codeInput) addCode(codeInput); }}
          />
        </div>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: hasBody ? "1fr 1fr" : "1fr", gap: 8, marginTop: 8 }}>
        <div>
          <label style={S.label}>HEADERS</label>
          {Object.entries(req.headers || {}).map(([k, v]) => (
            <div key={k} style={{ display: "flex", gap: 4, marginBottom: 4 }}>
              <input
                style={{ ...iStyle, flex: "0 0 38%", color: "#94a3b8" }}
                value={k}
                onChange={(e) => {
                  const h = { ...req.headers };
                  delete h[k];
                  h[e.target.value] = v;
                  onChange("headers", h);
                }}
              />
              <span style={{ color: "#1e3050", alignSelf: "center", fontSize: 11 }}>:</span>
              <input
                style={{ ...iStyle, flex: 1 }}
                value={v}
                onChange={(e) => onChange("headers", { ...req.headers, [k]: e.target.value })}
              />
              <button
                onClick={() => {
                  const h = { ...req.headers };
                  delete h[k];
                  onChange("headers", h);
                }}
                style={{ ...S.btn("danger"), padding: "4px 7px", fontSize: 10, flexShrink: 0 }}
              >✕</button>
            </div>
          ))}
          <div style={{ display: "flex", gap: 4 }}>
            <input
              style={{ ...iStyle, flex: "0 0 38%", color: "#94a3b8" }}
              value={headerKey}
              placeholder="Authorization"
              onChange={(e) => setHeaderKey(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && headerKey.trim()) {
                  e.preventDefault();
                  onChange("headers", { ...req.headers, [headerKey.trim()]: headerVal });
                  setHeaderKey(""); setHeaderVal("");
                }
              }}
            />
            <span style={{ color: "#1e3050", alignSelf: "center", fontSize: 11 }}>:</span>
            <input
              style={{ ...iStyle, flex: 1 }}
              value={headerVal}
              placeholder="Bearer token"
              onChange={(e) => setHeaderVal(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && headerKey.trim()) {
                  e.preventDefault();
                  onChange("headers", { ...req.headers, [headerKey.trim()]: headerVal });
                  setHeaderKey(""); setHeaderVal("");
                }
              }}
            />
            <button
              onClick={() => {
                if (!headerKey.trim()) return;
                onChange("headers", { ...req.headers, [headerKey.trim()]: headerVal });
                setHeaderKey(""); setHeaderVal("");
              }}
              style={{ ...S.btn(), padding: "4px 8px", fontSize: 10, flexShrink: 0 }}
            >+</button>
          </div>
        </div>
        {hasBody && (
          <div>
            <label style={S.label}>BODY (JSON)</label>
            <textarea style={{ ...iStyle, height: 64, padding: "7px 10px", resize: "vertical" }}
              value={req.body || ""} onChange={(e) => onChange("body", e.target.value)}
              placeholder='{"key":"value"}' />
          </div>
        )}
      </div>
    </div>
  );
}

export function StageEditor({ stage, idx, onChange, onRemove, containers }) {
  const totalWeight = (stage.requests || []).reduce((s, r) => s + (r.probability ?? 0), 0);

  const updateReq = (ri, field, val) => {
    const reqs = [...(stage.requests || [])];
    reqs[ri] = { ...reqs[ri], [field]: val };
    onChange("requests", reqs);
  };

  const updateChaos = (ci, field, val) => {
    const events = [...(stage.chaos_events || [])];
    events[ci] = { ...events[ci], [field]: val };
    onChange("chaos_events", events);
  };

  const col = STAGE_COLORS[stage.type] || "#64748b";

  return (
    <div style={{ border: `1px solid ${col}30`, borderRadius: 8, marginBottom: 14, overflow: "hidden" }}>
      <div style={{ background: `${col}0a`, borderBottom: `1px solid ${col}20`, padding: "10px 16px", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span style={{ fontSize: 10, color: col, fontFamily: "monospace", fontWeight: 700, letterSpacing: "0.1em" }}>ЭТАП {idx + 1}</span>
          <Pill color={col} label={stage.type} />
          <span style={{ fontSize: 10, color: "#3d5068", fontFamily: "monospace" }}>{stage.target_users} VU · {fmt.dur(stage.duration)}</span>
          {(stage.chaos_events || []).length > 0 && <Pill color="#f87171" label={`⚡ ${stage.chaos_events.length} chaos`} />}
        </div>
        <button onClick={onRemove} style={{ ...S.btn("danger"), padding: "2px 9px", fontSize: 10 }}>✕ убрать</button>
      </div>

      <div style={{ padding: "14px 16px" }}>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 10, marginBottom: 18 }}>
          <div>
            <label style={S.label}>ТИП ЭТАПА</label>
            <select style={S.input} value={stage.type} onChange={(e) => onChange("type", e.target.value)}>
              <option value="ramp_up">ramp_up — плавный разгон</option>
              <option value="steady">steady — стабильная нагрузка</option>
              <option value="ramp_down">ramp_down — плавное снижение</option>
              <option value="spike">spike — стрессовый пик</option>
            </select>
          </div>
          <div>
            <label style={S.label}>ДЛИТЕЛЬНОСТЬ (секунд)</label>
            <input style={S.input} type="number" min={1} value={stage.duration || ""}
              placeholder="30" onFocus={(e) => e.target.select()} onChange={(e) => onChange("duration", +e.target.value)} />
          </div>
          <div>
            <label style={S.label}>ЦЕЛЕВЫЕ VU (пользователи)</label>
            <input style={S.input} type="number" min={1} value={stage.target_users || ""}
              placeholder="10" onFocus={(e) => e.target.select()} onChange={(e) => onChange("target_users", +e.target.value)} />
          </div>
        </div>

        {stage.type === "spike" && (
          <div style={{ marginBottom: 18 }}>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 2fr", gap: 10, padding: "12px 14px", background: "rgba(244,63,94,0.04)", border: "1px solid rgba(244,63,94,0.15)", borderRadius: 6 }}>
              <div>
                <label style={S.label}>BASELINE USERS (после пика)</label>
                <input style={S.input} type="number" min={0} value={stage.baseline_users ?? 0}
                  onFocus={(e) => e.target.select()} onChange={(e) => onChange("baseline_users", +e.target.value)} />
              </div>
              <div style={{ display: "flex", alignItems: "flex-end", paddingBottom: 2 }}>
                <span style={{ fontSize: 9, color: "#64748b", fontFamily: "monospace", lineHeight: 1.5 }}>
                  Пик до {stage.target_users} VU, затем возврат к {stage.baseline_users || "предыдущему уровню"} VU.{"\n"}
                  0 = вернуться к числу VU до начала этапа.
                </span>
              </div>
            </div>
          </div>
        )}

        <div style={{ marginBottom: 18 }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 8 }}>
            <span style={{ fontSize: 10, color: "#3d5068", fontFamily: "monospace", letterSpacing: "0.1em" }}>
              ЗАПРОСЫ
              {totalWeight > 0 && <span style={{ color: "#1e3050", marginLeft: 8 }}>сумма вероятностей: {totalWeight}</span>}
            </span>
            <button onClick={() => onChange("requests", [...(stage.requests || []), { name: "", method: "GET", endpoint: "", probability: 1, body: "", expected_codes: [200], headers: {} }])} style={S.btn()}>+ запрос</button>
          </div>
          {(stage.requests || []).map((req, ri) => (
            <RequestRow key={ri} req={req} totalWeight={totalWeight}
              onChange={(f, v) => updateReq(ri, f, v)}
              onRemove={() => onChange("requests", (stage.requests || []).filter((_, i) => i !== ri))} />
          ))}
          {!(stage.requests || []).length && (
            <div style={{ border: "1px dashed #1a2535", borderRadius: 6, padding: "16px", textAlign: "center", color: "#1e2a3a", fontSize: 11, fontFamily: "monospace" }}>
              нет запросов — добавьте хотя бы один
            </div>
          )}
        </div>

        <div>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 8 }}>
            <span style={{ fontSize: 10, color: "#3d5068", fontFamily: "monospace", letterSpacing: "0.1em" }}>
              CHAOS ENGINEERING
              <span style={{ color: "#1e2a3a", marginLeft: 8 }}>сбои выполняются параллельно нагрузке</span>
            </span>
            <button onClick={() => onChange("chaos_events", [...(stage.chaos_events || []), { type: "network_delay", target_container: "", start_delay: 0, duration: 10, delay: "100ms", jitter: "10ms" }])}
              style={{ ...S.btn(), color: "#f87171", borderColor: "#f8717130" }}>⚡ + сбой</button>
          </div>
          {(stage.chaos_events || []).map((ev, ci) => (
            <ChaosEventForm key={ci} event={ev}
              containers={containers}
              onChange={(f, v) => updateChaos(ci, f, v)}
              onRemove={() => onChange("chaos_events", (stage.chaos_events || []).filter((_, i) => i !== ci))} />
          ))}
          {!(stage.chaos_events || []).length && (
            <div style={{ border: "1px dashed #1a252030", borderRadius: 6, padding: "10px 14px", color: "#1e2a3a", fontSize: 10, fontFamily: "monospace" }}>
              сбоев нет — штатные условия
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export function StopConditionsForm({ sc, onChange, containers }) {
  const iStyle = { ...S.input, fontSize: 11 };
  const enabled = sc.stop_conditions !== null && sc.stop_conditions !== undefined;

  const toggle = () => {
    if (enabled) {
      onChange("stop_conditions", null);
    } else {
      onChange("stop_conditions", {});
    }
  };

  const upCond = (field, value) => {
    onChange("stop_conditions", { ...sc.stop_conditions, [field]: value });
  };

  const clearCond = (field) => {
    const next = { ...sc.stop_conditions };
    delete next[field];
    onChange("stop_conditions", next);
  };

  const cond = sc.stop_conditions || {};

  return (
    <div style={{ border: "1px solid #1a2535", borderRadius: 8, marginBottom: 14, overflow: "hidden" }}>
      <div
        style={{
          background: enabled ? "rgba(248,113,113,0.04)" : "transparent",
          borderBottom: enabled ? "1px solid rgba(248,113,113,0.15)" : "1px solid #1a2535",
          padding: "10px 16px",
          display: "flex", alignItems: "center", justifyContent: "space-between", cursor: "pointer",
        }}
        onClick={toggle}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span style={{ fontSize: 10, color: enabled ? "#f87171" : "#3d5068", fontFamily: "monospace", fontWeight: 700, letterSpacing: "0.1em" }}>
            КРИТЕРИИ ОСТАНОВКИ
          </span>
          {enabled && <Pill color="#f87171" label="включено" />}
          {!enabled && <span style={{ fontSize: 10, color: "#1e2a3a", fontFamily: "monospace" }}>тест всегда идёт до конца</span>}
        </div>
        <span style={{ fontSize: 11, color: enabled ? "#f87171" : "#3d5068" }}>{enabled ? "▲ свернуть" : "▼ настроить"}</span>
      </div>

      {enabled && (
        <div style={{ padding: "14px 16px" }}>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10, marginBottom: 12 }}>
            {/* Error rate */}
            <div>
              <label style={S.label}>ПОРОГ ОШИБОК (%, 1–100)</label>
              <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                <input
                  style={{ ...iStyle, flex: 1 }}
                  type="number" min={1} max={100}
                  value={cond.error_rate_percent ?? ""}
                  placeholder="не задано"
                  onChange={(e) => {
                    const v = e.target.value;
                    v === "" ? clearCond("error_rate_percent") : upCond("error_rate_percent", +v);
                  }}
                />
                {cond.error_rate_percent != null && (
                  <button onClick={() => clearCond("error_rate_percent")} style={{ ...S.btn("danger"), padding: "4px 8px", fontSize: 10 }}>✕</button>
                )}
              </div>
              <span style={{ fontSize: 9, color: "#3d5068", fontFamily: "monospace" }}>остановить если % ошибок превысит порог</span>
            </div>

            {/* Max response time */}
            <div>
              <label style={S.label}>МАКС. ВРЕМЯ ОТВЕТА (сек)</label>
              <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                <input
                  style={{ ...iStyle, flex: 1 }}
                  type="number" min={0.001} step={0.1}
                  value={cond.max_response_time_sec ?? ""}
                  placeholder="не задано"
                  onChange={(e) => {
                    const v = e.target.value;
                    v === "" ? clearCond("max_response_time_sec") : upCond("max_response_time_sec", +v);
                  }}
                />
                {cond.max_response_time_sec != null && (
                  <button onClick={() => clearCond("max_response_time_sec")} style={{ ...S.btn("danger"), padding: "4px 8px", fontSize: 10 }}>✕</button>
                )}
              </div>
              <span style={{ fontSize: 9, color: "#3d5068", fontFamily: "monospace" }}>остановить если latency превысит порог</span>
            </div>
          </div>

          {/* Container + CPU/RAM row */}
          <div style={{ borderTop: "1px solid #1a2535", paddingTop: 12 }}>
            <div style={{ marginBottom: 8 }}>
              <label style={S.label}>КОНТЕЙНЕР ДЛЯ CPU/RAM МОНИТОРИНГА</label>
              {containers && containers.length > 0 ? (
                <select
                  style={iStyle}
                  value={cond.target_container || ""}
                  onChange={(e) => {
                    const v = e.target.value;
                    v === "" ? clearCond("target_container") : upCond("target_container", v);
                  }}
                >
                  <option value="">— не задано (CPU/RAM не мониторятся) —</option>
                  {containers.map((ct) => (
                    <option key={ct.ID} value={ct.ID}>
                      {ct.Name} · {ct.Image} · {ct.Status}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  style={iStyle}
                  value={cond.target_container || ""}
                  placeholder="ID контейнера (оставьте пустым если не нужен CPU/RAM)"
                  onChange={(e) => {
                    const v = e.target.value;
                    v === "" ? clearCond("target_container") : upCond("target_container", v);
                  }}
                />
              )}
            </div>

            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10, opacity: cond.target_container ? 1 : 0.35 }}>
              <div>
                <label style={S.label}>МАКС. CPU КОНТЕЙНЕРА (%, 1–95)</label>
                <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                  <input
                    style={{ ...iStyle, flex: 1 }}
                    type="number" min={1} max={95}
                    value={cond.max_cpu_percent ?? ""}
                    placeholder="не задано"
                    disabled={!cond.target_container}
                    onChange={(e) => {
                      const v = e.target.value;
                      v === "" ? clearCond("max_cpu_percent") : upCond("max_cpu_percent", +v);
                    }}
                  />
                  {cond.max_cpu_percent != null && (
                    <button onClick={() => clearCond("max_cpu_percent")} style={{ ...S.btn("danger"), padding: "4px 8px", fontSize: 10 }}>✕</button>
                  )}
                </div>
              </div>
              <div>
                <label style={S.label}>МАКС. RAM КОНТЕЙНЕРА (%, 1–95)</label>
                <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
                  <input
                    style={{ ...iStyle, flex: 1 }}
                    type="number" min={1} max={95}
                    value={cond.max_ram_percent ?? ""}
                    placeholder="не задано"
                    disabled={!cond.target_container}
                    onChange={(e) => {
                      const v = e.target.value;
                      v === "" ? clearCond("max_ram_percent") : upCond("max_ram_percent", +v);
                    }}
                  />
                  {cond.max_ram_percent != null && (
                    <button onClick={() => clearCond("max_ram_percent")} style={{ ...S.btn("danger"), padding: "4px 8px", fontSize: 10 }}>✕</button>
                  )}
                </div>
              </div>
            </div>
            {!cond.target_container && (cond.max_cpu_percent != null || cond.max_ram_percent != null) && (
              <div style={{ fontSize: 10, color: "#f87171", fontFamily: "monospace", marginTop: 6 }}>
                ⚠ для CPU/RAM порогов нужен контейнер
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export function ScenarioForm({ initial, onSave, onCancel, containers }) {
  const [sc, setSc] = useState(() => initial ? JSON.parse(JSON.stringify(initial)) : JSON.parse(JSON.stringify(EMPTY_SCENARIO)));
  const [err, setErr] = useState("");

  const upField = (f, v) => setSc((s) => ({ ...s, [f]: v }));

  const upStage = (idx, field, val) => setSc((s) => {
    const stages = [...s.stages];
    stages[idx] = { ...stages[idx], [field]: val };
    return { ...s, stages };
  });

  const computedTotal = sc.stages.reduce((sum, st) => sum + (st.duration || 0), 0);

  const handleSave = () => {
    if (!sc.name.trim())     return setErr("Укажите имя сценария");
    if (!sc.base_url.trim()) return setErr("Укажите base URL");
    if (!sc.stages.length)   return setErr("Нужен хотя бы один этап");
    for (let i = 0; i < sc.stages.length; i++) {
      const st = sc.stages[i];
      if (st.type === "spike" && (st.baseline_users ?? 0) < 0) return setErr(`Этап ${i + 1} (spike): baseline_users не может быть отрицательным`);
      if (!st.requests?.length) return setErr(`Этап ${i + 1}: добавьте хотя бы один запрос`);
      const wSum = st.requests.reduce((s, r) => s + (r.probability ?? 0), 0);
      if (wSum <= 0) return setErr(`Этап ${i + 1}: суммарная вероятность запросов должна быть > 0`);
      for (let j = 0; j < (st.chaos_events || []).length; j++) {
        const ch = st.chaos_events[j];
        if (!ch.target_container) return setErr(`Этап ${i + 1}, сбой ${j + 1}: укажите ID контейнера`);
        if (ch.start_delay + ch.duration > st.duration) return setErr(`Этап ${i + 1}, сбой ${j + 1}: выходит за пределы этапа (${ch.start_delay}+${ch.duration} > ${st.duration})`);
        if (ch.type === "network_delay" && !ch.delay) return setErr(`Этап ${i + 1}, сбой ${j + 1}: network_delay требует поле delay`);
      }
    }
    if (sc.stop_conditions) {
      const cond = sc.stop_conditions;
      if (cond.error_rate_percent != null && (cond.error_rate_percent < 1 || cond.error_rate_percent > 100))
        return setErr("Критерии остановки: error_rate_percent должен быть от 1 до 100");
      if (cond.max_response_time_sec != null && cond.max_response_time_sec <= 0)
        return setErr("Критерии остановки: max_response_time_sec должен быть положительным");
      if (cond.max_cpu_percent != null && (cond.max_cpu_percent < 1 || cond.max_cpu_percent > 95))
        return setErr("Критерии остановки: max_cpu_percent должен быть от 1 до 95");
      if (cond.max_ram_percent != null && (cond.max_ram_percent < 1 || cond.max_ram_percent > 95))
        return setErr("Критерии остановки: max_ram_percent должен быть от 1 до 95");
      if ((cond.max_cpu_percent != null || cond.max_ram_percent != null) && !cond.target_container)
        return setErr("Критерии остановки: укажите контейнер для CPU/RAM мониторинга");
    }
    setErr("");
    onSave({ ...sc, total_duration: sc.total_duration || computedTotal });
  };

  return (
    <div>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 160px", gap: 12, marginBottom: 20 }}>
        <div>
          <label style={S.label}>НАЗВАНИЕ СЦЕНАРИЯ</label>
          <input style={S.input} value={sc.name} onChange={(e) => upField("name", e.target.value)} placeholder="Checkout Load Test" />
        </div>
        <div>
          <label style={S.label}>BASE URL</label>
          <input style={S.input} value={sc.base_url} onChange={(e) => upField("base_url", e.target.value)} placeholder="https://api.example.com" />
        </div>
        <div>
          <label style={S.label}>TOTAL DURATION</label>
          <input style={S.input} type="number" min={0} value={sc.total_duration}
            onChange={(e) => upField("total_duration", +e.target.value)} placeholder={String(computedTotal)} />
        </div>
      </div>

      {sc.stages.length > 0 && <StageTimeline stages={sc.stages} />}

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
        <span style={{ fontSize: 10, color: "#3d5068", fontFamily: "monospace", letterSpacing: "0.1em" }}>
          ЭТАПЫ · {computedTotal}с итого
        </span>
        <button onClick={() => setSc((s) => ({ ...s, stages: [...s.stages, { id: s.stages.length + 1, type: "steady", duration: 60, target_users: 10, requests: [], chaos_events: [] }] }))} style={S.btn()}>+ этап</button>
      </div>

      {sc.stages.map((stage, si) => (
        <StageEditor key={si} stage={stage} idx={si}
          containers={containers}
          onChange={(f, v) => upStage(si, f, v)}
          onRemove={() => setSc((s) => ({ ...s, stages: s.stages.filter((_, i) => i !== si) }))} />
      ))}

      <StopConditionsForm sc={sc} onChange={upField} containers={containers} />

      {err && (
        <div style={{ color: "#f87171", fontSize: 11, fontFamily: "monospace", marginBottom: 12, padding: "8px 12px", background: "rgba(248,113,113,0.06)", borderRadius: 4, border: "1px solid #f8717130" }}>
          ⚠ {err}
        </div>
      )}

      <div style={{ display: "flex", gap: 10 }}>
        <button onClick={handleSave} style={S.btn("primary")}>{initial ? "СОХРАНИТЬ" : "СОЗДАТЬ СЦЕНАРИЙ"}</button>
        <button onClick={onCancel} style={S.btn()}>ОТМЕНА</button>
      </div>
    </div>
  );
}