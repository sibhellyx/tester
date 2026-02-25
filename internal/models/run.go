package models

import "time"

// TestRunStatus - статус запуска теста.
type TestRunStatus string

const (
	StatusPending  TestRunStatus = "pending"  // Ожидает запуска.
	StatusRunning  TestRunStatus = "running"  // Выполняется прямо сейчас.
	StatusStopped  TestRunStatus = "stopped"  // Остановлен вручную.
	StatusFinished TestRunStatus = "finished" // Завершён штатно.
	StatusFailed   TestRunStatus = "failed"   // Завершён с ошибкой.
)

// TestRun - сущность запуска теста. Хранит метаданные о конкретном прогоне.
type TestRun struct {
	ID         string        `json:"id"`          // UUID запуска.
	ScenarioID string        `json:"scenario_id"` // Ссылка на сценарий.
	Status     TestRunStatus `json:"status"`      // Текущий статус.
	StartedAt  *time.Time    `json:"started_at"`  // Время старта (nil если ещё не стартовал).
	FinishedAt *time.Time    `json:"finished_at"` // Время завершения (nil если ещё идёт).
	Error      string        `json:"error"`       // Текст ошибки (если завершился с ошибкой).
}

// TestWithStatus - сценарий + статус его последнего запуска (удобно для UI).
type TestWithStatus struct {
	Run      TestRun      `json:"run"`
	Scenario TestScenario `json:"scenario"`
}

// TestRunSummary - агрегированные метрики по завершённому запуску.
type TestRunSummary struct {
	RunID         string  `json:"run_id"`
	TotalCalls    int64   `json:"total_calls"`
	SuccessCalls  int64   `json:"success_calls"`
	FailedCalls   int64   `json:"failed_calls"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	P95LatencyMs  float64 `json:"p95_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	MaxLatencyMs  float64 `json:"max_latency_ms"`
	TotalBytesIn  int64   `json:"total_bytes_in"`
	TotalBytesOut int64   `json:"total_bytes_out"`
}
