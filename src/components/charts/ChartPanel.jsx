import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from "recharts";
import { S } from "../../styles/theme";
import { CHART_COLORS } from "../../utils/constants";

export function ChartPanel({ chart }) {
  const data = (chart.labels || []).map((label, i) => {
    const point = { label };
    (chart.series || []).forEach((s) => { point[s.name] = s.data?.[i]; });
    return point;
  });
  const Comp = chart.type === "bar" ? BarChart : LineChart;
  const Series = chart.type === "bar" ? Bar : Line;
  
  return (
    <div style={S.card}>
      <div style={{ ...S.label, marginBottom: 12 }}>{chart.title}</div>
      <ResponsiveContainer width="100%" height={180}>
        <Comp data={data} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
          <CartesianGrid strokeDasharray="2 4" stroke="#1a2535" />
          <XAxis dataKey="label" tick={{ fill: "#3d5068", fontSize: 9, fontFamily: "monospace" }} />
          <YAxis tick={{ fill: "#3d5068", fontSize: 9, fontFamily: "monospace" }} />
          <Tooltip contentStyle={{ background: "#0b0f18", border: "1px solid #1e2a3a", borderRadius: 4, fontFamily: "monospace", fontSize: 11 }} labelStyle={{ color: "#64748b" }} />
          <Legend wrapperStyle={{ fontSize: 10, fontFamily: "monospace", color: "#4a5568" }} />
          {(chart.series || []).map((s, i) => (
            <Series key={s.name} type="monotone" dataKey={s.name}
              stroke={CHART_COLORS[i % CHART_COLORS.length]}
              fill={CHART_COLORS[i % CHART_COLORS.length]}
              dot={false} strokeWidth={2} />
          ))}
        </Comp>
      </ResponsiveContainer>
    </div>
  );
}