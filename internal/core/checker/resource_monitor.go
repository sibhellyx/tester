package checker

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

type StatsProvider interface {
	// GetStats возвращает состояние контейнера.
	GetStats(ctx context.Context, containerID string) (*models.ContainerStats, error)
}

type ResourceMonitor struct {
	conditions *models.StopConditions
	docker     StatsProvider
	logger     *slog.Logger

	violated atomic.Bool
	reason   atomic.Value // string
}

func NewResourceMonitor(
	sc *models.StopConditions,
	docker StatsProvider,
	logger *slog.Logger,
) *ResourceMonitor {
	return &ResourceMonitor{
		conditions: sc,
		docker:     docker,
		logger:     logger,
	}
}

// Start запускает фоновый опрос метрик.
// Если контейнер не указан — сразу возвращается, ничего не делает.
func (m *ResourceMonitor) Start(ctx context.Context) {
	// Нет контейнера или нет Docker-критериев — мониторинг не нужен
	if m.conditions == nil ||
		m.conditions.TargetContainerID == "" ||
		(m.conditions.MaxCPUPercent == nil && m.conditions.MaxRAMPercent == nil) {
		return
	}

	go func() {
		// Первый снимок Docker Stats всегда возвращает CPU=0 (нет дельты).
		// Пропускаем его явно — ждём один тик перед первой проверкой.
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		firstTick := true
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if firstTick {
					firstTick = false
					continue // пропускаем первый тик
				}
				m.check(ctx)
			}
		}
	}()
}

func (m *ResourceMonitor) check(ctx context.Context) {
	stats, err := m.docker.GetStats(ctx, m.conditions.TargetContainerID)
	if err != nil {
		m.logger.Warn("ResourceMonitor: failed to get container stats",
			slog.String("container", m.conditions.TargetContainerID),
			slog.Any("err", err),
		)
		return
	}

	if m.conditions.MaxCPUPercent != nil && stats.CPUPercent >= *m.conditions.MaxCPUPercent {
		m.violated.Store(true)
		m.reason.Store(fmt.Sprintf(
			"CPU %.1f%% exceeded threshold %.1f%%",
			stats.CPUPercent, *m.conditions.MaxCPUPercent,
		))
		return
	}

	if m.conditions.MaxRAMPercent != nil && stats.MemPercent >= *m.conditions.MaxRAMPercent {
		m.violated.Store(true)
		m.reason.Store(fmt.Sprintf(
			"RAM %.1f%% exceeded threshold %.1f%%",
			stats.MemPercent, *m.conditions.MaxRAMPercent,
		))
	}
}

func (m *ResourceMonitor) IsViolated() (bool, string) {
	if m.violated.Load() {
		reason, _ := m.reason.Load().(string)
		return true, reason
	}
	return false, ""
}
