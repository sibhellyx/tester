package models

import "time"

// Metrics - агрегированные метрики по набору CallResult.
type Metrics struct {
	TotalRequests int           `json:"total_requests"`
	SuccessCount  int           `json:"success_count"`
	ErrorCount    int           `json:"error_count"`
	ErrorRate     float64       `json:"error_rate"` // 0.0 - 1.0
	P50           time.Duration `json:"p50_ms"          swaggertype:"integer"`
	P90           time.Duration `json:"p90_ms"          swaggertype:"integer"`
	P95           time.Duration `json:"p95_ms"          swaggertype:"integer"`
	P99           time.Duration `json:"p99_ms"          swaggertype:"integer"`
	MaxLatency    time.Duration `json:"max_latency_ms"  swaggertype:"integer"`
	MinLatency    time.Duration `json:"min_latency_ms"  swaggertype:"integer"`
	AvgLatency    time.Duration `json:"avg_latency_ms"  swaggertype:"integer"`
	RPS           float64       `json:"rps"`
	TotalBytesIn  int64         `json:"total_bytes_in"`
	TotalBytesOut int64         `json:"total_bytes_out"`
}

// Series - одна линия/серия данных на графике.
type Series struct {
	Name string    `json:"name"`
	Data []float64 `json:"data"`
}

// ChartData - данные для построения одного графика.
type ChartData struct {
	Type   string   `json:"type"` // "line", "bar", "histogram"
	Title  string   `json:"title"`
	Labels []string `json:"labels"` // метки по оси X (время, бакеты)
	Series []Series `json:"series"`
}

// TestReport - полный отчёт по завершённому запуску.
type TestReport struct {
	RunID        string             `json:"run_id"`
	ScenarioName string             `json:"scenario_name"`
	StartTime    time.Time          `json:"start_time"`
	EndTime      time.Time          `json:"end_time"`
	Summary      Metrics            `json:"summary"`      // общая статистика
	PerRequest   map[string]Metrics `json:"per_request"`  // метрики по каждому запросу
	PerStage     map[int]Metrics    `json:"per_stage"`    // метрики по каждому этапу
	Charts       []ChartData        `json:"charts"`
}
