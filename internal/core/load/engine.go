package load

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// AttackerTool - интерфейс предоставляющий доступ к методу, для выполнения запроса к тесту.
type AttackerTool interface {
	Shoot(requestModel models.TestRequest) models.CallResult
}

// Engine - структура, координирующая нагрузочное тестирование.
type Engine struct {
	logger   *slog.Logger   // вывод logs.
	attacker AttackerTool   // инструмент для выполнения запроса.
	wg       sync.WaitGroup // для координации тестированния.
}

// NewEngine - создает оркестратор нагрузочного тестирования.
func NewEngine(logger *slog.Logger, attacker AttackerTool) *Engine {
	return &Engine{
		logger:   logger,
		attacker: attacker,
	}
}

// Run - функция, выполняющая тестирование.
func (e *Engine) Run(ctx context.Context, scenario models.TestScenario) (<-chan models.CallResult, error) {
	// Проверка корректности сценария.
	err := scenario.Validate()
	if err != nil {
		return nil, err
	}
	// Создание канала для результатов тестирования.
	results := make(chan models.CallResult, 1000)
	// Проход и выполненеие всех этапов из сценария.
	go func() {
		// закрытие канала с результами по завершению.
		defer close(results)

		e.logger.Info("Test started", slog.String("scenario_id", scenario.ID))

		for _, stage := range scenario.Stages {
			select {
			case <-ctx.Done():
				e.logger.Info("Test execution cancelled")
				return
			default:
				// Продолжаем выполение теста.
			}
			// Выполнение конкретного этапа.
			e.ExecuteStage(ctx, stage, results)
		}
		e.logger.Info("Test finished successfully", slog.String("scenario_id", scenario.ID))
	}()
	return results, nil
}

// ExecuteStage - метод выполняющий конкретный переданный этап.
func (e *Engine) ExecuteStage(
	ctx context.Context,
	stage models.Stage,
	results chan<- models.CallResult,
) {
	e.logger.Info("Starting stage",
		slog.Int("stage_id", stage.ID),
		slog.String("type", string(stage.Type)),
	)
	// Инициализация генератора для выполнения нагрузки.
	generatorForStage, err := NewWeightedGenerator(stage.Requests)
	if err != nil {
		e.logger.Error("Failed to create generator", slog.Any("err", err))
		return // прерываем тест при невозможности создать генератор.
	}

	// Контекст этапа с отменой по длительности этапа.
	stageCtx, cancel := context.WithTimeout(ctx, time.Duration(stage.Duration)*time.Second)
	defer cancel()

	// выполнение этапа в соответствие с типом его тестирования.
	switch stage.Type {
	case models.StageSteady:
		e.wg.Add(stage.TargetUsers)
		for i := 0; i < stage.TargetUsers; i++ {
			go RunVirtualUser(stageCtx, &e.wg, generatorForStage, e.attacker, results)
		}
	default:
		e.logger.Warn("Unknown stage type", slog.String("type", string(stage.Type)))
	}
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-stageCtx.Done():
		// Проверяем причину завершения контекста.
		if ctx.Err() != nil {
			// Родительский контекст был отменен.
			e.logger.Info("Stage cancelled by parent context",
				slog.Int("stage_id", stage.ID),
			)
		} else {
			// Истекло время этапа.
			e.logger.Info("Stage duration expired",
				slog.Int("stage_id", stage.ID),
				slog.Int("duration", stage.Duration),
			)
		}
		// Ждем завершения всех горутин.
		e.wg.Wait()
	case <-done:
		// Все пользователи завершились раньше таймаута.
		e.logger.Info("Stage completed",
			slog.Int("stage_id", stage.ID),
		)
	}

}
