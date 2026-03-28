package processor

import (
	"math"
	"sort"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// ResultProcessorInterface - интерфейс для вычисления метрик.
type ResultProcessorInterface interface {
	ComputeMetrics(results []models.CallResult) models.Metrics
	ComputePerRequest(results []models.CallResult) map[string]models.Metrics
	ComputePerStage(results []models.CallResult) map[int]models.Metrics
}

// ResultProcessor вычисляет метрики по срезу результатов,
type ResultProcessor struct{}

// NewResultProcessor - конструктор.
func NewResultProcessor() *ResultProcessor {
	return &ResultProcessor{}
}

// ComputeMetrics считает агрегированные метрики по всем результатам.
func (p *ResultProcessor) ComputeMetrics(results []models.CallResult) models.Metrics {
	if len(results) == 0 {
		return models.Metrics{}
	}

	var (
		totalBytesIn  int64
		totalBytesOut int64
		successCount  int
		errorCount    int
		latencies     = make([]time.Duration, 0, len(results))
	)

	// Временной диапазон для вычисления RPS.
	minTime := results[0].Timestamp
	maxTime := results[0].Timestamp

	for _, r := range results {
		latencies = append(latencies, r.Duration)
		totalBytesIn += r.BytesIn
		totalBytesOut += r.BytesOut

		// Успешным считаем любой ответ с кодом 200-399.
		if r.Status >= 200 && r.Status < 400 {
			successCount++
		} else {
			errorCount++
		}

		if r.Timestamp.Before(minTime) {
			minTime = r.Timestamp
		}
		if r.Timestamp.After(maxTime) {
			maxTime = r.Timestamp
		}
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	total := len(results)
	elapsed := maxTime.Sub(minTime).Seconds()

	var rps float64
	if elapsed > 0 {
		rps = float64(total) / elapsed
	}

	var errorRate float64
	if total > 0 {
		errorRate = float64(errorCount) / float64(total)
	}

	return models.Metrics{
		TotalRequests: total,
		SuccessCount:  successCount,
		ErrorCount:    errorCount,
		ErrorRate:     math.Round(errorRate*10000) / 10000, // 4 знака
		P50:           percentile(latencies, 0.50),
		P90:           percentile(latencies, 0.90),
		P95:           percentile(latencies, 0.95),
		P99:           percentile(latencies, 0.99),
		MinLatency:    latencies[0],
		MaxLatency:    latencies[len(latencies)-1],
		AvgLatency:    avgLatency(latencies),
		RPS:           math.Round(rps*100) / 100,
		TotalBytesIn:  totalBytesIn,
		TotalBytesOut: totalBytesOut,
	}
}

// ComputePerRequest считает метрики отдельно для каждого запроса (по имени).
func (p *ResultProcessor) ComputePerRequest(results []models.CallResult) map[string]models.Metrics {
	// Группируем результаты по имени запроса.
	grouped := make(map[string][]models.CallResult)
	for _, r := range results {
		grouped[r.RequestName] = append(grouped[r.RequestName], r)
	}

	perRequest := make(map[string]models.Metrics, len(grouped))
	for name, group := range grouped {
		perRequest[name] = p.ComputeMetrics(group)
	}
	return perRequest
}

// ComputePerStage считает метрики отдельно для каждого этапа теста.
func (p *ResultProcessor) ComputePerStage(results []models.CallResult) map[int]models.Metrics {
	grouped := make(map[int][]models.CallResult)
	for _, r := range results {
		grouped[r.StageID] = append(grouped[r.StageID], r)
	}
	perStage := make(map[int]models.Metrics, len(grouped))
	for id, group := range grouped {
		perStage[id] = p.ComputeMetrics(group)
	}
	return perStage
}

// percentile возвращает перцентиль из уже отсортированного слайса.
// p: 0.0 - 1.0
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	// Nearest-rank метод.
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// avgLatency считает среднее значение латентности.
func avgLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	return sum / time.Duration(len(latencies))
}
