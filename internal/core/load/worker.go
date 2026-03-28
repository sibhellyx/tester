package load

import (
	"context"
	"sync"

	"github.com/sibhellyx/tester/internal/models"
)

// stageAwareGenerator — интерфейс генератора, который требуется RunVirtualUser.
type stageAwareGenerator interface {
	Next() *models.TestRequest
	CurrentStageID() int
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
	generator stageAwareGenerator,
	attacker AttackerToolInterface,
	results chan<- models.CallResult,
) {
	defer wg.Done()

	for {
		// Проверяем отмену контекста перед каждым запросом.
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
		result.StageID = generator.CurrentStageID()

		// При записи результата тоже проверяем ctx.
		select {
		case results <- result:
		case <-ctx.Done():
			return
		}
	}
}
