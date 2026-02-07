package load

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// MockAttacker для тестов
type MockAttackerEngine struct {
	ShootDuration time.Duration
	mu            sync.Mutex
	callCount     int
}

func (m *MockAttackerEngine) Shoot(requestModel models.TestRequest) models.CallResult {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.ShootDuration > 0 {
		time.Sleep(m.ShootDuration)
	}

	return models.CallResult{
		RequestName: requestModel.Name,
		Status:      200,
		Duration:    m.ShootDuration,
		Timestamp:   time.Now(),
	}
}

func (m *MockAttackerEngine) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Создание тестового логгера
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Минимум логов в тестах
	}))
}

// Создание простого сценария
func createSimpleScenario() models.TestScenario {
	return models.TestScenario{
		ID:      "test-1",
		Name:    "Simple Test",
		BaseURL: "http://example.com",
		Stages: []models.Stage{
			{
				ID:          1,
				Type:        models.StageSteady,
				Duration:    1,
				TargetUsers: 2,
				Requests: []models.TestRequest{
					{
						Name:   "TestRequest",
						Method: "GET",
						Path:   "http://example.com/test",
						Weight: 100,
					},
				},
			},
		},
	}
}

// TestNewEngine - проверка создания движка.
func TestNewEngine(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{}

	engine := NewEngine(logger, attacker)

	if engine == nil {
		t.Fatal("Engine should not be nil")
	}
	if engine.logger != logger {
		t.Error("Logger not set correctly")
	}
	if engine.attacker != attacker {
		t.Error("Attacker not set correctly")
	}
}

// TestEngine_Run_Success - базовый успешный сценарий.
func TestEngine_Run_Success(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{ShootDuration: 10 * time.Millisecond}
	engine := NewEngine(logger, attacker)

	scenario := createSimpleScenario()
	ctx := context.Background()

	results, err := engine.Run(ctx, scenario)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Собираем результаты
	var count int
	for range results {
		count++
	}

	if count == 0 {
		t.Error("Expected some results, got 0")
	}

	t.Logf("Received %d results", count)
}

// TestEngine_Run_InvalidScenario - тест с невалидным сценарием.
func TestEngine_Run_InvalidScenario(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{}
	engine := NewEngine(logger, attacker)

	// Сценарий без BaseURL.
	invalidScenario := models.TestScenario{
		ID:      "invalid",
		BaseURL: "", // Пустой BaseURL.
		Stages:  []models.Stage{},
	}

	ctx := context.Background()
	results, err := engine.Run(ctx, invalidScenario)

	if err == nil {
		t.Error("Expected error for invalid scenario, got nil")
	}
	if results != nil {
		t.Error("Results should be nil for invalid scenario")
	}
}

// TestEngine_Run_ContextCancel - тест отмены через контекст.
func TestEngine_Run_ContextCancel(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{ShootDuration: 20 * time.Millisecond}
	engine := NewEngine(logger, attacker)

	// Длинный сценарий.
	scenario := models.TestScenario{
		ID:      "cancel-test",
		Name:    "Cancel Test",
		BaseURL: "http://example.com",
		Stages: []models.Stage{
			{
				ID:          1,
				Type:        models.StageSteady,
				Duration:    10, // 10 секунд.
				TargetUsers: 5,
				Requests: []models.TestRequest{
					{
						Name:   "LongRequest",
						Method: "GET",
						Path:   "http://example.com/long",
						Weight: 100,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	results, err := engine.Run(ctx, scenario)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Отменяем через 200ms.
	time.AfterFunc(200*time.Millisecond, cancel)

	start := time.Now()
	for range results {
		// Читаем результаты.
	}
	elapsed := time.Since(start)

	// Должен завершиться быстрее, чем за 10 секунд.
	if elapsed >= 5*time.Second {
		t.Errorf("Test should cancel quickly, took %v", elapsed)
	}

	// Проверяем, что контекст действительно отменен.
	select {
	case <-ctx.Done():
		t.Log("Context is cancelled")

		// Проверяем ошибку контекста.
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected context.Canceled error, got: %v", ctx.Err())
		}
	default:
		t.Error("Context should be cancelled but it's not")
	}

	t.Logf("Test cancelled after %v", elapsed)
}

// TestEngine_Run_StageDuration - тест истечения времени этапа.
func TestEngine_Run_StageDuration(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{ShootDuration: 50 * time.Millisecond}
	engine := NewEngine(logger, attacker)

	scenario := models.TestScenario{
		ID:      "duration-test",
		Name:    "Duration Test",
		BaseURL: "http://example.com",
		Stages: []models.Stage{
			{
				ID:          1,
				Type:        models.StageSteady,
				Duration:    1, // 1 секунда. 
				TargetUsers: 3,
				Requests: []models.TestRequest{
					{
						Name:   "SlowRequest",
						Method: "GET",
						Path:   "http://example.com/slow",
						Weight: 100,
					},
				},
			},
		},
	}

	ctx := context.Background()
	results, err := engine.Run(ctx, scenario)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	start := time.Now()
	for range results {
		// Читаем результаты. 
	}
	elapsed := time.Since(start)

	// Должен завершиться примерно за 1 секунду (+/- погрешность). 
	if elapsed < 800*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("Expected ~1s duration, got %v", elapsed)
	}

	t.Logf("Stage completed in %v", elapsed)
}

// TestEngine_Run_MultipleStages - тест нескольких этапов.
func TestEngine_Run_MultipleStages(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{ShootDuration: 10 * time.Millisecond}
	engine := NewEngine(logger, attacker)

	scenario := models.TestScenario{
		ID:      "multi-stage",
		Name:    "Multi Stage Test",
		BaseURL: "http://example.com",
		Stages: []models.Stage{
			{
				ID:          1,
				Type:        models.StageSteady,
				Duration:    1,
				TargetUsers: 2,
				Requests: []models.TestRequest{
					{
						Name:   "Stage1Request",
						Method: "GET",
						Path:   "http://example.com/stage1",
						Weight: 100,
					},
				},
			},
			{
				ID:          2,
				Type:        models.StageSteady,
				Duration:    1,
				TargetUsers: 3,
				Requests: []models.TestRequest{
					{
						Name:   "Stage2Request",
						Method: "GET",
						Path:   "http://example.com/stage2",
						Weight: 100,
					},
				},
			},
		},
	}

	ctx := context.Background()
	results, err := engine.Run(ctx, scenario)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var count int
	for range results {
		count++
	}

	if count == 0 {
		t.Error("Expected results from both stages")
	}

	t.Logf("Total results from both stages: %d", count)
}

// TestEngine_Run_ZeroUsers - тест с нулевым количеством пользователей.
func TestEngine_Run_ZeroUsers(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{}
	engine := NewEngine(logger, attacker)

	scenario := models.TestScenario{
		ID:      "zero-users",
		Name:    "Zero Users",
		BaseURL: "http://example.com",
		Stages: []models.Stage{
			{
				ID:          1,
				Type:        models.StageSteady,
				Duration:    1,
				TargetUsers: 0,
				Requests: []models.TestRequest{
					{
						Name:   "Request",
						Method: "GET",
						Path:   "http://example.com/test",
						Weight: 100,
					},
				},
			},
		},
	}

	ctx := context.Background()
	results, err := engine.Run(ctx, scenario)

	// Validate вернет ошибку для TargetUsers = 0.
	if err == nil {
		// Если валидация пропустила, проверяем результаты.
		var count int
		for range results {
			count++
		}
		if count != 0 {
			t.Errorf("Expected 0 results with 0 users, got %d", count)
		}
	}
}

// TestEngine_Run_UnknownStageType - тест неизвестного типа этапа.
func TestEngine_Run_UnknownStageType(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{}
	engine := NewEngine(logger, attacker)

	scenario := models.TestScenario{
		ID:      "unknown-type",
		Name:    "Unknown Type",
		BaseURL: "http://example.com",
		Stages: []models.Stage{
			{
				ID:          1,
				Type:        "INVALID_TYPE", // Неизвестный тип
				Duration:    1,
				TargetUsers: 2,
				Requests: []models.TestRequest{
					{
						Name:   "Request",
						Method: "GET",
						Path:   "http://example.com/test",
						Weight: 100,
					},
				},
			},
		},
	}

	ctx := context.Background()
	results, err := engine.Run(ctx, scenario)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var count int
	for range results {
		count++
	}

	// С неизвестным типом этапа результатов быть не должно.
	if count != 0 {
		t.Errorf("Expected 0 results with unknown stage type, got %d", count)
	}
}

// TestEngine_Run_ResultsChannel - проверка канала результатов.
func TestEngine_Run_ResultsChannel(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{ShootDuration: 10 * time.Millisecond}
	engine := NewEngine(logger, attacker)

	scenario := createSimpleScenario()
	ctx := context.Background()

	results, err := engine.Run(ctx, scenario)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Проверяем, что можем читать из канала.
	result, ok := <-results
	if !ok {
		t.Fatal("Channel closed too early")
	}

	if result.Status != 200 {
		t.Errorf("Expected status 200, got %d", result.Status)
	}

	// Дочитываем остальное.
	for range results {
	}

	// Проверяем, что канал закрыт.
	_, ok = <-results
	if ok {
		t.Error("Channel should be closed")
	}
}

// TestEngine_Run_ConcurrentUsers - тест параллельной работы пользователей.
func TestEngine_Run_ConcurrentUsers(t *testing.T) {
	logger := newTestLogger()
	attacker := &MockAttackerEngine{ShootDuration: 50 * time.Millisecond}
	engine := NewEngine(logger, attacker)

	scenario := models.TestScenario{
		ID:      "concurrent",
		Name:    "Concurrent Test",
		BaseURL: "http://example.com",
		Stages: []models.Stage{
			{
				ID:          1,
				Type:        models.StageSteady,
				Duration:    1,
				TargetUsers: 10, // Много пользователей
				Requests: []models.TestRequest{
					{
						Name:   "Request",
						Method: "GET",
						Path:   "http://example.com/test",
						Weight: 100,
					},
				},
			},
		},
	}

	ctx := context.Background()
	results, err := engine.Run(ctx, scenario)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var count int
	for range results {
		count++
	}

	if count == 0 {
		t.Error("Expected some results from concurrent users")
	}

	t.Logf("Concurrent users made %d requests", count)
}
