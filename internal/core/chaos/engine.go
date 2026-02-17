package chaos

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

type Engine struct {
	logger *slog.Logger
	client ContainerOps
}

func NewEngine(logger *slog.Logger, client ContainerOps) *Engine {
	return &Engine{
		logger: logger,
		client: client,
	}
}

// ExecuteRunning запускает сбои для текущего этапа.
// Метод неблокирующий (запускает горутины).
// Возвращает waitGroup, чтобы основной оркестратор мог дождаться завершения очистки (Recover), если нужно.
func (e *Engine) ExecuteRunning(ctx context.Context, events []models.ChaosParams) *sync.WaitGroup {
	var wg sync.WaitGroup

	for _, event := range events {
		// Создаем инжектор через фабрику.
		injector, err := CreateInjector(e.client, event)
		if err != nil {
			e.logger.Error("Failed to create chaos injector", slog.String("type", string(event.Type)), slog.Any("err", err))
			continue
		}

		wg.Add(1)
		go func(ev models.ChaosParams, inj Injector) {
			defer wg.Done()
			e.runSingleFault(ctx, ev, inj)
		}(event, injector)
	}

	return &wg
}

func (e *Engine) runSingleFault(ctx context.Context, params models.ChaosParams, injector Injector) {
	// Ждем StartDelay.
	timer := time.NewTimer(time.Duration(params.StartDelay) * time.Second)
	select {
	case <-timer.C:
		// Время пришло.
	case <-ctx.Done():
		timer.Stop()
		return // Этап закончился раньше, чем сбой начался.
	}

	e.logger.Info("Injecting fault", slog.String("fault", injector.String()))

	// Ломаем (Inject)
	err := injector.Inject(ctx)
	if err != nil {
		e.logger.Error("Chaos injection failed", slog.Any("err", err))
		return
	}

	// Ждем Duration (пока система сломана).
	select {
	case <-time.After(time.Duration(params.Duration) * time.Second):
		// Сбой отработал свое время
	case <-ctx.Done():
		// Этап завершился досрочно - нужно срочно чинить!
		e.logger.Info("Context cancelled during chaos, recovering immediately")
	}

	e.logger.Info("Recovering fault", slog.String("fault", injector.String()))

	// 4. Чиним (Recover)
	// Используем отдельный контекст, так как ctx уже может быть отменен,
	// а нам нужно гарантированно восстановить систему.
	recoverCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = injector.Recover(recoverCtx)
	if err != nil {
		e.logger.Error("CRITICAL: Failed to recover fault", slog.Any("err", err))
	}
}
