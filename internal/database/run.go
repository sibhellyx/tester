package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sibhellyx/tester/internal/models"
	"github.com/sibhellyx/tester/internal/service"
)

// TestRunRepository реализует хранение и чтение запусков тестов.
type TestRunRepository struct {
	logger *slog.Logger
	db     *sql.DB
}

// NewTestRunRepository - конструктор.
func NewTestRunRepository(logger *slog.Logger, db *sql.DB) *TestRunRepository {
	return &TestRunRepository{
		logger: logger,
		db:     db,
	}
}

// Create - создаёт новую запись о запуске теста.
func (r *TestRunRepository) Create(ctx context.Context, run models.TestRun) error {
	const q = `
		INSERT INTO test_runs (id, scenario_id, status, started_at, finished_at, error)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, q,
		run.ID,
		run.ScenarioID,
		run.Status,
		run.StartedAt,
		run.FinishedAt,
		run.Error,
	)
	if err != nil {
		return fmt.Errorf("create test_run: %w", err)
	}
	return nil
}

// Get - возвращает запуск по ID. Возвращает nil, nil если не найден.
func (r *TestRunRepository) Get(ctx context.Context, id string) (*models.TestRun, error) {
	const q = `
		SELECT id, scenario_id, status, started_at, finished_at, error
		FROM test_runs
		WHERE id = $1`

	var run models.TestRun
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&run.ID,
		&run.ScenarioID,
		&run.Status,
		&run.StartedAt,
		&run.FinishedAt,
		&run.Error,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get test_run: %w", err)
	}
	return &run, nil
}

// UpdateStatus - обновляет статус запуска и сопутствующие поля.
func (r *TestRunRepository) UpdateStatus(ctx context.Context, id string, status models.TestRunStatus, finishedAt *time.Time, errMsg string) error {
	const q = `
		UPDATE test_runs
		SET status      = $1,
		    finished_at = $2,
		    error       = $3
		WHERE id = $4`

	result, err := r.db.ExecContext(ctx, q, status, finishedAt, errMsg, id)
	if err != nil {
		return fmt.Errorf("update test_run status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return service.ErrRunNotFound
	}
	return nil
}

// ListByScenario - возвращает все запуски для конкретного сценария, от новых к старым.
func (r *TestRunRepository) ListByScenario(ctx context.Context, scenarioID string) ([]models.TestRun, error) {
	const q = `
		SELECT id, scenario_id, status, started_at, finished_at, error
		FROM test_runs
		WHERE scenario_id = $1
		ORDER BY started_at DESC NULLS LAST`

	rows, err := r.db.QueryContext(ctx, q, scenarioID)
	if err != nil {
		return nil, fmt.Errorf("list test_runs: %w", err)
	}
	defer rows.Close()

	var runs []models.TestRun
	for rows.Next() {
		var run models.TestRun
		if err = rows.Scan(
			&run.ID,
			&run.ScenarioID,
			&run.Status,
			&run.StartedAt,
			&run.FinishedAt,
			&run.Error,
		); err != nil {
			return nil, fmt.Errorf("scan test_run: %w", err)
		}
		runs = append(runs, run)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("test_runs rows: %w", err)
	}
	return runs, nil
}

// SaveResults - пакетно сохраняет результаты вызовов (CallResult) для запуска.
// Используем COPY-подобный подход через транзакцию для производительности.
func (r *TestRunRepository) SaveResults(ctx context.Context, runID string, results []models.CallResult) error {
	if len(results) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const q = `
		INSERT INTO call_results
		    (run_id, request_name, timestamp, duration_ms, status, error, bytes_out, bytes_in)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare call_results stmt: %w", err)
	}
	defer stmt.Close()

	for _, res := range results {
		_, err = stmt.ExecContext(ctx,
			runID,
			res.RequestName,
			res.Timestamp,
			res.Duration.Milliseconds(),
			res.Status,
			res.Error,
			res.BytesOut,
			res.BytesIn,
		)
		if err != nil {
			return fmt.Errorf("insert call_result: %w", err)
		}
	}

	return tx.Commit()
}

// GetSummary - возвращает агрегированную статистику по запуску из БД.
func (r *TestRunRepository) GetSummary(ctx context.Context, runID string) (*models.TestRunSummary, error) {
	const q = `
		SELECT
		    COUNT(*)                                                AS total_calls,
		    COUNT(*) FILTER (WHERE status >= 200 AND status < 400) AS success_calls,
		    COUNT(*) FILTER (WHERE status = 0 OR status >= 400)    AS failed_calls,
		    COALESCE(AVG(duration_ms), 0)                          AS avg_latency_ms,
		    COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms), 0) AS p95,
		    COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms), 0) AS p99,
		    COALESCE(MAX(duration_ms), 0)                          AS max_latency_ms,
		    COALESCE(SUM(bytes_in), 0)                             AS total_bytes_in,
		    COALESCE(SUM(bytes_out), 0)                            AS total_bytes_out
		FROM call_results
		WHERE run_id = $1`

	var s models.TestRunSummary
	s.RunID = runID

	err := r.db.QueryRowContext(ctx, q, runID).Scan(
		&s.TotalCalls,
		&s.SuccessCalls,
		&s.FailedCalls,
		&s.AvgLatencyMs,
		&s.P95LatencyMs,
		&s.P99LatencyMs,
		&s.MaxLatencyMs,
		&s.TotalBytesIn,
		&s.TotalBytesOut,
	)
	if err != nil {
		return nil, fmt.Errorf("get summary: %w", err)
	}
	return &s, nil
}
