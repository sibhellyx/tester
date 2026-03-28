package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/lib/pq"
	"github.com/sibhellyx/tester/internal/models"
	"github.com/sibhellyx/tester/internal/service"
)

// ScenarioRepository - универсальная структура для доступа к базе данных.
type ScenarioRepository struct {
	logger *slog.Logger
	db     *sql.DB
}

// NewScenarioRepository - функция для создания структуры доступа до repository.
func NewScenarioRepository(logger *slog.Logger, db *sql.DB) *ScenarioRepository {
	return &ScenarioRepository{
		logger: logger,
		db:     db,
	}
}

// Create - создает новую запись о сценарии в базе данных.
func (r *ScenarioRepository) Create(ctx context.Context, s models.TestScenario) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const scenarioQuery = `
		INSERT INTO scenarios (id, name, base_url, total_duration)
		VALUES ($1, $2, $3, $4)`

	if _, err = tx.ExecContext(ctx, scenarioQuery,
		s.ID, s.Name, s.BaseURL, s.TotalDuration,
	); err != nil {
		return fmt.Errorf("insert scenario: %w", err)
	}

	if err = insertStages(ctx, tx, s.ID, s.Stages); err != nil {
		return err
	}

	if err = insertStopConditions(ctx, tx, s.ID, s.StopConditions); err != nil {
		return err
	}

	return tx.Commit()
}

// Get - возвращает сценриай и (nil, nil) если сценарий не найден — сервис интерпретирует это как ErrScenarioNotFound.
func (r *ScenarioRepository) Get(ctx context.Context, id string) (*models.TestScenario, error) {
	const query = `
		SELECT id, name, base_url, total_duration
		FROM scenarios
		WHERE id = $1`

	var s models.TestScenario
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&s.ID, &s.Name, &s.BaseURL, &s.TotalDuration)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("Get scenario failed", slog.String("id", id), slog.String("error", err.Error()))
		return nil, fmt.Errorf("get scenario: %w", err)
	}

	stages, err := r.loadStages(ctx, id)
	if err != nil {
		return nil, err
	}
	s.Stages = stages

	s.StopConditions, err = r.loadStopConditions(ctx, s.ID)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// List - функция возвращает список созданных сценариев.
func (r *ScenarioRepository) List(ctx context.Context) ([]models.TestScenario, error) {
	const query = `
		SELECT id, name, base_url, total_duration
		FROM scenarios
		ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("List scenarios failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("list scenarios: %w", err)
	}
	defer rows.Close()

	var scenarios []models.TestScenario

	for rows.Next() {
		var s models.TestScenario
		if err = rows.Scan(&s.ID, &s.Name, &s.BaseURL, &s.TotalDuration); err != nil {
			return nil, fmt.Errorf("scan scenario: %w", err)
		}

		stages, err := r.loadStages(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		s.Stages = stages

		s.StopConditions, err = r.loadStopConditions(ctx, s.ID)
		if err != nil {
			return nil, err
		}

		scenarios = append(scenarios, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return scenarios, nil
}

// Update - обновляет сценарий в базе данных.
func (r *ScenarioRepository) Update(ctx context.Context, s models.TestScenario) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const query = `
		UPDATE scenarios
		SET name           = $1,
		    base_url       = $2,
		    total_duration = $3,
		    updated_at     = NOW()
		WHERE id = $4`

	result, err := tx.ExecContext(ctx, query,
		s.Name, s.BaseURL, s.TotalDuration, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update scenario: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return service.ErrScenarioNotFound
	}

	// Удаляем старые stages каскадом (requests и chaos_params удалятся сами)
	// и вставляем заново — проще и надёжнее чем diff.
	if _, err = tx.ExecContext(ctx, `DELETE FROM stages WHERE scenario_id = $1`, s.ID); err != nil {
		return fmt.Errorf("delete old stages: %w", err)
	}

	if err = insertStages(ctx, tx, s.ID, s.Stages); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx,
		`DELETE FROM stop_conditions WHERE scenario_id = $1`, s.ID,
	); err != nil {
		return fmt.Errorf("delete old stop_conditions: %w", err)
	}

	if err = insertStopConditions(ctx, tx, s.ID, s.StopConditions); err != nil {
		return err
	}

	return tx.Commit()
}

// Delete - удаляет сценрий.
func (r *ScenarioRepository) Delete(ctx context.Context, id string) error {
	// stages, requests, chaos_params, stop_conditiosn удалятся каскадом через ON DELETE CASCADE
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM scenarios WHERE id = $1`, id,
	)
	if err != nil {
		r.logger.Error("Delete scenario failed", slog.String("id", id), slog.String("error", err.Error()))
		return fmt.Errorf("delete scenario: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return service.ErrScenarioNotFound
	}

	return nil
}

// loadStages загружает все этапы сценария вместе с запросами и chaos-событиями.
func (r *ScenarioRepository) loadStages(ctx context.Context, scenarioID string) ([]models.Stage, error) {
	const query = `
		SELECT id, type, duration, target_users
		FROM stages
		WHERE scenario_id = $1
		ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query, scenarioID)
	if err != nil {
		return nil, fmt.Errorf("load stages: %w", err)
	}
	defer rows.Close()

	var stages []models.Stage

	for rows.Next() {
		var stage models.Stage
		if err = rows.Scan(&stage.ID, &stage.Type, &stage.Duration, &stage.TargetUsers); err != nil {
			return nil, fmt.Errorf("scan stage: %w", err)
		}

		stage.Requests, err = r.loadRequests(ctx, scenarioID, stage.ID)
		if err != nil {
			return nil, err
		}

		stage.ChaosEvents, err = r.loadChaosParams(ctx, scenarioID, stage.ID)
		if err != nil {
			return nil, err
		}

		stages = append(stages, stage)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("stages rows: %w", err)
	}

	return stages, nil
}

// loadRequests загружает все запросы для конкретного этапа.
func (r *ScenarioRepository) loadRequests(ctx context.Context, scenarioID string, stageID int) ([]models.TestRequest, error) {
	const query = `
		SELECT name, method, path, headers, body, weight, expected_codes
		FROM test_requests
		WHERE scenario_id = $1 AND stage_id = $2
		ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query, scenarioID, stageID)
	if err != nil {
		return nil, fmt.Errorf("load requests: %w", err)
	}
	defer rows.Close()

	var requests []models.TestRequest

	for rows.Next() {
		var (
			req           models.TestRequest
			headersRaw    string        // для чтения headers.
			expectedCodes pq.Int64Array // pq умеет сканировать Int64Array из integer[].
		)

		if err = rows.Scan(
			&req.Name,
			&req.Method,
			&req.Path,
			&headersRaw,
			&req.Body,
			&req.Weight,
			&expectedCodes,
		); err != nil {
			return nil, fmt.Errorf("scan request: %w", err)
		}

		// JSON-строка → map[string]string.
		req.Headers = make(map[string]string)
		err = json.Unmarshal([]byte(headersRaw), &req.Headers)
		if err != nil {
			return nil, fmt.Errorf("unmarshal headers for request '%s': %w", req.Name, err)
		}

		// pq.Int64Array → []int
		req.ExpectedStatusCodes = make([]int, len(expectedCodes))
		for i, code := range expectedCodes {
			req.ExpectedStatusCodes[i] = int(code)
		}

		requests = append(requests, req)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("requests rows: %w", err)
	}

	return requests, nil
}

// loadChaosParams загружает все chaos-события для конкретного этапа.
func (r *ScenarioRepository) loadChaosParams(ctx context.Context, scenarioID string, stageID int) ([]models.ChaosParams, error) {
	const query = `
		SELECT type, target_container_id, start_delay, duration,
		       delay, jitter, packet_loss, cpu_quota, memory_bytes
		FROM chaos_params
		WHERE scenario_id = $1 AND stage_id = $2
		ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query, scenarioID, stageID)
	if err != nil {
		return nil, fmt.Errorf("load chaos params: %w", err)
	}
	defer rows.Close()

	var events []models.ChaosParams

	for rows.Next() {
		var c models.ChaosParams
		if err = rows.Scan(
			&c.Type,
			&c.TargetContainerID,
			&c.StartDelay,
			&c.Duration,
			&c.Delay,
			&c.Jitter,
			&c.PacketLoss,
			&c.CPUQuota,
			&c.MemoryBytes,
		); err != nil {
			return nil, fmt.Errorf("scan chaos param: %w", err)
		}
		events = append(events, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("chaos rows: %w", err)
	}

	return events, nil
}

func (r *ScenarioRepository) loadStopConditions(ctx context.Context, scenarioID string) (*models.StopConditions, error) {
	const q = `
        SELECT target_container_id, error_rate_percent,
               max_response_time_sec, max_cpu_percent, max_ram_percent
        FROM stop_conditions
        WHERE scenario_id = $1`

	var sc models.StopConditions
	err := r.db.QueryRowContext(ctx, q, scenarioID).Scan(
		&sc.TargetContainerID,
		&sc.ErrorRatePercent,
		&sc.MaxResponseTimeSec,
		&sc.MaxCPUPercent,
		&sc.MaxRAMPercent,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // критерии не заданы — это нормально
		}
		return nil, fmt.Errorf("load stop_conditions: %w", err)
	}
	return &sc, nil
}

type txExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertStages(ctx context.Context, tx txExecutor, scenarioID string, stages []models.Stage) error {
	for _, stage := range stages {
		const q = `
			INSERT INTO stages (id, scenario_id, type, duration, target_users)
			VALUES ($1, $2, $3, $4, $5)`

		if _, err := tx.ExecContext(ctx, q,
			stage.ID, scenarioID, stage.Type, stage.Duration, stage.TargetUsers,
		); err != nil {
			return fmt.Errorf("insert stage %d: %w", stage.ID, err)
		}

		if err := insertRequests(ctx, tx, scenarioID, stage.ID, stage.Requests); err != nil {
			return err
		}

		if err := insertChaosParams(ctx, tx, scenarioID, stage.ID, stage.ChaosEvents); err != nil {
			return err
		}
	}
	return nil
}

func insertRequests(ctx context.Context, tx txExecutor, scenarioID string, stageID int, requests []models.TestRequest) error {
	const q = `
		INSERT INTO test_requests
		    (scenario_id, stage_id, name, method, path, headers, body, weight, expected_codes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	for _, req := range requests {
		// Для хранения headers в базе.
		headersJSON, err := json.Marshal(req.Headers)
		if err != nil {
			return fmt.Errorf("marshal headers for request '%s': %w", req.Name, err)
		}
		if _, err := tx.ExecContext(ctx, q,
			scenarioID,
			stageID,
			req.Name,
			req.Method,
			req.Path,
			string(headersJSON),
			req.Body,
			req.Weight,
			pq.Array(req.ExpectedStatusCodes),
		); err != nil {
			return fmt.Errorf("insert request '%s': %w", req.Name, err)
		}
	}
	return nil
}

func insertChaosParams(ctx context.Context, tx txExecutor, scenarioID string, stageID int, events []models.ChaosParams) error {
	const q = `
		INSERT INTO chaos_params
		    (scenario_id, stage_id, type, target_container_id, start_delay, duration,
		     delay, jitter, packet_loss, cpu_quota, memory_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	for _, c := range events {
		if _, err := tx.ExecContext(ctx, q,
			scenarioID,
			stageID,
			c.Type,
			c.TargetContainerID,
			c.StartDelay,
			c.Duration,
			c.Delay,
			c.Jitter,
			c.PacketLoss,
			c.CPUQuota,
			c.MemoryBytes,
		); err != nil {
			return fmt.Errorf("insert chaos param: %w", err)
		}
	}
	return nil
}

func insertStopConditions(ctx context.Context, tx txExecutor, scenarioID string, sc *models.StopConditions) error {
	if sc == nil {
		return nil
	}

	const q = `
        INSERT INTO stop_conditions
            (scenario_id, target_container_id, error_rate_percent,
             max_response_time_sec, max_cpu_percent, max_ram_percent)
        VALUES ($1, $2, $3, $4, $5, $6)`

	if _, err := tx.ExecContext(ctx, q,
		scenarioID,
		sc.TargetContainerID,
		sc.ErrorRatePercent,
		sc.MaxResponseTimeSec,
		sc.MaxCPUPercent,
		sc.MaxRAMPercent,
	); err != nil {
		return fmt.Errorf("insert stop_conditions: %w", err)
	}
	return nil
}
