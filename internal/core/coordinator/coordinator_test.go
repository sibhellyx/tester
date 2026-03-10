package coordinator

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// MockLoadEngine имитирует работу нагрузочного движка.
type MockLoadEngine struct {
	mu           sync.Mutex
	CalledStages []int // Записываем ID этапов, которые были запущены
}

func (m *MockLoadEngine) ExecuteStage(ctx context.Context, stage models.Stage, results chan<- models.CallResult) {
	m.mu.Lock()
	m.CalledStages = append(m.CalledStages, stage.ID)
	m.mu.Unlock()

	// Имитируем бурную деятельность (отправляем 1 результат).
	select {
	case results <- models.CallResult{Status: 200, RequestName: "mock-req"}:
	case <-ctx.Done():
	}

	// В реальном LoadEngine этот метод блокирующий и ждет конца этапа.
	// Для теста отмены контекста лучше  небольшую задержку.
	select {
	case <-time.After(10 * time.Millisecond):
	case <-ctx.Done():
	}
}

func (m *MockLoadEngine) Shutdown() {
	// Заглушка для интерфейса.
}

// MockChaosEngine имитирует работу хаос-движка.
type MockChaosEngine struct {
	mu           sync.Mutex
	CalledEvents int // Считаем количество вызовов
}

func (m *MockChaosEngine) ExecuteRunning(ctx context.Context, events []models.ChaosParams) *sync.WaitGroup {
	m.mu.Lock()
	m.CalledEvents++
	m.mu.Unlock()
	return &sync.WaitGroup{}
}

// Инициализация новоого мок координатора
func newTestCoordinator() (*Coordinator, *MockLoadEngine, *MockChaosEngine) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil)) // Логгер в пустоту
	mockLoad := &MockLoadEngine{}
	mockChaos := &MockChaosEngine{}

	coord := NewCoordinator(logger, mockLoad, mockChaos)
	return coord, mockLoad, mockChaos
}

func TestCoordinator_RunTest_Success(t *testing.T) {
	coord, mockLoad, _ := newTestCoordinator()

	scenario := models.TestScenario{
		ID:      "test-success",
		BaseURL: "http://example.com", // Важно для Validate().
		Stages: []models.Stage{
			{ID: 1, Type: models.StageSteady, Duration: 10, TargetUsers: 5, Requests: []models.TestRequest{{Name: "R1", Weight: 1}}},
			{ID: 2, Type: models.StageSteady, Duration: 10, TargetUsers: 5, Requests: []models.TestRequest{{Name: "R2", Weight: 1}}},
		},
	}

	ctx := context.Background()
	resultsCh, err := coord.RunTest(ctx, scenario)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Читаем результаты до закрытия канала.
	count := 0
	for range resultsCh {
		count++
	}

	// Проверяем, что канал закрылся и данные пришли.
	if count == 0 {
		t.Error("Coordinator produced 0 results")
	}

	// Проверяем, что были вызваны оба этапа в правильном порядке.
	if len(mockLoad.CalledStages) != 2 {
		t.Errorf("Expected 2 stages executed, got %d", len(mockLoad.CalledStages))
	}
	if mockLoad.CalledStages[0] != 1 || mockLoad.CalledStages[1] != 2 {
		t.Errorf("Stages executed in wrong order: %v", mockLoad.CalledStages)
	}
}

func TestCoordinator_RunTest_WithChaos(t *testing.T) {
	coord, _, mockChaos := newTestCoordinator()

	scenario := models.TestScenario{
		ID:      "test-chaos",
		BaseURL: "http://example.com",
		Stages: []models.Stage{
			{
				ID: 1, Duration: 1, TargetUsers: 1,
				Requests: []models.TestRequest{{Name: "R1", Weight: 1}},
				// Добавляем событие хаоса
				ChaosEvents: []models.ChaosParams{
					{Type: "shutdown", TargetContainerID: "db", Duration: 1},
				},
			},
			{
				ID: 2, Duration: 1, TargetUsers: 1,
				Requests: []models.TestRequest{{Name: "R2", Weight: 1}},
				// Без хаоса.
			},
		},
	}

	resultsCh, _ := coord.RunTest(context.Background(), scenario)
	for range resultsCh {
	} // Ждем завершения.

	// Проверяем, что ChaosEngine был вызван ровно 1 раз (только для первого этапа).
	if mockChaos.CalledEvents != 1 {
		t.Errorf("Expected ChaosEngine to be called 1 time, got %d", mockChaos.CalledEvents)
	}
}

func TestCoordinator_RunTest_Cancel(t *testing.T) {
	coord, mockLoad, _ := newTestCoordinator()

	scenario := models.TestScenario{
		ID:      "test-cancel",
		BaseURL: "http://example.com",
		Stages: []models.Stage{
			{ID: 1, Duration: 1, TargetUsers: 1, Requests: []models.TestRequest{{Name: "R1", Weight: 1}}},
			{ID: 2, Duration: 1, TargetUsers: 1, Requests: []models.TestRequest{{Name: "R2", Weight: 1}}},
			{ID: 3, Duration: 1, TargetUsers: 1, Requests: []models.TestRequest{{Name: "R3", Weight: 1}}},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	resultsCh, _ := coord.RunTest(ctx, scenario)

	// Даем чуть-чуть времени на запуск первого этапа.
	time.Sleep(5 * time.Millisecond)

	// Отменяем тест!
	cancel()

	// Вычитываем канал до закрытия
	for range resultsCh {
	}

	// Проверяем, сколько этапов успело запуститься.
	// Должен быть только 1 (первый), так как мы отменили почти сразу.
	// Второй и третий не должны были стартовать.
	if len(mockLoad.CalledStages) >= 3 {
		t.Errorf("Coordinator should stop after cancel, but executed all stages: %v", mockLoad.CalledStages)
	}
}

func TestCoordinator_RunTest_InvalidScenario(t *testing.T) {
	coord, _, _ := newTestCoordinator()

	// Сценарий без BaseURL (невалидный)
	scenario := models.TestScenario{
		ID:     "invalid",
		Stages: []models.Stage{{}},
	}

	resultsCh, err := coord.RunTest(context.Background(), scenario)

	if err == nil {
		t.Error("Expected validation error, got nil")
	}
	if resultsCh != nil {
		t.Error("Expected nil channel on error")
	}
}
