import { useState, useEffect } from "react"; // ← ДОБАВЛЕНО: useEffect
import { api } from "../../api";
import { fmt } from "../../utils/formatters";
import { S } from "../../styles/theme";
import { Pill, Flash } from "../ui";
import { ScenarioForm, StageTimeline } from "../scenarios";

export function ScenariosTab({ scenarios, loading, onRefresh }) {
  const [view, setView] = useState("list");
  const [editSc, setEditSc] = useState(null);
  const [msg, setMsg] = useState("");

  const [containers, setContainers] = useState([]);
  useEffect(() => {
    api("/chaos/containers")
      .then(setContainers)
      .catch(() => setContainers([]));
  }, []);

  const [startingId, setStartingId] = useState(null);

  const FLASH_DURATION = 4000;
  const flash = (m) => { setMsg(m); setTimeout(() => setMsg(""), FLASH_DURATION); };

  const handleCreate = async (sc) => {
    try { await api("/scenarios",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(sc)}); flash("✓ Сценарий создан"); onRefresh(); setView("list"); }
    catch(e){ flash("✗ "+e.message); }
  };

  const handleUpdate = async (sc) => {
    try { await api(`/scenarios/${sc.id}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(sc)}); flash("✓ Обновлён"); onRefresh(); setView("list"); }
    catch(e){ flash("✗ "+e.message); }
  };

  const handleDelete = async (id) => {
    if(!confirm("Удалить сценарий?")) return;
    try { await api(`/scenarios/${id}`,{method:"DELETE"}); flash("✓ Удалён"); onRefresh(); }
    catch(e){ flash("✗ "+e.message); }
  };

  const handleStart = async (id) => {
    if (startingId === id) return; // ← ИЗМЕНЕНО
    setStartingId(id);             // ← ИЗМЕНЕНО
    try {
      await api(`/scenarios/${id}/runs`, { method: "POST" });
      flash("✓ Тест запущен");
      onRefresh();
      setTimeout(() => { setStartingId(null); }, FLASH_DURATION); // ← ИЗМЕНЕНО
    } catch (e) {
      flash("✗ " + e.message);
      setStartingId(null); // ← ИЗМЕНЕНО
    }
  };

  if (view==="create") return (
    <div>
      <div style={{fontSize:11,color:"#4a5568",fontFamily:"monospace",marginBottom:20}}>
        <span style={{cursor:"pointer",color:"#3d5068"}} onClick={()=>setView("list")}>сценарии</span>
        <span style={{color:"#22d3ee"}}> / новый</span>
      </div>
      {/* ← ИЗМЕНЕНО: передаём containers */}
      <ScenarioForm containers={containers} onSave={handleCreate} onCancel={()=>setView("list")} />
    </div>
  );

  if (view==="edit"&&editSc) return (
    <div>
      <div style={{fontSize:11,color:"#4a5568",fontFamily:"monospace",marginBottom:20}}>
        <span style={{cursor:"pointer",color:"#3d5068"}} onClick={()=>setView("list")}>сценарии</span>
        <span style={{color:"#22d3ee"}}> / {editSc.name}</span>
      </div>
      {/* ← ИЗМЕНЕНО: передаём containers */}
      <ScenarioForm containers={containers} initial={editSc} onSave={handleUpdate} onCancel={()=>setView("list")} />
    </div>
  );

  return (
    <div>
      <div style={{display:"flex",justifyContent:"space-between",alignItems:"center",marginBottom:20}}>
        <span style={{fontSize:11,color:"#3d5068",fontFamily:"monospace"}}>{scenarios.length} сценариев</span>
        <button onClick={()=>setView("create")} style={S.btn("primary")}>
          + НОВЫЙ СЦЕНАРИЙ
        </button>
      </div>
      <Flash msg={msg} />
      {loading && <div style={{color:"#3d5068",fontFamily:"monospace",fontSize:11,textAlign:"center",padding:40}}>загрузка…</div>}
      {!loading&&scenarios.length===0 && <div style={{color:"#1a2535",fontFamily:"monospace",fontSize:13,textAlign:"center",padding:60}}>нет сценариев</div>}
      {scenarios.map((sc)=>{
        const hasChaos = sc.stages?.some(st=>st.chaos_events?.length>0);
        const isThisStarting = startingId === sc.id; // ← ИЗМЕНЕНО
        return (
          <div key={sc.id} style={{...S.card,display:"flex",alignItems:"flex-start",justifyContent:"space-between",marginBottom:8,opacity:isThisStarting?0.7:1}}>
            <div style={{flex:1,minWidth:0}}>
              <div style={{display:"flex",alignItems:"center",gap:8,marginBottom:4}}>
                <span style={{fontSize:13,color:"#e2e8f0",fontFamily:"monospace",fontWeight:600}}>{sc.name}</span>
                {hasChaos && <Pill color="#f87171" label="⚡ CHAOS" />}
              </div>
              <div style={{fontSize:10,color:"#3d5068",fontFamily:"monospace",marginBottom:8}}>
                {sc.base_url} · {sc.stages?.length||0} этапов · {fmt.dur(sc.total_duration)}
              </div>
              <StageTimeline stages={sc.stages} />
            </div>
            <div style={{display:"flex",gap:8,marginLeft:16,flexShrink:0}}>
              <button onClick={()=>handleStart(sc.id)} disabled={isThisStarting}
                style={{...S.btn(),color:isThisStarting?"#3d5068":"#4ade80",borderColor:isThisStarting?"#1a2535":"#4ade8030",cursor:isThisStarting?"not-allowed":"pointer",minWidth:"80px"}}>
                {isThisStarting ? "WAIT..." : "▶ RUN"}
              </button>
              <button onClick={()=>{setEditSc(sc);setView("edit");}} style={S.btn()}>✎</button>
              <button onClick={()=>handleDelete(sc.id)} style={S.btn("danger")}>✕</button>
            </div>
          </div>
        );
      })}
    </div>
  );
}