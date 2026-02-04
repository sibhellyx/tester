package load

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TestRequest - структура описывающая запрос, который необходимо выполнить.
type TestRequest struct {
	Name                string            `json:"name"`           // "Login", "GetUsers".
	Method              string            `json:"method"`         // GET, POST, PUT, DELETE.
	Path                string            `json:"endpoint"`       // "/api/v1/users".
	Headers             map[string]string `json:"headers"`        // "Content-Type": "application/json".
	Body                string            `json:"body"`           // JSON payload.
	Weight              int               `json:"probability"`    // Вероятность выполнения (в %).
	ExpectedStatusCodes []int             `json:"expected_codes"` // [401, 404] Ожидаемые коды.
}

// CallResult - Результат одного конкретного запроса.
type CallResult struct {
	RequestName string        `json:"request_name"` // Название выполненного запроса.
	Timestamp   time.Time     `json:"timestamp"`    // Время начала запроса.
	Duration    time.Duration `json:"duration"`     // Сколько длился запрос (Latency).
	Status      int           `json:"status"`       // HTTP код (200, 500, etc).
	Error       string        `json:"error"`        // Текст ошибки (если статус 0 или сетевая ошибка).
	BytesOut    int64         `json:"bytes_out"`    // Размер отправленных данных (для подсчета пропускной способности).
	BytesIn     int64         `json:"bytes_in"`     // Размер полученных данных.
}

// Attacker - структура, которая обеспечивает выполнение действия(нагрузки).
// Данная структура будет обеспечивать выполнение запроса, по заданным критериям.
type Attacker struct {
	client *http.Client //http client для выполнения запроса
}

// NewAttacker - функция для инициализации инструмента для выполнения запроса.
// Получает на вход timeout - максимальное время для выполнения запроса
func NewAttacker(timeout time.Duration) *Attacker {
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}

	return &Attacker{
		client: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}
}

// Shoot - функция выполняющая запрос.
func (a *Attacker) Shoot(requestModel TestRequest) CallResult {
	// Подготовка к выполнению запроса.
	// Подготовка тела запроса(при наличии) и подсчет отправленных данных.
	// Reader для создания request.
	var body io.Reader
	// int64 для подсчета количества отправленных данных.
	var bytesOut int64
	if requestModel.Body != "" {
		body = bytes.NewBufferString(requestModel.Body)
		bytesOut += int64(len(requestModel.Body))
	}
	// Подсчет байтов RequestLine.
	bytesOut += int64(len(requestModel.Method) + len(requestModel.Path) + 12) // 12 = пробелы + "HTTP/1.1\r\n"
	// Создание http.Request.
	request, err := http.NewRequest(requestModel.Method, requestModel.Path, body)
	if err != nil {
		return CallResult{
			Status: 0,
			Error:  "invalid request: " + err.Error(),
		}
	}
	// Подготовка заголовков.
	for headerKey, headerValue := range requestModel.Headers {
		request.Header.Set(headerKey, headerValue)
		bytesOut += int64(len(headerKey) + len(headerValue) + 4) // 4 = ": " + "\r\n"
	}
	// Host заголовок хранится отдельно.
	bytesOut += int64(len("Host") + len(request.Host) + 4)

	// Пустая строка, отделяющая заголовки от тела ("\r\n")
	bytesOut += 2
	// Выполнение запроса и получение длительности выполнения.
	start := time.Now()
	// Выполнение запроса.
	response, err := a.client.Do(request)
	// Итоговое время выполнения запроса.
	duration := time.Since(start)
	// Подготовка результата выполнения запроса.
	result := CallResult{
		Timestamp:   start,
		Duration:    duration,
		RequestName: requestModel.Name,
	}
	// Проверка сетевых ошибок при выполнение запроса.
	if err != nil {
		result.Error = err.Error()
		result.Status = 0
		return result
	}
	// Закрытие чтения тела запроса.
	defer response.Body.Close()
	// Чтение тела ответа, иначе потеря TCP-соединения.
	written, _ := io.Copy(io.Discard, response.Body)
	// Фиксирование результата выполнения запроса.
	result.BytesIn = written
	result.BytesOut = bytesOut
	result.Status = response.StatusCode
	// Валидация статус кодов и установка ошибок.
	err = validateStatus(response.StatusCode, requestModel.ExpectedStatusCodes)
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

// validateStatus - функция для валидации статуса и установки ошибки
func validateStatus(code int, expected []int) error {
	// Если пользователь не указал ожидания, используем стандарт (2xx - ок).
	if len(expected) == 0 {
		if code >= 200 && code < 300 {
			return nil
		}
		return fmt.Errorf("unexpected status: %d", code)
	}
	// Если список задан, ищем код в нем.
	for _, e := range expected {
		if code == e {
			return nil
		}
	}
	return fmt.Errorf("status %d not in expected list %v", code, expected)
}
