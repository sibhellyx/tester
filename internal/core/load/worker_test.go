package load

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type MockGenerator struct {
	req *TestRequest
}

func (m *MockGenerator) Next() *TestRequest {
	return m.req
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

	req := &TestRequest{
		Method: "GET",
		Path:   server.URL,
	}
	gen := &MockGenerator{req: req}

	// Буферизированный канал для записи результатов.
	results := make(chan CallResult, 1000)
	var wg sync.WaitGroup

	// Запуск воркера.
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)

	go RunVirtualUser(ctx, attacker, gen, results, &wg)

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
