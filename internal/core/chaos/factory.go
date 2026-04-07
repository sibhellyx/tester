package chaos

import (
	"fmt"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// CreateInjector создает нужную реализацию Injector на основе параметров.
func CreateInjector(client ContainerOps, params models.ChaosParams) (Injector, error) {
	switch params.Type {
	case models.ChaosShutdown:
		return NewShutdownInjector(client, params.TargetContainerID, 10), nil

	case models.ChaosNetworkDelay:
		delay, err := time.ParseDuration(params.Delay)
		if err != nil {
			return nil, fmt.Errorf("invalid delay format: %w", err)
		}
		jitter, _ := time.ParseDuration(params.Jitter) // Игнорируем ошибку, если пусто = 0.
		return NewNetworkDelayInjector(client, params.TargetContainerID, delay, jitter), nil

	case models.ChaosPacketLoss:
		return NewPacketLossInjector(client, params.TargetContainerID, params.PacketLoss), nil

	case models.ChaosResource:
		// nil = не трогать этот ресурс.
		var cpuQuota *int64
		if params.CPUPercent > 0 {
			// Конвертируем cpu_percent → Docker CFS quota (мкс за 100мс период).
			// 100% = 100000 мкс (1 полный CPU), 1% = 1000 мкс (Docker минимум).
			v := int64(params.CPUPercent / 100.0 * 100000)
			cpuQuota = &v
		}
		var memoryBytes *int64
		if params.MemoryMB > 0 {
			// Конвертируем memory_mb → байты.
			v := params.MemoryMB * 1024 * 1024
			memoryBytes = &v
		}
		return NewResourceLimitInjector(client, params.TargetContainerID, cpuQuota, memoryBytes), nil

	default:
		return nil, fmt.Errorf("unknown chaos type: %s", params.Type)
	}
}
