package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// TestResultsRepository читает результаты тестов из БД.
// Запись данных происходит в TestRunRepository.SaveResults.
type TestResultsRepository struct {
	logger *slog.Logger
	db     *sql.DB
}

// NewTestResultsRepository - конструктор.
func NewTestResultsRepository(logger *slog.Logger, db *sql.DB) *TestResultsRepository {
	return &TestResultsRepository{
		logger: logger,
		db:     db,
	}
}

// GetCallResults загружает все CallResult для конкретного запуска.
func (r *TestResultsRepository) GetCallResults(ctx context.Context, runID string) ([]models.CallResult, error) {
	const q = `
		SELECT request_name, timestamp, duration_ms, status, error, bytes_out, bytes_in, stage_id
		FROM call_results
		WHERE run_id = $1
		ORDER BY timestamp ASC`

	rows, err := r.db.QueryContext(ctx, q, runID)
	if err != nil {
		r.logger.Error("GetCallResults failed", slog.String("run_id", runID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("get call results: %w", err)
	}
	defer rows.Close()

	var results []models.CallResult

	for rows.Next() {
		var (
			res        models.CallResult
			durationMs int64
		)
		if err = rows.Scan(
			&res.RequestName,
			&res.Timestamp,
			&durationMs,
			&res.Status,
			&res.Error,
			&res.BytesOut,
			&res.BytesIn,
			&res.StageID,
		); err != nil {
			return nil, fmt.Errorf("scan call result: %w", err)
		}
		res.Duration = time.Duration(durationMs) * time.Millisecond
		results = append(results, res)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("call results rows: %w", err)
	}

	return results, nil
}

// GetRunWithScenario возвращает запуск вместе со сценарием и его этапами.
func (r *TestResultsRepository) GetRunWithScenario(ctx context.Context, runID string) (*models.TestRun, *models.TestScenario, error) {
	const q = `
		SELECT
		    tr.id, tr.scenario_id, tr.status, tr.started_at, tr.finished_at, tr.error,
		    s.id, s.name, s.base_url, s.total_duration
		FROM test_runs tr
		JOIN scenarios s ON s.id = tr.scenario_id
		WHERE tr.id = $1`

	var (
		run      models.TestRun
		scenario models.TestScenario
	)

	err := r.db.QueryRowContext(ctx, q, runID).Scan(
		&run.ID, &run.ScenarioID, &run.Status, &run.StartedAt, &run.FinishedAt, &run.Error,
		&scenario.ID, &scenario.Name, &scenario.BaseURL, &scenario.TotalDuration,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("get run with scenario: %w", err)
	}

	// Загружаем этапы сценария — нужны для обогащения per-stage метрик в отчёте.
	scenario.Stages, err = r.loadStages(ctx, scenario.ID)
	if err != nil {
		return nil, nil, err
	}

	return &run, &scenario, nil
}

// loadStages загружает этапы сценария (без запросов — только метаданные).
func (r *TestResultsRepository) loadStages(ctx context.Context, scenarioID string) ([]models.Stage, error) {
	const q = `
		SELECT id, type, duration, target_users
		FROM stages
		WHERE scenario_id = $1
		ORDER BY id`

	rows, err := r.db.QueryContext(ctx, q, scenarioID)
	if err != nil {
		return nil, fmt.Errorf("load stages: %w", err)
	}
	defer rows.Close()

	var stages []models.Stage
	for rows.Next() {
		var s models.Stage
		if err = rows.Scan(&s.ID, &s.Type, &s.Duration, &s.TargetUsers); err != nil {
			return nil, fmt.Errorf("scan stage: %w", err)
		}
		stages = append(stages, s)
	}
	return stages, rows.Err()
}

// GetListWithStatus возвращает все запуски со сценариями, отсортированные от новых к старым.
func (r *TestResultsRepository) GetListWithStatus(ctx context.Context) ([]models.TestWithStatus, error) {
	const q = `
		SELECT
		    tr.id, tr.scenario_id, tr.status, tr.started_at, tr.finished_at, tr.error,
		    s.id, s.name, s.base_url, s.total_duration
		FROM test_runs tr
		JOIN scenarios s ON s.id = tr.scenario_id
		ORDER BY tr.started_at DESC NULLS LAST`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		r.logger.Error("GetListWithStatus failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("list runs with status: %w", err)
	}
	defer rows.Close()

	var list []models.TestWithStatus

	for rows.Next() {
		var (
			run      models.TestRun
			scenario models.TestScenario
		)
		if err = rows.Scan(
			&run.ID, &run.ScenarioID, &run.Status, &run.StartedAt, &run.FinishedAt, &run.Error,
			&scenario.ID, &scenario.Name, &scenario.BaseURL, &scenario.TotalDuration,
		); err != nil {
			return nil, fmt.Errorf("scan run with status: %w", err)
		}
		list = append(list, models.TestWithStatus{
			Run:      run,
			Scenario: scenario,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list rows: %w", err)
	}

	return list, nil
}
