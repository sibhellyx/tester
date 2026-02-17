package chaos

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// MockInjector имитирует работу реального сбоя.
type MockInjector struct {
	InjectCalled  bool
	RecoverCalled bool
	InjectError   error
	RecoverError  error
}

func (m *MockInjector) Inject(ctx context.Context) error {
	m.InjectCalled = true
	return m.InjectError
}

func (m *MockInjector) Recover(ctx context.Context) error {
	m.RecoverCalled = true
	return m.RecoverError
}

func (m *MockInjector) String() string {
	return "MockInjector"
}

// Создает Engine с отключенным логгером.
func newTestEngine() *Engine {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewEngine(logger, &MockContainerOps{})
}

// Успешный цикл сбоя.
func TestEngine_runSingleFault_Success(t *testing.T) {
	engine := newTestEngine()
	mockInj := &MockInjector{}

	params := models.ChaosParams{
		Type:       models.ChaosShutdown,
		StartDelay: 0, // Сразу.
		Duration:   1, // 1 секунда.
	}

	ctx := context.Background()

	// Запускаем синхронно, чтобы проверить результат.
	engine.runSingleFault(ctx, params, mockInj)

	if !mockInj.InjectCalled {
		t.Error("Inject was not called")
	}
	if !mockInj.RecoverCalled {
		t.Error("Recover was not called")
	}
}

func TestEngine_runSingleFault_StartDelay(t *testing.T) {
	engine := newTestEngine()
	mockInj := &MockInjector{}

	params := models.ChaosParams{
		StartDelay: 1, // Ждем 1 сек.
		Duration:   1,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Запускаем в фоне.
	done := make(chan bool)
	go func() {
		engine.runSingleFault(ctx, params, mockInj)
		done <- true
	}()

	// Проверяем через 100мс (рано).
	time.Sleep(100 * time.Millisecond)
	if mockInj.InjectCalled {
		t.Error("Inject called too early! Should wait for StartDelay")
	}

	// Ждем окончания (1s delay + 1s duration + запас).
	time.Sleep(2500 * time.Millisecond)

	if !mockInj.InjectCalled {
		t.Error("Inject was never called")
	}

	cancel()
	<-done
}

func TestEngine_runSingleFault_RecoverOnCancel(t *testing.T) {
	engine := newTestEngine()
	mockInj := &MockInjector{}

	params := models.ChaosParams{
		StartDelay: 0,
		Duration:   10, // Долгий сбой.
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Запускаем
	go engine.runSingleFault(ctx, params, mockInj)

	// Ждем, пока сбой начнется (Inject).
	time.Sleep(100 * time.Millisecond)
	if !mockInj.InjectCalled {
		t.Fatal("Inject not called")
	}

	// Отменяем тест.
	cancel()

	// Ждем немного, чтобы сработал Recover.
	time.Sleep(100 * time.Millisecond)

	if !mockInj.RecoverCalled {
		t.Error("Recover should be called immediately after context cancel")
	}
}

func TestEngine_runSingleFault_CancelBeforeStart(t *testing.T) {
	engine := newTestEngine()
	mockInj := &MockInjector{}

	params := models.ChaosParams{
		StartDelay: 5, // Ждать долго.
		Duration:   1,
	}

	ctx, cancel := context.WithCancel(context.Background())

	go engine.runSingleFault(ctx, params, mockInj)

	time.Sleep(100 * time.Millisecond)

	// Отменяем.
	cancel()

	// Ждем завершения.
	time.Sleep(100 * time.Millisecond)

	if mockInj.InjectCalled {
		t.Error("Inject should NOT be called if context is cancelled during delay")
	}
	if mockInj.RecoverCalled {
		t.Error("Recover should NOT be called if inject never happened")
	}
}
