import { useState } from "react";
import { api } from "../../api";
import { fmt } from "../../utils/formatters";
import { S } from "../../styles/theme";
import { Pill, Badge, Flash } from "../ui";
import { RunDetail } from "../runs/RunDetail";

export function RunsTab({ runs, setRuns, scenarios, loading, onRefresh }) {
  const [detail, setDetail] = useState(null);
  const [msg, setMsg] = useState("");
  const [stoppingId, setStoppingId] = useState(null);

  const flash = (m) => { setMsg(m); setTimeout(()=>setMsg(""), 3000); };

  const handleStop = async (runId) => {
    setStoppingId(runId);
    try {
      await api(`/runs/${runId}/stop`, { method: "POST" });
      
      // ОПТИМИСТИЧНОЕ ОБНОВЛЕНИЕ
      // Находим тест в массиве и меняем ему статус локально
      if (setRuns) {
        setRuns(prevRuns => prevRuns.map(item => {
          const currentRun = item.run || item;
          if (currentRun.id === runId) {
            // Сохраняем структуру (объект может быть {run:...} или просто сам объект)
            return item.run 
              ? { ...item, run: { ...item.run, status: "stopped" } }
              : { ...item, status: "stopped" };
          }
          return item;
        }));
      }

      flash("✓ Остановлен");
      onRefresh(); 
    } catch (e) {
      flash("✗ " + e.message);
    } finally {
      setStoppingId(null);
    }
  };

  const scMap = Object.fromEntries((scenarios||[]).map(s=>[s.id,s]));

  if (detail) {
    const run = detail.run||detail;
    const sc  = detail.scenario||scMap[run?.scenario_id];
    return <RunDetail run={run} scenario={sc} onStop={handleStop} onClose={()=>{setDetail(null);onRefresh();}} />;
  }

  const activeCount = runs.filter(r=>(r.run||r)?.status==="running").length;

  return (
    <div>
      <div style={{display:"flex",justifyContent:"space-between",alignItems:"center",marginBottom:20}}>
        <div style={{display:"flex",alignItems:"center",gap:12}}>
          <span style={{fontSize:11,color:"#3d5068",fontFamily:"monospace"}}>{runs.length} запусков</span>
          {activeCount>0 && <Pill color="#22d3ee" label={`${activeCount} активных`} />}
        </div>
        <button onClick={() => onRefresh()} style={S.btn()} disabled={loading}>
          {loading ? "..." : "↻ обновить"}
        </button>
      </div>
      
      <Flash msg={msg} />

      {/* Изменено условие: лоадер только если список совсем пуст */}
      {loading && runs.length === 0 && (
        <div style={{color:"#3d5068",fontFamily:"monospace",fontSize:11,textAlign:"center",padding:40}}>загрузка…</div>
      )}

      {runs.length === 0 && !loading && (
        <div style={{color:"#1a2535",fontFamily:"monospace",fontSize:13,textAlign:"center",padding:60}}>нет запусков</div>
      )}

      {runs.length > 0 && (
        <div style={{
          border: "1px solid #1a2535", 
          borderRadius: 8, 
          overflow: "hidden",
          opacity: (loading && runs.length > 0) ? 0.7 : 1, // Мягкое затемнение при фоновом обновлении
          transition: "opacity 0.2s ease"
        }}>
          <table style={{width:"100%", borderCollapse:"collapse", fontFamily:"monospace", fontSize:13}}>
            <thead>
              <tr style={{background:"#0a0e17", color:"#3d5068"}}>
                {["СЦЕНАРИЙ","RUN ID","СТАТУС","ЭТАПОВ","СТАРТ","ФИНИШ",""].map(h=>(
                  <th key={h} style={{padding:"9px 14px", textAlign:"left", fontSize:11, letterSpacing:"0.08em"}}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {runs.map((item, i)=>{
                const run = item.run || item;
                const sc = item.scenario || scMap[run?.scenario_id];
                const hasChaos = sc?.stages?.some(st => st.chaos_events?.length > 0);
                const isStopping = stoppingId === run.id;

                return (
                  <tr 
                    key={run?.id || i} 
                    style={{
                      borderTop: "1px solid #1a2535", 
                      background: i % 2 === 0 ? "transparent" : "#0a0e1715", 
                      cursor: isStopping ? "default" : "pointer"
                    }} 
                    onClick={() => !isStopping && setDetail(item)}
                  >
                    <td style={{padding:"10px 14px", color:"#e2e8f0"}}>
                      <div style={{display:"flex", alignItems:"center", gap:6}}>
                        {sc?.name || run?.scenario_id?.slice(0,8) || "—"}
                        {hasChaos && <Pill color="#f87171" label="⚡" />}
                      </div>
                    </td>
                    <td style={{padding:"10px 14px", color:"#3d5068"}}>{run?.id?.slice(0,8)}…</td>
                    <td style={{padding:"10px 14px"}}><Badge status={run?.status} /></td>
                    <td style={{padding:"10px 14px", color:"#4a5568"}}>{sc?.stages?.length ?? "—"}</td>
                    <td style={{padding:"10px 14px", color:"#4a5568"}}>{fmt.time(run?.started_at)}</td>
                    <td style={{padding:"10px 14px", color:"#4a5568"}}>{fmt.time(run?.finished_at)}</td>
                    <td style={{padding:"10px 14px"}} onClick={e => e.stopPropagation()}>
                      {run?.status === "running" && (
                        <button 
                          onClick={() => handleStop(run.id)} 
                          disabled={!!stoppingId}
                          style={{
                            ...S.btn("danger"), 
                            padding:"3px 10px",
                            minWidth: "65px",
                            opacity: isStopping ? 0.5 : 1
                          }}
                        >
                          {isStopping ? "..." : "■ стоп"}
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}