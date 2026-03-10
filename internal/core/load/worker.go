package load

import (
	"context"
	"sync"

	"github.com/sibhellyx/tester/internal/models"
)

// RequestGeneratorInterface - интерфейс генератора, предоставляет метод для получения следующего запроса.
type RequestGeneratorInterface interface {
	Next() *models.TestRequest
}

// RunVirtualUser выполняет бесконечный цикл запросов от имени одного виртуального пользователя.
// Завершается когда:
//   - ctx отменён (этап завершился, тест остановлен, или индивидуальный kill)
//   - генератор вернул nil (теоретически недостижимо для WeightedGenerator)
//
// wg.Done() вызывается через defer — гарантированно даже при панике.
// Вызывающий код (userPool.Spawn) закрывает done-канал строго после возврата из этой функции.
func RunVirtualUser(
	ctx context.Context,
	wg *sync.WaitGroup,
	generator RequestGeneratorInterface,
	attacker AttackerToolInterface,
	results chan<- models.CallResult,
) {
	defer wg.Done()

	for {
		// Проверяем отмену контекста перед каждым запросом.
		// Это позволяет быстро среагировать на остановку без выполнения лишнего запроса.
		select {
		case <-ctx.Done():
			return
		default:
		}

		request := generator.Next()
		if request == nil {
			return
		}

		result := attacker.Shoot(*request)

		// При записи результата тоже проверяем ctx —
		// канал results может быть переполнен, и мы не хотим зависнуть навсегда.
		select {
		case results <- result:
		case <-ctx.Done():
			return
		}
	}
}
