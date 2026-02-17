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
		return NewResourceLimitInjector(client, params.TargetContainerID, params.CPUQuota, params.MemoryBytes), nil

	default:
		return nil, fmt.Errorf("unknown chaos type: %s", params.Type)
	}
}
