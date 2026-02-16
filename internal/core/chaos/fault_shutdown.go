package chaos

import (
	"context"
	"fmt"
)

// ShutdownInjector выключает и включает контейнер.
type ShutdownInjector struct {
	client      ContainerOps
	containerID string
	timeout     int
}

// NewShutdownInjector функция инициализации сбоя.
func NewShutdownInjector(client ContainerOps, containerID string, timeout int) *ShutdownInjector {
	return &ShutdownInjector{
		client:      client,
		containerID: containerID,
		timeout:     timeout,
	}
}

// Inject функция применяющая сбой.
func (s *ShutdownInjector) Inject(ctx context.Context) error {
	return s.client.StopContainer(ctx, s.containerID, s.timeout)
}

// Recover функция восстанавливающая контейнер после сбоя.
func (s *ShutdownInjector) Recover(ctx context.Context) error {
	return s.client.StartContainer(ctx, s.containerID)
}

// String функция для возврата типа сбоя.
func (s *ShutdownInjector) String() string {
	return fmt.Sprintf("Shutdown container %s", s.containerID)
}
