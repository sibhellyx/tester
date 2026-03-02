package report

import (
	"fmt"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

// ReportBuilderInterface - интерфейс построителя отчётов.
type ReportBuilderInterface interface {
	BuildReport() models.TestReport
	BuildChartData() []models.ChartData
}

// ReportBuilder собирает TestReport из метрик и сырых результатов.
type ReportBuilder struct {
	run        models.TestRun
	scenario   models.TestScenario
	metrics    models.Metrics
	perRequest map[string]models.Metrics
	results    []models.CallResult
}

// NewReportBuilder - конструктор.
func NewReportBuilder(
	run models.TestRun,
	scenario models.TestScenario,
	metrics models.Metrics,
	perRequest map[string]models.Metrics,
	results []models.CallResult,
) *ReportBuilder {
	return &ReportBuilder{
		run:        run,
		scenario:   scenario,
		metrics:    metrics,
		perRequest: perRequest,
		results:    results,
	}
}

// BuildReport - строит полный отчёт.
func (b *ReportBuilder) BuildReport() models.TestReport {
	charts := b.BuildChartData()

	startTime := time.Time{}
	endTime := time.Time{}
	if b.run.StartedAt != nil {
		startTime = *b.run.StartedAt
	}
	if b.run.FinishedAt != nil {
		endTime = *b.run.FinishedAt
	}

	return models.TestReport{
		RunID:        b.run.ID,
		ScenarioName: b.scenario.Name,
		StartTime:    startTime,
		EndTime:      endTime,
		Summary:      b.metrics,
		PerRequest:   b.perRequest,
		Charts:       charts,
	}
}

// BuildChartData - строит набор графиков для отчёта.
func (b *ReportBuilder) BuildChartData() []models.ChartData {
	if len(b.results) == 0 {
		return nil
	}

	charts := []models.ChartData{
		b.buildLatencyOverTime(),
		b.buildRPSOverTime(),
		b.buildErrorRateOverTime(),
		b.buildLatencyHistogram(),
		b.buildPerRequestLatency(),
	}

	return charts
}

// buildLatencyOverTime - средняя латентность по 5-секундным окнам.
func (b *ReportBuilder) buildLatencyOverTime() models.ChartData {
	buckets := b.groupByTimeBucket(5 * time.Second)

	labels := make([]string, 0, len(buckets))
	data := make([]float64, 0, len(buckets))

	for _, bucket := range buckets {
		var sum time.Duration
		for _, r := range bucket.results {
			sum += r.Duration
		}
		avg := float64(sum/time.Millisecond) / float64(len(bucket.results))

		labels = append(labels, bucket.label)
		data = append(data, roundFloat(avg, 2))
	}

	return models.ChartData{
		Type:   "line",
		Title:  "Latency over time (avg ms)",
		Labels: labels,
		Series: []models.Series{
			{Name: "Avg Latency (ms)", Data: data},
		},
	}
}

// buildRPSOverTime - количество запросов в секунду по 5-секундным окнам.
func (b *ReportBuilder) buildRPSOverTime() models.ChartData {
	buckets := b.groupByTimeBucket(5 * time.Second)
	windowSec := 5.0

	labels := make([]string, 0, len(buckets))
	data := make([]float64, 0, len(buckets))

	for _, bucket := range buckets {
		rps := float64(len(bucket.results)) / windowSec
		labels = append(labels, bucket.label)
		data = append(data, roundFloat(rps, 2))
	}

	return models.ChartData{
		Type:   "line",
		Title:  "RPS over time",
		Labels: labels,
		Series: []models.Series{
			{Name: "RPS", Data: data},
		},
	}
}

// buildErrorRateOverTime - процент ошибок по 5-секундным окнам.
func (b *ReportBuilder) buildErrorRateOverTime() models.ChartData {
	buckets := b.groupByTimeBucket(5 * time.Second)

	labels := make([]string, 0, len(buckets))
	data := make([]float64, 0, len(buckets))

	for _, bucket := range buckets {
		var errors int
		for _, r := range bucket.results {
			if r.Status == 0 || r.Status >= 400 {
				errors++
			}
		}
		rate := float64(errors) / float64(len(bucket.results)) * 100
		labels = append(labels, bucket.label)
		data = append(data, roundFloat(rate, 2))
	}

	return models.ChartData{
		Type:   "line",
		Title:  "Error rate over time (%)",
		Labels: labels,
		Series: []models.Series{
			{Name: "Error rate (%)", Data: data},
		},
	}
}

// buildLatencyHistogram - распределение латентностей по бакетам (гистограмма).
func (b *ReportBuilder) buildLatencyHistogram() models.ChartData {
	// Бакеты: 0-50ms, 50-100ms, 100-250ms, 250-500ms, 500ms-1s, 1s+
	edges := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
		1000 * time.Millisecond,
	}
	labels := []string{"0-50ms", "50-100ms", "100-250ms", "250-500ms", "500ms-1s", "1s+"}
	counts := make([]float64, len(labels))

	for _, r := range b.results {
		idx := len(edges) // дефолт — последний бакет "1s+"
		for i, edge := range edges {
			if r.Duration <= edge {
				idx = i
				break
			}
		}
		counts[idx]++
	}

	return models.ChartData{
		Type:   "bar",
		Title:  "Latency distribution",
		Labels: labels,
		Series: []models.Series{
			{Name: "Requests", Data: counts},
		},
	}
}

// buildPerRequestLatency - avg латентность по каждому типу запроса (bar chart).
func (b *ReportBuilder) buildPerRequestLatency() models.ChartData {
	labels := make([]string, 0, len(b.perRequest))
	avgData := make([]float64, 0, len(b.perRequest))
	p95Data := make([]float64, 0, len(b.perRequest))

	for name, m := range b.perRequest {
		labels = append(labels, name)
		avgData = append(avgData, float64(m.AvgLatency/time.Millisecond))
		p95Data = append(p95Data, float64(m.P95/time.Millisecond))
	}

	return models.ChartData{
		Type:   "bar",
		Title:  "Latency per request type (ms)",
		Labels: labels,
		Series: []models.Series{
			{Name: "Avg (ms)", Data: avgData},
			{Name: "P95 (ms)", Data: p95Data},
		},
	}
}

// --- вспомогательные типы и функции ---

type timeBucket struct {
	label   string
	results []models.CallResult
}

// groupByTimeBucket группирует результаты по временным окнам фиксированного размера.
func (b *ReportBuilder) groupByTimeBucket(window time.Duration) []timeBucket {
	if len(b.results) == 0 {
		return nil
	}

	// Находим начало теста.
	start := b.results[0].Timestamp
	for _, r := range b.results {
		if r.Timestamp.Before(start) {
			start = r.Timestamp
		}
	}

	// Определяем количество бакетов.
	end := b.results[0].Timestamp
	for _, r := range b.results {
		if r.Timestamp.After(end) {
			end = r.Timestamp
		}
	}
	totalDuration := end.Sub(start)
	numBuckets := int(totalDuration/window) + 1

	buckets := make([]timeBucket, numBuckets)
	for i := range buckets {
		t := start.Add(window * time.Duration(i))
		buckets[i].label = fmt.Sprintf("+%ds", int(window.Seconds())*i)
		_ = t
	}

	// Раскладываем результаты по бакетам.
	for _, r := range b.results {
		idx := int(r.Timestamp.Sub(start) / window)
		if idx >= numBuckets {
			idx = numBuckets - 1
		}
		buckets[idx].results = append(buckets[idx].results, r)
	}

	// Убираем пустые бакеты с конца (если тест закончился раньше).
	for len(buckets) > 0 && len(buckets[len(buckets)-1].results) == 0 {
		buckets = buckets[:len(buckets)-1]
	}

	return buckets
}

func roundFloat(val float64, precision int) float64 {
	ratio := 1.0
	for i := 0; i < precision; i++ {
		ratio *= 10
	}
	return float64(int(val*ratio+0.5)) / ratio
}