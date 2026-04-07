package chaos

import (
	"context"
	"fmt"
)

type ResourceLimitInjector struct {
	client      ContainerOps
	containerID string
	// Целевые значения (nil = не трогать).
	cpuQuota    *int64
	memoryBytes *int64
	// Оригинальные значения, прочитанные перед Inject — нужны для Recover.
	originalCPUQuota    *int64
	originalMemory      *int64
	originalMemorySwap  *int64
}

// NewResourceLimitInjector функция инициализации сбоя.
// nil для cpuQuota или memoryBytes означает "не трогать этот ресурс".
func NewResourceLimitInjector(client ContainerOps, containerID string, cpuQuota *int64, memoryBytes *int64) *ResourceLimitInjector {
	return &ResourceLimitInjector{
		client:      client,
		containerID: containerID,
		cpuQuota:    cpuQuota,
		memoryBytes: memoryBytes,
	}
}

// Inject функция применяющая сбой.
// Перед применением читает оригинальные лимиты для последующего восстановления.
func (r *ResourceLimitInjector) Inject(ctx context.Context) error {
	origCPU, origMem, origSwap, err := r.client.GetContainerLimits(ctx, r.containerID)
	if err != nil {
		return fmt.Errorf("failed to read original limits: %w", err)
	}

	if r.cpuQuota != nil {
		r.originalCPUQuota = &origCPU
	}
	if r.memoryBytes != nil {
		r.originalMemory = &origMem
		r.originalMemorySwap = &origSwap
	}

	// При установке нового лимита памяти Docker требует Memory <= MemorySwap.
	// Передаём -1 (unlimited swap), чтобы ограничить только RAM.
	var newSwap *int64
	if r.memoryBytes != nil {
		v := int64(-1)
		newSwap = &v
	}

	return r.client.UpdateResources(ctx, r.containerID, r.cpuQuota, r.memoryBytes, newSwap)
}

// Recover восстанавливает оригинальные лимиты, прочитанные перед Inject.
func (r *ResourceLimitInjector) Recover(ctx context.Context) error {
	return r.client.UpdateResources(ctx, r.containerID, r.originalCPUQuota, r.originalMemory, r.originalMemorySwap)
}

// String функция для возврата типа сбоя.
func (r *ResourceLimitInjector) String() string {
	cpu := "unlimited"
	if r.cpuQuota != nil {
		cpu = fmt.Sprintf("%.1f%%", float64(*r.cpuQuota)/100000.0*100)
	}
	mem := "unlimited"
	if r.memoryBytes != nil && *r.memoryBytes > 0 {
		mem = fmt.Sprintf("%dMB", *r.memoryBytes/1024/1024)
	}
	return fmt.Sprintf("Resource Limit CPU:%s MEM:%s on %s", cpu, mem, r.containerID)
}
