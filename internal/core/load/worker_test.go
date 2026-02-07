package load

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

type MockGenerator struct {
	requests []*models.TestRequest
	index    int
	mu       sync.Mutex
}

func NewMockGenerator(count int) *MockGenerator {
	requests := make([]*models.TestRequest, count)
	for i := 0; i < count; i++ {
		requests[i] = &models.TestRequest{
			Name:   "TestReq",
			Weight: 100,
		}
	}
	return &MockGenerator{
		requests: requests,
		index:    0,
	}
}

func (m *MockGenerator) Next() *models.TestRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.index >= len(m.requests) {
		return nil
	}

	req := m.requests[m.index]
	m.index++
	return req
}

// TestRunVirtualUser проверяет работу worker
func TestRunVirtualUser(t *testing.T) {
	// Подготовка зависимостей.
	// Подготовка сервера для теста.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Проверка механики цикла и остановки.
	attacker := NewAttacker(10 * time.Millisecond)

	gen := NewMockGenerator(1)
	// Буферизированный канал для записи результатов.
	results := make(chan models.CallResult, 1000)
	var wg sync.WaitGroup

	// Запуск воркера.
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)

	go RunVirtualUser(ctx, &wg, gen, attacker, results)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	wg.Wait() // Ждем завершения горутины.

	// Проверяем, что в канале появились результаты.
	if len(results) == 0 {
		t.Fatal("Worker did not produce any results")
	}

	// Вычитываем то, что успело нападать.
	close(results)
	count := 0
	for range results {
		count++
	}
	t.Logf("Worker made %d requests", count)
}
