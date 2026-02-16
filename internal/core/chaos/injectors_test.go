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
	CPUQuota     int64
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
func (m *MockContainerOps) UpdateResources(ctx context.Context, id string, cpu int64, mem int64) error {
	m.UpdateCalled = true
	m.CPUQuota = cpu
	return nil
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
	mock := &MockContainerOps{}
	injector := NewResourceLimitInjector(mock, "test-container", 50000, 1024)
	ctx := context.Background()

	// Test Inject.
	_ = injector.Inject(ctx)
	if !mock.UpdateCalled {
		t.Error("Inject should call UpdateResources")
	}
	if mock.CPUQuota != 50000 {
		t.Errorf("Expected CPU 50000, got %d", mock.CPUQuota)
	}

	// Test Recover.
	_ = injector.Recover(ctx)
	if mock.CPUQuota != -1 {
		t.Errorf("Recover should reset CPU to -1 (unlimited), got %d", mock.CPUQuota)
	}
}
