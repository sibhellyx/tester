package chaos

import (
	"context"
	"fmt"
)

// PacketLossInjector добавляет потерю пакетов.
type PacketLossInjector struct {
	client      ContainerOps
	containerID string
	percentage  int // Процент потерь (0-100)
}

// NewPacketLossInjector функция инициализации сбоя. 
func NewPacketLossInjector(client ContainerOps, containerID string, percentage int) *PacketLossInjector {
	return &PacketLossInjector{
		client:      client,
		containerID: containerID,
		percentage:  percentage,
	}
}

// Inject функция применяющая сбой.
func (p *PacketLossInjector) Inject(ctx context.Context) error {
	// tc qdisc add dev eth0 root netem loss 10%
	lossStr := fmt.Sprintf("%d%%", p.percentage)
	cmd := []string{
		"tc", "qdisc", "add", "dev", "eth0", "root", "netem",
		"loss", lossStr,
	}
	return p.client.ExecCommand(ctx, p.containerID, cmd)
}

// Recover функция восстанавливающая контейнер после сбоя.
func (p *PacketLossInjector) Recover(ctx context.Context) error {
	cmd := []string{"tc", "qdisc", "del", "dev", "eth0", "root"}
	_ = p.client.ExecCommand(ctx, p.containerID, cmd)
	return nil
}

// String функция для возврата типа сбоя.
func (p *PacketLossInjector) String() string {
	return fmt.Sprintf("Packet Loss %d%% on %s", p.percentage, p.containerID)
}
