package chaos

import (
	"context"
	"fmt"
)

type ResourceLimitInjector struct {
	client      ContainerOps
	containerID string
	cpuQuota    int64 // -1 = unlimit, 100000 = 1 CPU
	memoryBytes int64 // 0 = no change
}

// NewResourceLimitInjector функция инициализации сбоя.
func NewResourceLimitInjector(client ContainerOps, containerID string, cpu int64, mem int64) *ResourceLimitInjector {
	return &ResourceLimitInjector{
		client:      client,
		containerID: containerID,
		cpuQuota:    cpu,
		memoryBytes: mem,
	}
}

// Inject функция применяющая сбой.
func (r *ResourceLimitInjector) Inject(ctx context.Context) error {
	return r.client.UpdateResources(ctx, r.containerID, r.cpuQuota, r.memoryBytes)
}

// Recover функция восстанавливающая контейнер после сбоя.
func (r *ResourceLimitInjector) Recover(ctx context.Context) error {
	// Восстанавливаем значения по умолчанию (-1 для CPU означает безлимит)
	// Для памяти 0 означает "не обновлять", но чтобы снять лимит памяти,
	// нужно знать изначальное значение или выставить очень большое.
	// В Docker API для снятия лимита памяти обычно ставят 0 (если это swap) или -1 (недокументировано явно в go-sdk, но работает в API).
	// Безопаснее всего вернуть -1 для CPUQuota.
	return r.client.UpdateResources(ctx, r.containerID, -1, 0)
}

// String функция для возврата типа сбоя.
func (r *ResourceLimitInjector) String() string {
	return fmt.Sprintf("Resource Limit CPU:%d MEM:%d on %s", r.cpuQuota, r.memoryBytes, r.containerID)
}
