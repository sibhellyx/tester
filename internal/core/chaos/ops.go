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
	// UpdateResources обновляет потребляемые ресурсы.
	UpdateResources(ctx context.Context, containerID string, cpuQuota int64, memoryBytes int64) error
}
