package models

import "time"

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
