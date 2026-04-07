package chaos

import "context"

// ContainerOps интерфейс описывающий взаимодействие с docker контейнерами.
type ContainerOps interface {
	// StopContainer останавливает контейнер.
	StopContainer(ctx context.Context, containerID string, timeout int) error
	// StartContainer запускает контейнер.
	StartContainer(ctx context.Context, containerID string) error
	// ExecCommand выполнение команды в контейнере.
	ExecCommand(ctx context.Context, containerID string, cmd []string) error
	// UpdateResources обновляет лимиты ресурсов контейнера.
	// nil означает "не трогать этот ресурс".
	// memorySwap: -1 = без лимита swap, 0 = не трогать, >0 = явный лимит (memory + swap).
	UpdateResources(ctx context.Context, containerID string, cpuQuota *int64, memoryBytes *int64, memorySwap *int64) error
	// GetContainerLimits возвращает текущие лимиты ресурсов контейнера.
	// Возвращает cpuQuota (-1 = без лимита), memory (0 = без лимита), memorySwap (-1 = без лимита).
	GetContainerLimits(ctx context.Context, containerID string) (cpuQuota int64, memory int64, memorySwap int64, err error)
}
