package load

import (
	"context"
	"sync"
)

// RequestGenerator - интерфейс генератора, предоставляет метод для получения следующего для выполения запроса.
type RequestGenerator interface {
	Next() *TestRequest // Возвращаетс следующий запрос или nil.
}

// RunVirtualUser - функция запускающая
func RunVirtualUser(
	ctx context.Context,
	attacker *Attacker,
	generator RequestGenerator,
	results chan<- CallResult,
	wg *sync.WaitGroup,
) {
	// Закрываем waitgroup.
	defer wg.Done()
	for {
		// Проверка отмены контекста (остановка теста).
		select {
		case <-ctx.Done():
			return
		default:
		}
		// Получение следующего запроса.
		request := generator.Next()
		if request == nil {
			// Если генератор вернул nil, значит сценарий исчерпан.
			return
		}
		result := attacker.Shoot(*request)
		select {
		// Запись результата.
		case results <- result:
		case <-ctx.Done():
			// Тест остановлен, при записи результата.
			// Выход, результат теряется.
			return
		}

	}
}
