package load

import (
	"fmt"
	"math/rand/v2"
	"sync"
)

// WeightedGenerator реализует выбор запроса на основе вероятностей (весов).
type WeightedGenerator struct {
	requests          []TestRequest // Запросы для выполнения.
	cumulativeWeights []int         // Накопительные веса для быстрого выбора.
	totalWeight       int           // Общая сумма весов.
	mu                sync.Mutex    // Для защиты генератора случайных чисел.
	randomGenerator   *rand.Rand    // Локальный генератор.
}

// NewWeightedGenerator создает генератор с валидацией и нормализацией весов.
func NewWeightedGenerator(requests []TestRequest) (*WeightedGenerator, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("requests list cannot be empty")
	}

	totalWeight := 0
	// Проверка суммы весов (вероятностей) выполнения запросов.
	for _, request := range requests {
		if request.Weight < 0 {
			return nil, fmt.Errorf("request '%s' has negative weight: %d", request.Name, request.Weight)
		}
		totalWeight += request.Weight
	}

	// Валидация: сумма весов должна быть больше 0.
	if totalWeight <= 0 {
		return nil, fmt.Errorf("total weight is 0, at least one request must have weight > 0")
	}

	// Строим массив накопительных весов для эффективного выбора.
	// Пример: веса [30, 50, 20] → накопительные [30, 80, 100]
	cumulativeWeights := make([]int, len(requests))
	cumulative := 0
	for i, request := range requests {
		cumulative += request.Weight
		cumulativeWeights[i] = cumulative
	}

	return &WeightedGenerator{
		requests:          requests,
		cumulativeWeights: cumulativeWeights,
		totalWeight:       totalWeight,
		randomGenerator:   rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}, nil
}

// Next возвращает следующий запрос на основе весов (вероятностей).
// Использует алгоритм взвешенного случайного выбора.
func (g *WeightedGenerator) Next() *TestRequest {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Генерируем случайное число от 1 до totalWeight (включительно).
	randomValue := g.randomGenerator.IntN(g.totalWeight) + 1

	// Бинарный поиск в накопительных весах.
	// Находим первый индекс, где cumulativeWeights[i] >= randomValue.
	// Выбор происходит - следующим путем:
	// Пример: есть три запросы имеющие веса [30, 50, 20], их накопительные веса будут соответственно [30, 80, 100]
	// randomValue в промежутке от 1 до 30 - отдаем первый запрос
	// randomValue в промежутке от 31 до 80 - отдаем второй запрос
	// randomValue в промежутке от 81 до 100 - отдаем третий запрос запрос
	for i, cumulativeWeight := range g.cumulativeWeights {
		if randomValue <= cumulativeWeight {
			return &g.requests[i]
		}
	}

	// Теоретически недостижимо, но для безопасности возвращаем последний.
	return &g.requests[len(g.requests)-1]
}
