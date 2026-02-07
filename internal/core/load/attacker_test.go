package load

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// TestAttacker_Shoot_Success проверяет успешный сценарий:
// 1. Корректность подсчета входящих байт (BytesIn).
// 2. Отсутствие ошибок при статусе 200.
// 3. Расчет BytesOut (базовая проверка).
func TestAttacker_Shoot_Success(t *testing.T) {
	// Поднимаем тестовый сервер, который отдает "pong".
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong")) // 4 байта.
	}))
	defer server.Close()

	attacker := NewAttacker(1 * time.Second)

	reqModel := models.TestRequest{
		Name:   "Ping Check",
		Method: "GET",
		Path:   server.URL, // Используем URL тестового сервера.
		Body:   "",
	}

	result := attacker.Shoot(reqModel)

	// Проверки.
	if result.Error != "" {
		t.Fatalf("Unexpected error: %s", result.Error)
	}

	if result.Status != 200 {
		t.Errorf("Expected status 200, got %d", result.Status)
	}

	if result.BytesIn != 4 {
		t.Errorf("Expected 4 bytes in (pong), got %d", result.BytesIn)
	}

	if result.BytesOut <= 0 {
		t.Errorf("BytesOut should be calculated, got %d", result.BytesOut)
	}

	if result.Duration <= 0 {
		t.Errorf("Duration should be positive")
	}
}

// TestAttacker_Shoot_BytesOutLogic детально проверяет математику подсчета исходящих байт.
func TestAttacker_Shoot_BytesOutLogic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	attacker := NewAttacker(1 * time.Second)

	// Параметры запроса.
	method := "POST"
	body := "12345" // 5 байт.
	headerKey := "X-Test"
	headerVal := "Value" // len("X-Test") + len("Value") + 4 = 6 + 5 + 4 = 15 байт.

	reqModel := models.TestRequest{
		Method: method,
		Path:   server.URL,
		Headers: map[string]string{
			headerKey: headerVal,
		},
		Body: body,
	}

	// Рассчитаем ожидаемый размер вручную, как в реализации Shoot.
	// 1. Body.
	var expectedBytes int64 = int64(len(body))

	// 2. Request Line: Method + URL + 12.
	expectedBytes += int64(len(method) + len(server.URL) + 12)

	// 3. Custom Header: Key + Value + 4.
	expectedBytes += int64(len(headerKey) + len(headerVal) + 4)

	// 4. Host Header: "Host" + HostUrl + 4.
	// http.NewRequest автоматически парсит Host из URL.
	// server.URL выглядит как http://127.0.0.1:54321.
	host := strings.TrimPrefix(server.URL, "http://")
	expectedBytes += int64(len("Host") + len(host) + 4)

	// 5. Separator.
	expectedBytes += 2

	result := attacker.Shoot(reqModel)

	if result.BytesOut != expectedBytes {
		t.Errorf("BytesOut calculation mismatch. Expected %d, got %d", expectedBytes, result.BytesOut)
	}
}

// TestAttacker_Shoot_DefaultValidation проверяет стандартную валидацию (без expected_codes).
// Ожидание: 2xx - ОК, 4xx/5xx - Ошибка.
func TestAttacker_Shoot_DefaultValidation(t *testing.T) {
	attacker := NewAttacker(1 * time.Second)

	tests := []struct {
		name          string
		serverStatus  int
		shouldBeError bool
	}{
		{"Status 200 OK", 200, false},
		{"Status 201 Created", 201, false},
		{"Status 299 OK", 299, false},
		{"Status 404 Not Found", 404, true},
		{"Status 500 Server Error", 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			req := models.TestRequest{Method: "GET", Path: server.URL}
			result := attacker.Shoot(req)

			if tt.shouldBeError {
				if result.Error == "" {
					t.Errorf("Expected error for status %d, but got nil", tt.serverStatus)
				}
			} else {
				if result.Error != "" {
					t.Errorf("Unexpected error for status %d: %s", tt.serverStatus, result.Error)
				}
			}
		})
	}
}

// TestAttacker_Shoot_CustomValidation проверяет работу поля ExpectedStatusCodes.
func TestAttacker_Shoot_CustomValidation(t *testing.T) {
	attacker := NewAttacker(1 * time.Second)

	// Сценарий 1: Ждем 404, и сервер отдает 404. Это УСПЕХ.
	t.Run("Expect 404 -> Got 404 (Success)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer server.Close()

		req := models.TestRequest{
			Method:              "GET",
			Path:                server.URL,
			ExpectedStatusCodes: []int{404},
		}

		result := attacker.Shoot(req)
		if result.Error != "" {
			t.Errorf("Expected success when getting expected 404, got error: %s", result.Error)
		}
		if result.Status != 404 {
			t.Errorf("Expected status 404, got %d", result.Status)
		}
	})

	// Сценарий 2: Ждем 200, но сервер отдает 500. Это ОШИБКА.
	t.Run("Expect 200 -> Got 500 (Error)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}))
		defer server.Close()

		req := models.TestRequest{
			Method:              "GET",
			Path:                server.URL,
			ExpectedStatusCodes: []int{200},
		}

		result := attacker.Shoot(req)
		if result.Error == "" {
			t.Error("Expected error because 500 is not in [200], but got success")
		}
	})

	// Сценарий 3: Ждем 201, сервер отдает 200. Это ОШИБКА (строгое соответствие).
	t.Run("Expect 201 -> Got 200 (Error)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer server.Close()

		req := models.TestRequest{
			Method:              "POST",
			Path:                server.URL,
			ExpectedStatusCodes: []int{201},
		}

		result := attacker.Shoot(req)
		if result.Error == "" {
			t.Error("Expected error because 200 is not in [201]")
		}
	})
}

// TestAttacker_Shoot_Timeout проверяет обработку сетевых таймаутов.
func TestAttacker_Shoot_Timeout(t *testing.T) {
	// "медленный" сервер.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Спим 100мс
		w.WriteHeader(200)
	}))
	defer server.Close()

	// Attacker с таймаутом всего 10мс.
	attacker := NewAttacker(10 * time.Millisecond)

	req := models.TestRequest{Method: "GET", Path: server.URL}
	result := attacker.Shoot(req)

	if result.Status != 0 {
		t.Errorf("Expected status 0 on timeout, got %d", result.Status)
	}
	if result.Error == "" {
		t.Error("Expected timeout error, got nil")
	}
	// Проверяем, что в тексте ошибки есть упоминание timeout или context deadline.
	if !strings.Contains(result.Error, "Timeout") && !strings.Contains(result.Error, "deadline") && !strings.Contains(result.Error, "timeout") {
		t.Errorf("Error message should contain timeout info, got: %s", result.Error)
	}
}

// TestAttacker_Shoot_InvalidRequest проверяет ошибку создания запроса (например, кривой метод).
func TestAttacker_Shoot_InvalidRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	attacker := NewAttacker(1 * time.Second)
	// Неккоректный метод запроса.
	req := models.TestRequest{
		Method: "GET/ERROR",
		Path:   server.URL,
	}

	result := attacker.Shoot(req)

	if result.Status != 0 {
		t.Errorf("Expected status 0, got %d", result.Status)
	}
	if result.Error == "" {
		t.Error("Expected error on invalid request, got nil")
	}
	if !strings.Contains(result.Error, "invalid request") {
		t.Errorf("Expected 'invalid request' error prefix, got: %s", result.Error)
	}
}
