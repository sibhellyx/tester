import { useState, useEffect, useCallback } from "react";
import { api } from "./api";
import { ScenariosTab } from "./components/views/ScenariosTab";
import { RunsTab } from "./components/views/RunsTab";

export default function App() {
  const [tab, setTab]             = useState("runs");
  const [scenarios, setScenarios] = useState([]);
  const [runs, setRuns]           = useState([]);
  const [loadingSc, setLoadingSc]     = useState(false);
  const [loadingRuns, setLoadingRuns] = useState(false);

  const loadScenarios = useCallback(async () => {
    setLoadingSc(true);
    try { setScenarios(await api("/scenarios")); } catch { setScenarios([]); } finally { setLoadingSc(false); }
  }, []);

  // ДОБАВЛЕНО: параметр isSilent. Если true — лоадер не показывается
  const loadRuns = useCallback(async (isSilent = false) => {
    if (!isSilent) setLoadingRuns(true);
    try { 
      const data = await api("/runs"); 
      setRuns(data);
    } catch (e) { 
      console.error("Failed to load runs:", e);
    } finally { 
      if (!isSilent) setLoadingRuns(false); 
    }
  }, []);

  useEffect(() => { loadScenarios(); loadRuns(); }, [loadScenarios, loadRuns]);

  // ИСПРАВЛЕНО: Интервал теперь делает "тихое" обновление
  useEffect(() => {
    const active = runs.some(r => {
      const status = (r.run || r)?.status;
      return status === "running" || status === "pending";
    });
    
    if (!active) return;
    
    const t = setInterval(() => {
      loadRuns(true); // Обновляем без лоадера
    }, 3000);
    
    return () => clearInterval(t);
  }, [runs, loadRuns]);

  const tabStyle = (t) => ({
    background: "none", border: "none", padding: "12px 20px", cursor: "pointer",
    fontFamily: "monospace", fontSize: 11, letterSpacing: "0.1em", textTransform: "uppercase",
    color: tab===t ? "#22d3ee" : "#3d5068",
    borderBottom: `2px solid ${tab===t ? "#22d3ee" : "transparent"}`,
    transition: "all 0.2s ease"
  });

  return (
    <div style={{ minHeight: "100vh", background: "#070a10", color: "#e2e8f0" }}>
      <div style={{ borderBottom: "1px solid #1a2535", padding: "0 36px", display: "flex", alignItems: "center", height: 52 }}>
        <div style={{ fontSize: 13, fontWeight: 800, fontFamily: "monospace", letterSpacing: "0.2em", color: "#22d3ee", marginRight: 32 }}>▣ TESTER</div>
        <button style={tabStyle("runs")} onClick={() => setTab("runs")}>Запуски</button>
        <button style={tabStyle("scenarios")} onClick={() => setTab("scenarios")}>Сценарии</button>
      </div>
      <div style={{ maxWidth: 1160, margin: "0 auto", padding: "32px 36px" }}>
        {tab==="scenarios" && <ScenariosTab scenarios={scenarios} loading={loadingSc} onRefresh={loadScenarios} />}
        {tab==="runs" && (
          <RunsTab 
            runs={runs} 
            setRuns={setRuns} // Передаем setRuns для оптимистичного обновления
            scenarios={scenarios} 
            loading={loadingRuns} 
            onRefresh={() => loadRuns(true)} 
          />
        )}
      </div>
    </div>
  );
}