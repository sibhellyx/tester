package load

import (
	"context"
	"log/slog"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// AttackerToolInterface - интерфейс для выполнения запроса.
type AttackerToolInterface interface {
	Shoot(requestModel models.TestRequest) models.CallResult
}

// Engine - структура, координирующая нагрузочное тестирование.
// pool хранит активных виртуальных пользователей между этапами.
type Engine struct {
	logger   *slog.Logger
	attacker AttackerToolInterface
	pool     *userPool // пул пользователей, живёт на протяжении всего теста
}

// NewEngine создаёт оркестратор нагрузочного тестирования.
func NewEngine(logger *slog.Logger, attacker AttackerToolInterface) *Engine {
	return &Engine{
		logger:   logger,
		attacker: attacker,
		pool:     newUserPool(),
	}
}

// ExecuteStage выполняет один этап теста.
//
// Ключевое решение по контексту:
//   - stageCtx используется только для ожидания завершения этапа (select внизу)
//   - пользователи запускаются с родительским ctx — они переживают завершение этапа
//   - это позволяет пользователям из RampUp продолжать работу на Steady и RampDown
func (e *Engine) ExecuteStage(
	ctx context.Context,
	stage models.Stage,
	results chan<- models.CallResult,
) {
	e.logger.Info("Starting stage",
		slog.Int("stage_id", stage.ID),
		slog.String("type", string(stage.Type)),
	)

	generator, err := NewWeightedGenerator(stage.Requests)
	if err != nil {
		e.logger.Error("Failed to create generator", slog.Any("err", err))
		return
	}

	// stageCtx определяет длительность этапа.
	// Пользователи к нему НЕ привязаны — они привязаны к родительскому ctx.
	stageCtx, cancel := context.WithTimeout(ctx, time.Duration(stage.Duration)*time.Second)
	defer cancel()

	switch stage.Type {

	case models.StageSteady:
		// Выравниваем количество пользователей до TargetUsers.
		// Если пользователей меньше — добавляем, если больше — убираем.
		current := e.pool.Len()
		delta := stage.TargetUsers - current

		switch {
		case delta > 0:
			e.logger.Info("Steady: spawning users",
				slog.Int("current", current),
				slog.Int("target", stage.TargetUsers),
				slog.Int("spawning", delta),
			)
			e.pool.Spawn(ctx, delta, generator, e.attacker, results)

		case delta < 0:
			e.logger.Info("Steady: killing excess users",
				slog.Int("current", current),
				slog.Int("target", stage.TargetUsers),
				slog.Int("killing", -delta),
			)
			e.pool.Kill(-delta)

		default:
			e.logger.Info("Steady: user count already at target",
				slog.Int("target", stage.TargetUsers),
			)
		}

	case models.StageRampUp:
		// Постепенно добавляем пользователей до TargetUsers в течение Duration.
		// Интервал между добавлением = Duration / количество_новых_пользователей.
		current := e.pool.Len()
		toAdd := stage.TargetUsers - current

		if toAdd <= 0 {
			e.logger.Info("RampUp: already at or above target, nothing to add",
				slog.Int("current", current),
				slog.Int("target", stage.TargetUsers),
			)
			break
		}

		interval := time.Duration(stage.Duration) * time.Second / time.Duration(toAdd)
		e.logger.Info("RampUp: starting",
			slog.Int("current", current),
			slog.Int("target", stage.TargetUsers),
			slog.Int("to_add", toAdd),
			slog.Duration("interval", interval),
		)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for i := 0; i < toAdd; i++ {
			select {
			case <-stageCtx.Done():
				// Этап прервался раньше — прекращаем добавление.
				e.logger.Info("RampUp: interrupted before reaching target",
					slog.Int("added", i),
					slog.Int("target", toAdd),
				)
				return
			case <-ticker.C:
				e.pool.Spawn(ctx, 1, generator, e.attacker, results)
				e.logger.Info("RampUp: user added",
					slog.Int("current", e.pool.Len()),
					slog.Int("target", stage.TargetUsers),
				)
			}
		}

	case models.StageRampDown:
		// Постепенно убираем пользователей до TargetUsers в течение Duration.
		// Интервал между удалением = Duration / количество_удаляемых_пользователей.
		current := e.pool.Len()
		toRemove := current - stage.TargetUsers

		if toRemove <= 0 {
			e.logger.Info("RampDown: already at or below target, nothing to remove",
				slog.Int("current", current),
				slog.Int("target", stage.TargetUsers),
			)
			break
		}

		interval := time.Duration(stage.Duration) * time.Second / time.Duration(toRemove)
		e.logger.Info("RampDown: starting",
			slog.Int("current", current),
			slog.Int("target", stage.TargetUsers),
			slog.Int("to_remove", toRemove),
			slog.Duration("interval", interval),
		)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for i := 0; i < toRemove; i++ {
			select {
			case <-stageCtx.Done():
				e.logger.Info("RampDown: interrupted before reaching target",
					slog.Int("removed", i),
					slog.Int("target", toRemove),
				)
				return
			case <-ticker.C:
				e.pool.Kill(1)
				e.logger.Info("RampDown: user removed",
					slog.Int("current", e.pool.Len()),
					slog.Int("target", stage.TargetUsers),
				)
			}
		}

	default:
		e.logger.Warn("Unknown stage type", slog.String("type", string(stage.Type)))
	}

	// Ожидаем истечения длительности этапа.
	// Пользователи продолжают работать в это время.
	<-stageCtx.Done()
	if ctx.Err() != nil {
		e.logger.Info("Stage cancelled by parent context",
			slog.Int("stage_id", stage.ID),
		)
	} else {
		e.logger.Info("Stage duration expired",
			slog.Int("stage_id", stage.ID),
			slog.Int("duration", stage.Duration),
			slog.Int("active_users", e.pool.Len()),
		)
	}
}

// Shutdown останавливает всех активных пользователей и ждёт их завершения.
// Должен вызываться из Coordinator после завершения всех этапов,
// строго перед закрытием канала results.
func (e *Engine) Shutdown() {
	e.pool.KillAll()
}
