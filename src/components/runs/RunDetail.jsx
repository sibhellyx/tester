import { useState, useEffect } from "react";
import { api, BASE } from "../../api";
import { fmt } from "../../utils/formatters";
import { S } from "../../styles/theme";
import { Badge, StatCard } from "../ui";
import { StageTimeline } from "../scenarios";
import { ChartPanel } from "../charts/ChartPanel";
import { STAGE_COLORS } from "../../utils/constants";

export function RunDetail({ run, scenario, onStop, onClose }) {
  const [report, setReport] = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (["finished","stopped","failed"].includes(run.status)) {
      setLoading(true);
      api(`/runs/${run.id}/report`).then(setReport).catch(()=>null).finally(()=>setLoading(false));
    }
  }, [run.id, run.status]);

  const m = report?.summary;

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 20 }}>
        <div style={{ fontFamily: "monospace", fontSize: 11 }}>
          <span style={{ color: "#3d5068" }}>ЗАПУСК / </span>
          <span style={{ color: "#e2e8f0" }}>{run.id}</span>
        </div>
        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <Badge status={run.status} />
          {run.status === "running" && <button onClick={() => onStop(run.id)} style={{ ...S.btn("danger"), fontWeight: 700 }}>■ СТОП</button>}
          <button onClick={onClose} style={S.btn()}>← назад</button>
        </div>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8, marginBottom: 16, fontSize: 11, fontFamily: "monospace" }}>
        {[["Сценарий", scenario?.name||"—"],["Base URL", scenario?.base_url||"—"],["Старт", fmt.time(run.started_at)],["Финиш", fmt.time(run.finished_at)]].map(([k,v])=>(
          <div key={k} style={{ ...S.card, padding: "8px 14px" }}><span style={{ color: "#3d5068" }}>{k}: </span><span style={{ color: "#94a3b8" }}>{v}</span></div>
        ))}
      </div>

      {scenario?.stages?.length > 0 && <StageTimeline stages={scenario.stages} />}

      {run.error && <div style={{ background: "rgba(248,113,113,0.06)", border: "1px solid #f8717130", borderRadius: 5, padding: "10px 14px", marginBottom: 16, color: "#f87171", fontSize: 11, fontFamily: "monospace" }}>⚠ {run.error}</div>}
      {loading && <div style={{ color: "#3d5068", fontFamily: "monospace", fontSize: 11, textAlign: "center", padding: 40 }}>загрузка отчёта…</div>}

      {m && (
        <>
          <div style={{ ...S.label, marginBottom: 12 }}>СВОДКА</div>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginBottom: 24 }}>
            <StatCard label="ИТОГО"   value={m.total_requests ?? "—"} />
            <StatCard label="OK"      value={m.success_count ?? "—"}  accent="#4ade80" />
            <StatCard label="ОШИБОК"  value={m.error_count ?? "—"}    accent="#f87171" />
            <StatCard label="ERROR %"  value={fmt.pct(m.error_rate)}   accent={m.error_rate > 0.05 ? "#f87171" : "#4ade80"} />
            <StatCard label="AVG"     value={fmt.ms(m.avg_latency_ms)} />
            <StatCard label="P50"     value={fmt.ms(m.p50_ms)} />
            <StatCard label="P90"     value={fmt.ms(m.p90_ms)} />
            <StatCard label="P95"     value={fmt.ms(m.p95_ms)} accent="#f59e0b" />
            <StatCard label="P99"     value={fmt.ms(m.p99_ms)} accent="#f87171" />
            <StatCard label="MAX"     value={fmt.ms(m.max_latency_ms)} />
            <StatCard label="RPS"     value={fmt.rps(m.rps)}           accent="#22d3ee" />
            <StatCard label="IN"      value={fmt.bytes(m.total_bytes_in)} />
            <StatCard label="OUT"     value={fmt.bytes(m.total_bytes_out)} />
          </div>
        </>
      )}

      {report?.charts?.length > 0 && (
        <>
          <div style={{ ...S.label, marginBottom: 12 }}>ГРАФИКИ</div>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10, marginBottom: 24 }}>
            {report.charts.map((ch, i) => <ChartPanel key={i} chart={ch} />)}
          </div>
        </>
      )}

      {report?.per_stage && Object.keys(report.per_stage).length > 0 && (
        <>
          <div style={{ ...S.label, marginBottom: 10 }}>ПО ЭТАПАМ</div>
          <div style={{ border: "1px solid #1a2535", borderRadius: 8, overflow: "hidden", marginBottom: 24 }}>
            <table style={{ width: "100%", borderCollapse: "collapse", fontFamily: "monospace", fontSize: 11 }}>
              <thead>
                <tr style={{ background: "#0a0e17", color: "#3d5068" }}>
                  {["ЭТАП","ТИП","VU","ИТОГО","OK","ERROR%","AVG","P50","P95","P99","MAX","RPS"].map((h) => (
                    <th key={h} style={{ padding: "8px 12px", textAlign: "left", fontSize: 10, letterSpacing: "0.06em", fontWeight: 600 }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {Object.entries(report.per_stage)
                  .sort(([a], [b]) => +a - +b)
                  .map(([stageId, mx], i) => {
                    const col = STAGE_COLORS[mx.type] || "#64748b";
                    return (
                      <tr key={stageId} style={{ borderTop: "1px solid #1a2535", background: i % 2 === 0 ? "transparent" : "#0a0e1715" }}>
                        <td style={{ padding: "8px 12px", color: "#64748b" }}>#{stageId}</td>
                        <td style={{ padding: "8px 12px", color: col, fontWeight: 700 }}>{mx.type ?? "—"}</td>
                        <td style={{ padding: "8px 12px", color: "#64748b" }}>{mx.target_users ?? "—"}</td>
                        <td style={{ padding: "8px 12px", color: "#64748b" }}>{mx.total_requests}</td>
                        <td style={{ padding: "8px 12px", color: "#4ade80" }}>{mx.success_count}</td>
                        <td style={{ padding: "8px 12px", color: mx.error_rate > 0.05 ? "#f87171" : "#4a5568" }}>{fmt.pct(mx.error_rate)}</td>
                        <td style={{ padding: "8px 12px", color: "#94a3b8" }}>{fmt.ms(mx.avg_latency_ms)}</td>
                        <td style={{ padding: "8px 12px", color: "#64748b" }}>{fmt.ms(mx.p50_ms)}</td>
                        <td style={{ padding: "8px 12px", color: "#f59e0b" }}>{fmt.ms(mx.p95_ms)}</td>
                        <td style={{ padding: "8px 12px", color: "#f87171" }}>{fmt.ms(mx.p99_ms)}</td>
                        <td style={{ padding: "8px 12px", color: "#94a3b8" }}>{fmt.ms(mx.max_latency_ms)}</td>
                        <td style={{ padding: "8px 12px", color: "#22d3ee" }}>{fmt.rps(mx.rps)}</td>
                      </tr>
                    );
                  })}
              </tbody>
            </table>
          </div>
        </>
      )}

      {report?.per_request && Object.keys(report.per_request).length > 0 && (
        <>
          <div style={{ ...S.label, marginBottom: 10 }}>ПО ЗАПРОСАМ</div>
          <div style={{ border: "1px solid #1a2535", borderRadius: 8, overflow: "hidden" }}>
            <table style={{ width: "100%", borderCollapse: "collapse", fontFamily: "monospace", fontSize: 11 }}>
              <thead>
                <tr style={{ background: "#0a0e17", color: "#3d5068" }}>
                  {["ЗАПРОС","ИТОГО","OK","ERROR%","AVG","P50","P95","P99","MAX","RPS"].map((h) => (
                    <th key={h} style={{ padding: "8px 12px", textAlign: "left", fontSize: 10, letterSpacing: "0.06em", fontWeight: 600 }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {Object.entries(report.per_request).map(([name, mx], i) => (
                  <tr key={name} style={{ borderTop: "1px solid #1a2535", background: i%2===0?"transparent":"#0a0e1715" }}>
                    <td style={{ padding: "8px 12px", color: "#e2e8f0" }}>{name}</td>
                    <td style={{ padding: "8px 12px", color: "#64748b" }}>{mx.total_requests}</td>
                    <td style={{ padding: "8px 12px", color: "#4ade80" }}>{mx.success_count}</td>
                    <td style={{ padding: "8px 12px", color: mx.error_rate>0.05?"#f87171":"#4a5568" }}>{fmt.pct(mx.error_rate)}</td>
                    <td style={{ padding: "8px 12px", color: "#94a3b8" }}>{fmt.ms(mx.avg_latency_ms)}</td>
                    <td style={{ padding: "8px 12px", color: "#64748b" }}>{fmt.ms(mx.p50_ms)}</td>
                    <td style={{ padding: "8px 12px", color: "#f59e0b" }}>{fmt.ms(mx.p95_ms)}</td>
                    <td style={{ padding: "8px 12px", color: "#f87171" }}>{fmt.ms(mx.p99_ms)}</td>
                    <td style={{ padding: "8px 12px", color: "#94a3b8" }}>{fmt.ms(mx.max_latency_ms)}</td>
                    <td style={{ padding: "8px 12px", color: "#22d3ee" }}>{fmt.rps(mx.rps)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {["finished","stopped"].includes(run.status) && (
        <div style={{ marginTop: 20 }}>
          <a href={`${BASE}/runs/${run.id}/report/download`} style={{ ...S.btn(), display: "inline-block", textDecoration: "none", padding: "7px 16px" }}>
            ↓ скачать CSV
          </a>
        </div>
      )}
    </div>
  );
}