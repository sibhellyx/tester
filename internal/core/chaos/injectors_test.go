package chaos

import (
	"context"
	"strings"
	"testing"
	"time"
)

// MockContainerOps - заглушка для тестов.
type MockContainerOps struct {
	StopCalled   bool
	StartCalled  bool
	ExecCmd      []string
	UpdateCalled bool
	CPUQuota     *int64
	MemoryBytes  *int64
	MemorySwap   *int64
	// Значения, которые возвращает GetContainerLimits (имитация текущих лимитов).
	MockCPUQuota   int64
	MockMemory     int64
	MockMemorySwap int64
}

func (m *MockContainerOps) StopContainer(ctx context.Context, id string, t int) error {
	m.StopCalled = true
	return nil
}
func (m *MockContainerOps) StartContainer(ctx context.Context, id string) error {
	m.StartCalled = true
	return nil
}
func (m *MockContainerOps) ExecCommand(ctx context.Context, id string, cmd []string) error {
	m.ExecCmd = cmd
	return nil
}
func (m *MockContainerOps) UpdateResources(ctx context.Context, id string, cpu *int64, mem *int64, swap *int64) error {
	m.UpdateCalled = true
	m.CPUQuota = cpu
	m.MemoryBytes = mem
	m.MemorySwap = swap
	return nil
}
func (m *MockContainerOps) GetContainerLimits(_ context.Context, _ string) (int64, int64, int64, error) {
	return m.MockCPUQuota, m.MockMemory, m.MockMemorySwap, nil
}

func TestShutdownInjector(t *testing.T) {
	mock := &MockContainerOps{}
	injector := NewShutdownInjector(mock, "test-container", 10)
	ctx := context.Background()

	// Test Inject.
	_ = injector.Inject(ctx)
	if !mock.StopCalled {
		t.Error("Inject should call StopContainer")
	}

	// Test Recover.
	_ = injector.Recover(ctx)
	if !mock.StartCalled {
		t.Error("Recover should call StartContainer")
	}
}

func TestNetworkDelayInjector(t *testing.T) {
	mock := &MockContainerOps{}
	delay := 100 * time.Millisecond
	injector := NewNetworkDelayInjector(mock, "test-container", delay, 0)
	ctx := context.Background()

	// Test Inject
	_ = injector.Inject(ctx)

	// Проверяем, что команда tc сформирована верно.
	// Ожидаем: tc qdisc add dev eth0 root netem delay 100ms 0s.
	cmdStr := strings.Join(mock.ExecCmd, " ")
	if !strings.Contains(cmdStr, "netem delay 100ms") {
		t.Errorf("Unexpected exec command: %s", cmdStr)
	}

	// Test Recover.
	_ = injector.Recover(ctx)
	cmdStrRecover := strings.Join(mock.ExecCmd, " ")
	if !strings.Contains(cmdStrRecover, "qdisc del") {
		t.Errorf("Recover should delete qdisc rules")
	}
}

func TestResourceLimitInjector(t *testing.T) {
	cpu := int64(50000)
	mem := int64(1024)
	// Оригинальные лимиты контейнера до инжекта.
	mock := &MockContainerOps{MockCPUQuota: 100000, MockMemory: 6291456, MockMemorySwap: -1}
	injector := NewResourceLimitInjector(mock, "test-container", &cpu, &mem)
	ctx := context.Background()

	// Test Inject: применяем новые лимиты.
	_ = injector.Inject(ctx)
	if !mock.UpdateCalled {
		t.Error("Inject should call UpdateResources")
	}
	if mock.CPUQuota == nil || *mock.CPUQuota != 50000 {
		t.Errorf("Expected CPU 50000, got %v", mock.CPUQuota)
	}
	if mock.MemoryBytes == nil || *mock.MemoryBytes != 1024 {
		t.Errorf("Expected memory 1024, got %v", mock.MemoryBytes)
	}
	// При установке лимита памяти должен быть передан MemorySwap=-1.
	if mock.MemorySwap == nil || *mock.MemorySwap != -1 {
		t.Errorf("Inject should set MemorySwap=-1, got %v", mock.MemorySwap)
	}

	// Test Recover: восстанавливаем оригинальные значения.
	_ = injector.Recover(ctx)
	if mock.CPUQuota == nil || *mock.CPUQuota != 100000 {
		t.Errorf("Recover should restore original CPU 100000, got %v", mock.CPUQuota)
	}
	if mock.MemoryBytes == nil || *mock.MemoryBytes != 6291456 {
		t.Errorf("Recover should restore original memory 6291456, got %v", mock.MemoryBytes)
	}
	if mock.MemorySwap == nil || *mock.MemorySwap != -1 {
		t.Errorf("Recover should restore original MemorySwap -1, got %v", mock.MemorySwap)
	}
}

func TestResourceLimitInjector_OnlyCPU(t *testing.T) {
	cpu := int64(25000)
	mock := &MockContainerOps{MockCPUQuota: 100000, MockMemory: 0, MockMemorySwap: 0}
	injector := NewResourceLimitInjector(mock, "test-container", &cpu, nil)
	ctx := context.Background()

	_ = injector.Inject(ctx)
	if mock.CPUQuota == nil || *mock.CPUQuota != 25000 {
		t.Errorf("Expected CPU 25000, got %v", mock.CPUQuota)
	}
	if mock.MemoryBytes != nil {
		t.Errorf("Memory should not be touched when only CPU is set, got %v", mock.MemoryBytes)
	}

	_ = injector.Recover(ctx)
	if mock.CPUQuota == nil || *mock.CPUQuota != 100000 {
		t.Errorf("Recover should restore original CPU 100000, got %v", mock.CPUQuota)
	}
	if mock.MemoryBytes != nil {
		t.Errorf("Recover should not touch memory when it was not set, got %v", mock.MemoryBytes)
	}
}

func TestResourceLimitInjector_OnlyMemory(t *testing.T) {
	mem := int64(256 * 1024 * 1024)
	originalMem := int64(512 * 1024 * 1024)
	originalSwap := int64(-1)
	mock := &MockContainerOps{MockCPUQuota: -1, MockMemory: originalMem, MockMemorySwap: originalSwap}
	injector := NewResourceLimitInjector(mock, "test-container", nil, &mem)
	ctx := context.Background()

	_ = injector.Inject(ctx)
	if mock.CPUQuota != nil {
		t.Errorf("CPU should not be touched when only memory is set, got %v", mock.CPUQuota)
	}
	if mock.MemoryBytes == nil || *mock.MemoryBytes != mem {
		t.Errorf("Expected memory %d, got %v", mem, mock.MemoryBytes)
	}

	_ = injector.Recover(ctx)
	if mock.CPUQuota != nil {
		t.Errorf("Recover should not touch CPU when it was not set, got %v", mock.CPUQuota)
	}
	if mock.MemoryBytes == nil || *mock.MemoryBytes != originalMem {
		t.Errorf("Recover should restore original memory %d, got %v", originalMem, mock.MemoryBytes)
	}
	if mock.MemorySwap == nil || *mock.MemorySwap != originalSwap {
		t.Errorf("Recover should restore original MemorySwap %d, got %v", originalSwap, mock.MemorySwap)
	}
}
