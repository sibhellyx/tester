package chaos

import (
	"context"
	"fmt"
	"time"
)

// NetworkDelayInjector добавляет задержку (latency).
type NetworkDelayInjector struct {
	client      ContainerOps
	containerID string
	delay       time.Duration
	jitter      time.Duration
}

// NewNetworkDelayInjector функция инициализации сбоя.
func NewNetworkDelayInjector(client ContainerOps, containerID string, delay, jitter time.Duration) *NetworkDelayInjector {
	return &NetworkDelayInjector{
		client:      client,
		containerID: containerID,
		delay:       delay,
		jitter:      jitter,
	}
}

// Inject функция применяющая сбой.
func (n *NetworkDelayInjector) Inject(ctx context.Context) error {
	// tc qdisc add dev eth0 root netem delay 100ms 10ms
	cmd := []string{
		"tc", "qdisc", "add", "dev", "eth0", "root", "netem",
		"delay", n.delay.String(), n.jitter.String(),
	}
	return n.client.ExecCommand(ctx, n.containerID, cmd)
}

// Recover функция восстанавливающая контейнер после сбоя.
func (n *NetworkDelayInjector) Recover(ctx context.Context) error {
	// Удаляем правило.
	cmd := []string{"tc", "qdisc", "del", "dev", "eth0", "root"}
	// Игнорируем ошибку, так как Recover должен быть идемпотентным.
	_ = n.client.ExecCommand(ctx, n.containerID, cmd)
	return nil
}

// String функция для возврата типа сбоя.
func (n *NetworkDelayInjector) String() string {
	return fmt.Sprintf("Network Delay %s on %s", n.delay, n.containerID)
}
