package coordinator

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/sibhellyx/tester/internal/core/load"
	"github.com/sibhellyx/tester/internal/models"
)

type ChaosEngineInterface interface {
	ExecuteRunning(ctx context.Context, events []models.ChaosParams) *sync.WaitGroup
}

type LoadEngineInterface interface {
	ExecuteStage(ctx context.Context, stage models.Stage, pool *load.UserPool, results chan<- models.CallResult)
	Shutdown(pool *load.UserPool)
}

// Coordinator управляет всем тестом: и нагрузкой, и хаосом.
type Coordinator struct {
	logger *slog.Logger
	// LoadEngine для управления нагрузочным тестированием.
	loadEngine LoadEngineInterface
	// ChaosEngine для управления стрессовым тестированием.
	chaosEngine ChaosEngineInterface
}

// NewCoordinator инициализация оркестратора для управления тестом.
func NewCoordinator(logger *slog.Logger, load LoadEngineInterface, chaos ChaosEngineInterface) *Coordinator {
	return &Coordinator{
		logger:      logger,
		loadEngine:  load,
		chaosEngine: chaos,
	}
}

// RunTest запускает полный сценарий тестирования.
func (c *Coordinator) RunTest(ctx context.Context, scenario models.TestScenario) (<-chan models.CallResult, error) {
	if err := scenario.Validate(); err != nil {
		return nil, err
	}
	// склеивание baseUrl with request path.
	scenario = *baseUrlWithPathRequest(&scenario)

	results := make(chan models.CallResult, 1000)

	go func() {
		defer close(results)

		pool := load.NewUserPool()

		c.logger.Info("Coordinator started test", slog.String("id", scenario.ID))

		for _, stage := range scenario.Stages {
			select {
			case <-ctx.Done():
				c.loadEngine.Shutdown(pool)
				return
			default:
			}

			c.runStage(ctx, stage, pool, results)
		}

		c.loadEngine.Shutdown(pool)
		c.logger.Info("Coordinator finished test")
	}()

	return results, nil
}

func (c *Coordinator) runStage(ctx context.Context, stage models.Stage, pool *load.UserPool, results chan<- models.CallResult) {
	c.logger.Info("Coordinator starting stage", slog.Int("id", stage.ID))

	// Запускаем Хаос (если есть события)
	if len(stage.ChaosEvents) > 0 {
		c.logger.Info("Initializing chaos events", slog.Int("count", len(stage.ChaosEvents)))
		// Запускаем хаос. Он работает в фоне.
		// wgChaos можно использовать, если мы хотим убедиться, что Recover прошел (сейчас не проверяю).
		_ = c.chaosEngine.ExecuteRunning(ctx, stage.ChaosEvents)
	}

	// Запускаем Нагрузку.
	// LoadEngine будет работать ровно stage.Duration.
	// ChaosEngine будет работать параллельно.
	c.loadEngine.ExecuteStage(ctx, stage, pool, results)

	c.logger.Info("Stage finished")
}

func baseUrlWithPathRequest(scenario *models.TestScenario) *models.TestScenario {
	base := strings.TrimRight(scenario.BaseURL, "/")
	for i := range scenario.Stages {
		for j := range scenario.Stages[i].Requests {
			path := scenario.Stages[i].Requests[j].Path
			scenario.Stages[i].Requests[j].Path = base + "/" + strings.TrimLeft(path, "/")
		}
	}
	return scenario
}
