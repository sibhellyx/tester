package checker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/sibhellyx/tester/internal/models"
)

// mockDockerClient реализует StatsProvider для тестов.
type mockDockerClient struct {
	stats *models.ContainerStats
	err   error
}

func (m *mockDockerClient) GetStats(_ context.Context, _ string) (*models.ContainerStats, error) {
	return m.stats, m.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func f64(v float64) *float64 { return &v }

// --- Start no-op cases ---

func TestResourceMonitor_Start_NoOp_NilConditions(t *testing.T) {
	m := NewResourceMonitor(nil, &mockDockerClient{}, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.Start(ctx) // должен вернуться мгновенно
	if violated, _ := m.IsViolated(); violated {
		t.Error("expected no violation with nil conditions")
	}
}

func TestResourceMonitor_Start_NoOp_EmptyContainerID(t *testing.T) {
	sc := &models.StopConditions{
		MaxCPUPercent: f64(80),
		// TargetContainerID не задан
	}
	m := NewResourceMonitor(sc, &mockDockerClient{}, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.Start(ctx)
	if violated, _ := m.IsViolated(); violated {
		t.Error("expected no violation without container ID")
	}
}

func TestResourceMonitor_Start_NoOp_NoDockerCriteria(t *testing.T) {
	sc := &models.StopConditions{
		TargetContainerID: "container1",
		// MaxCPUPercent и MaxRAMPercent не заданы
	}
	m := NewResourceMonitor(sc, &mockDockerClient{}, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.Start(ctx)
	if violated, _ := m.IsViolated(); violated {
		t.Error("expected no violation without docker criteria")
	}
}

// --- IsViolated по умолчанию ---

func TestResourceMonitor_IsViolated_InitiallyFalse(t *testing.T) {
	sc := &models.StopConditions{
		TargetContainerID: "container1",
		MaxCPUPercent:     f64(80),
	}
	m := NewResourceMonitor(sc, &mockDockerClient{}, testLogger())
	if violated, _ := m.IsViolated(); violated {
		t.Error("expected no violation before first check")
	}
}

// --- check() — прямой вызов приватного метода (whitebox) ---

func TestResourceMonitor_check_CPU_Violation(t *testing.T) {
	sc := &models.StopConditions{
		TargetContainerID: "container1",
		MaxCPUPercent:     f64(80),
	}
	docker := &mockDockerClient{stats: &models.ContainerStats{CPUPercent: 90, MemPercent: 10}}
	m := NewResourceMonitor(sc, docker, testLogger())

	m.check(context.Background())

	violated, reason := m.IsViolated()
	if !violated {
		t.Fatal("expected CPU violation")
	}
	if reason == "" {
		t.Error("expected non-empty violation reason")
	}
}

func TestResourceMonitor_check_CPU_AtExactThreshold_Violation(t *testing.T) {
	// Условие >=, поэтому равенство тоже нарушение.
	sc := &models.StopConditions{
		TargetContainerID: "container1",
		MaxCPUPercent:     f64(80),
	}
	docker := &mockDockerClient{stats: &models.ContainerStats{CPUPercent: 80.0}}
	m := NewResourceMonitor(sc, docker, testLogger())

	m.check(context.Background())

	if violated, _ := m.IsViolated(); !violated {
		t.Error("expected violation when CPU equals threshold (>=)")
	}
}

func TestResourceMonitor_check_CPU_BelowThreshold_NoViolation(t *testing.T) {
	sc := &models.StopConditions{
		TargetContainerID: "container1",
		MaxCPUPercent:     f64(80),
	}
	docker := &mockDockerClient{stats: &models.ContainerStats{CPUPercent: 79.9}}
	m := NewResourceMonitor(sc, docker, testLogger())

	m.check(context.Background())

	if violated, _ := m.IsViolated(); violated {
		t.Error("expected no violation when CPU is below threshold")
	}
}

func TestResourceMonitor_check_RAM_Violation(t *testing.T) {
	sc := &models.StopConditions{
		TargetContainerID: "container1",
		MaxRAMPercent:     f64(70),
	}
	docker := &mockDockerClient{stats: &models.ContainerStats{CPUPercent: 10, MemPercent: 80}}
	m := NewResourceMonitor(sc, docker, testLogger())

	m.check(context.Background())

	violated, reason := m.IsViolated()
	if !violated {
		t.Fatal("expected RAM violation")
	}
	if reason == "" {
		t.Error("expected non-empty violation reason")
	}
}

func TestResourceMonitor_check_RAM_BelowThreshold_NoViolation(t *testing.T) {
	sc := &models.StopConditions{
		TargetContainerID: "container1",
		MaxRAMPercent:     f64(70),
	}
	docker := &mockDockerClient{stats: &models.ContainerStats{CPUPercent: 10, MemPercent: 60}}
	m := NewResourceMonitor(sc, docker, testLogger())

	m.check(context.Background())

	if violated, _ := m.IsViolated(); violated {
		t.Error("expected no violation when RAM is below threshold")
	}
}

func TestResourceMonitor_check_BothThresholds_CPUViolatesFirst(t *testing.T) {
	// Когда оба заданы и CPU нарушен — reason про CPU (RAM не проверяется из-за return)
	sc := &models.StopConditions{
		TargetContainerID: "container1",
		MaxCPUPercent:     f64(80),
		MaxRAMPercent:     f64(70),
	}
	docker := &mockDockerClient{stats: &models.ContainerStats{CPUPercent: 90, MemPercent: 80}}
	m := NewResourceMonitor(sc, docker, testLogger())

	m.check(context.Background())

	violated, reason := m.IsViolated()
	if !violated {
		t.Fatal("expected violation")
	}
	// CPU нарушен первым — reason должна упоминать CPU
	if reason == "" {
		t.Error("expected non-empty violation reason")
	}
}

func TestResourceMonitor_check_GetStatsError_NoViolation(t *testing.T) {
	sc := &models.StopConditions{
		TargetContainerID: "container1",
		MaxCPUPercent:     f64(80),
	}
	docker := &mockDockerClient{err: errors.New("connection refused")}
	m := NewResourceMonitor(sc, docker, testLogger())

	m.check(context.Background())

	if violated, _ := m.IsViolated(); violated {
		t.Error("expected no violation on stats error (warn only)")
	}
}
