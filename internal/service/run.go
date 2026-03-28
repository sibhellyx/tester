package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sibhellyx/tester/internal/core/checker"
	"github.com/sibhellyx/tester/internal/models"
)

// Ошибки слоя сервиса.
var (
	ErrRunNotFound      = errors.New("test run not found")
	ErrRunAlreadyActive = errors.New("test run is already active")
	ErrRunNotActive     = errors.New("test run is not running")
)

// TestRunRepository - интерфейс для работы с хранилищем запусков.
type TestRunRepositoryInterface interface {
	Create(ctx context.Context, run models.TestRun) error
	Get(ctx context.Context, id string) (*models.TestRun, error)
	UpdateStatus(ctx context.Context, id string, status models.TestRunStatus, finishedAt *time.Time, errMsg string) error
	ListByScenario(ctx context.Context, scenarioID string) ([]models.TestRun, error)
	SaveResults(ctx context.Context, runID string, results []models.CallResult) error
	GetSummary(ctx context.Context, runID string) (*models.TestRunSummary, error)
}

// CoordinatorInterface - интерфейс оркестратора нагрузочного теста.
type CoordinatorInterface interface {
	RunTest(ctx context.Context, scenario models.TestScenario) (<-chan models.CallResult, error)
}

// activeRun хранит контекст и cancel-функцию для запущенного теста.
type activeRun struct {
	cancel context.CancelFunc
	done   <-chan struct{} // закрывается когда горутина завершилась
}

// TestRunService реализует бизнес-логику управления запусками тестов.
type TestRunService struct {
	logger       *slog.Logger
	coordinator  CoordinatorInterface
	runRepo      TestRunRepositoryInterface
	scenarioRepo ScenarioRepositoryInterface // переиспользуем уже существующий интерфейс из scenario.go
	dockerClient checker.DockerClinetStatsInterface

	mu         sync.Mutex
	activeRuns map[string]*activeRun // runID -> activeRun
}

// NewTestRunService - конструктор.
func NewTestRunService(
	logger *slog.Logger,
	coordinator CoordinatorInterface,
	runRepo TestRunRepositoryInterface,
	scenarioRepo ScenarioRepositoryInterface,
	dockerClient checker.DockerClinetStatsInterface,
) *TestRunService {
	return &TestRunService{
		logger:       logger,
		coordinator:  coordinator,
		runRepo:      runRepo,
		scenarioRepo: scenarioRepo,
		dockerClient: dockerClient,
		activeRuns:   make(map[string]*activeRun),
	}
}

// StartTest запускает тест по ID сценария.
// Возвращает ID нового запуска.
func (s *TestRunService) StartTest(ctx context.Context, scenarioID string) (string, error) {
	// 1. Достаём сценарий из БД.
	scenario, err := s.scenarioRepo.Get(ctx, scenarioID)
	if err != nil {
		return "", ErrRepoError
	}
	if scenario == nil {
		return "", ErrScenarioNotFound
	}

	// 2. Создаём запись о запуске.
	now := time.Now()
	run := models.TestRun{
		ID:         uuid.New().String(),
		ScenarioID: scenarioID,
		Status:     models.StatusPending,
		StartedAt:  &now,
	}
	if err = s.runRepo.Create(ctx, run); err != nil {
		s.logger.Error("Failed to create test run", slog.String("error", err.Error()))
		return "", ErrRepoError
	}

	// 3. Запускаем тест асинхронно.
	runCtx, cancel := context.WithTimeout(context.Background(),
		time.Duration(scenario.TotalDuration+30)*time.Second, // +30с буфер
	)

	doneCh := make(chan struct{})
	ar := &activeRun{cancel: cancel, done: doneCh}

	s.mu.Lock()
	s.activeRuns[run.ID] = ar
	s.mu.Unlock()

	go s.executeRun(runCtx, cancel, run.ID, *scenario, doneCh)

	s.logger.Info("Test started", slog.String("run_id", run.ID), slog.String("scenario_id", scenarioID))
	return run.ID, nil
}

// executeRun запускает координатор и собирает результаты.
func (s *TestRunService) executeRun(
	ctx context.Context,
	cancel context.CancelFunc,
	runID string,
	scenario models.TestScenario,
	doneCh chan struct{},
) {
	defer func() {
		cancel()
		s.mu.Lock()
		delete(s.activeRuns, runID)
		s.mu.Unlock()
		close(doneCh)
	}()

	thresholdChecker := checker.NewThresholdChecker(scenario.StopConditions)
	monitor := checker.NewResourceMonitor(scenario.StopConditions, s.dockerClient, s.logger)
	monitor.Start(ctx) // no-op если контейнер не задан

	violated := false
	violationReason := ""

	// Обновляем статус на running.
	if err := s.runRepo.UpdateStatus(ctx, runID, models.StatusRunning, nil, ""); err != nil {
		s.logger.Error("Failed to set status=running", slog.String("run_id", runID), slog.String("error", err.Error()))
	}

	// Запускаем координатор, получаем канал результатов.
	resultsCh, err := s.coordinator.RunTest(ctx, scenario)
	if err != nil {
		s.logger.Error("Coordinator failed to start", slog.String("run_id", runID), slog.String("error", err.Error()))
		s.finishRun(runID, models.StatusFailed, err.Error())
		return
	}

	// Читаем результаты и пишем батчами в БД.
	const batchSize = 500
	batch := make([]models.CallResult, 0, batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer saveCancel()
		// Используем Background т.к. ctx теста мог быть отменён.
		if saveErr := s.runRepo.SaveResults(saveCtx, runID, batch); saveErr != nil {
			s.logger.Error("Failed to save results batch",
				slog.String("run_id", runID),
				slog.String("error", saveErr.Error()),
			)
		}
		batch = batch[:0]
	}

	for result := range resultsCh {
		batch = append(batch, result)
		thresholdChecker.Record(result)

		// Проверяем HTTP-критерии.
		if ok, reason := thresholdChecker.IsViolated(); ok {
			violated, violationReason = true, reason
			cancel()
			break
		}

		// Проверяем ресурсные критерии
		if ok, reason := monitor.IsViolated(); ok {
			violated, violationReason = true, reason
			cancel()
			break
		}

		if len(batch) >= batchSize {
			flush()
		}
	}
	flush() // последний батч

	// Определяем финальный статус.
	finalStatus := models.StatusFinished
	switch {
	case violated:
		s.logger.Warn("Test stopped by threshold",
			slog.String("run_id", runID),
			slog.String("reason", violationReason),
		)
		s.finishRun(runID, models.StatusFailed, violationReason)
		return
	case ctx.Err() != nil:
		finalStatus = models.StatusStopped
	}
	s.finishRun(runID, finalStatus, "")
}

// finishRun устанавливает финальный статус запуска в БД.
func (s *TestRunService) finishRun(runID string, status models.TestRunStatus, errMsg string) {
	now := time.Now()
	if err := s.runRepo.UpdateStatus(context.Background(), runID, status, &now, errMsg); err != nil {
		s.logger.Error("Failed to finish run",
			slog.String("run_id", runID),
			slog.String("status", string(status)),
			slog.String("error", err.Error()),
		)
	}
	s.logger.Info("Test run finished",
		slog.String("run_id", runID),
		slog.String("status", string(status)),
	)
}

// StopTest останавливает активный запуск теста.
func (s *TestRunService) StopTest(runID string) error {
	s.mu.Lock()
	ar, ok := s.activeRuns[runID]
	s.mu.Unlock()

	if !ok {
		// Проверяем, существует ли вообще такой run.
		run, err := s.runRepo.Get(context.Background(), runID)
		if err != nil || run == nil {
			return ErrRunNotFound
		}
		return ErrRunNotActive
	}

	ar.cancel()

	// Ждём завершения горутины (короткий таймаут).
	select {
	case <-ar.done:
	case <-time.After(5 * time.Second):
		s.logger.Warn("StopTest: goroutine did not finish in time", slog.String("run_id", runID))
	}

	s.logger.Info("Test stopped by request", slog.String("run_id", runID))
	return nil
}

// GetTestStatus возвращает сценарий + статус запуска.
func (s *TestRunService) GetTestStatus(ctx context.Context, runID string) (*models.TestWithStatus, error) {
	run, err := s.runRepo.Get(ctx, runID)
	if err != nil {
		return nil, ErrRepoError
	}
	if run == nil {
		return nil, ErrRunNotFound
	}

	scenario, err := s.scenarioRepo.Get(ctx, run.ScenarioID)
	if err != nil || scenario == nil {
		// Сценарий мог быть удалён — не фатально для статуса.
		s.logger.Warn("Scenario not found for run",
			slog.String("run_id", runID),
			slog.String("scenario_id", run.ScenarioID),
		)
		scenario = &models.TestScenario{ID: run.ScenarioID}
	}

	return &models.TestWithStatus{
		Run:      *run,
		Scenario: *scenario,
	}, nil
}

// ListRuns возвращает список запусков для сценария.
func (s *TestRunService) ListRuns(ctx context.Context, scenarioID string) ([]models.TestRun, error) {
	runs, err := s.runRepo.ListByScenario(ctx, scenarioID)
	if err != nil {
		return nil, ErrRepoError
	}
	return runs, nil
}

// GetSummary возвращает агрегированную статистику по запуску.
func (s *TestRunService) GetSummary(ctx context.Context, runID string) (*models.TestRunSummary, error) {
	run, err := s.runRepo.Get(ctx, runID)
	if err != nil {
		return nil, ErrRepoError
	}
	if run == nil {
		return nil, ErrRunNotFound
	}

	summary, err := s.runRepo.GetSummary(ctx, runID)
	if err != nil {
		return nil, ErrRepoError
	}
	return summary, nil
}
