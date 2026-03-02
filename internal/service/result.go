package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sibhellyx/tester/internal/models"
	"github.com/sibhellyx/tester/internal/processor"
	"github.com/sibhellyx/tester/internal/report"
)

// TestResultsRepositoryInterface - интерфейс репозитория результатов.
type TestResultsRepositoryInterface interface {
	GetCallResults(ctx context.Context, runID string) ([]models.CallResult, error)
	GetRunWithScenario(ctx context.Context, runID string) (*models.TestRun, *models.TestScenario, error)
	GetListWithStatus(ctx context.Context) ([]models.TestWithStatus, error)
}

// TestResultsService реализует бизнес-логику получения результатов.
// Данные уже в БД — ResultProcessor используется только для вычисления метрик
// поверх загруженных из БД CallResult.
type TestResultsService struct {
	logger          *slog.Logger
	repository      TestResultsRepositoryInterface
	resultProcessor processor.ResultProcessorInterface
	// Директория для хранения сгенерированных файлов отчётов.
	reportsDir string
}

// NewTestResultsService - конструктор.
func NewTestResultsService(
	logger *slog.Logger,
	repository TestResultsRepositoryInterface,
	resultProcessor processor.ResultProcessorInterface,
	reportsDir string,
) *TestResultsService {
	return &TestResultsService{
		logger:          logger,
		repository:      repository,
		resultProcessor: resultProcessor,
		reportsDir:      reportsDir,
	}
}

// GetReport возвращает полный отчёт по запуску.
func (s *TestResultsService) GetReport(ctx context.Context, runID string) (*models.TestReport, error) {
	run, scenario, results, err := s.loadRunData(ctx, runID)
	if err != nil {
		return nil, err
	}

	metrics := s.resultProcessor.ComputeMetrics(results)
	perRequest := s.resultProcessor.ComputePerRequest(results)

	builder := report.NewReportBuilder(*run, *scenario, metrics, perRequest, results)
	r := builder.BuildReport()
	// Нормализация длительности в ответе.
	r.Summary = normalizeMetrics(r.Summary)
	for name, m := range r.PerRequest {
		r.PerRequest[name] = normalizeMetrics(m)
	}
	return &r, nil
}

// GetChartData возвращает только данные для графиков (без полного отчёта).
// Полезно для lazy-загрузки графиков на фронте отдельным запросом.
func (s *TestResultsService) GetChartData(ctx context.Context, runID string) ([]models.ChartData, error) {
	run, scenario, results, err := s.loadRunData(ctx, runID)
	if err != nil {
		return nil, err
	}

	metrics := s.resultProcessor.ComputeMetrics(results)
	perRequest := s.resultProcessor.ComputePerRequest(results)

	builder := report.NewReportBuilder(*run, *scenario, metrics, perRequest, results)
	return builder.BuildChartData(), nil
}

// GenerateReportFile генерирует CSV-файл отчёта и возвращает путь к нему.
// Файл сохраняется в reportsDir и доступен для скачивания через хендлер.
func (s *TestResultsService) GenerateReportFile(ctx context.Context, runID string) (string, error) {
	run, scenario, results, err := s.loadRunData(ctx, runID)
	if err != nil {
		return "", err
	}

	metrics := s.resultProcessor.ComputeMetrics(results)
	perRequest := s.resultProcessor.ComputePerRequest(results)

	builder := report.NewReportBuilder(*run, *scenario, metrics, perRequest, results)
	testReport := builder.BuildReport()

	// Генерируем CSV.
	csvData, err := buildCSV(testReport)
	if err != nil {
		return "", fmt.Errorf("build csv: %w", err)
	}

	// Сохраняем файл.
	if err = os.MkdirAll(s.reportsDir, 0o755); err != nil {
		return "", fmt.Errorf("create reports dir: %w", err)
	}

	filename := fmt.Sprintf("report_%s_%s.csv", runID[:8], time.Now().Format("20060102_150405"))
	filePath := filepath.Join(s.reportsDir, filename)

	if err = os.WriteFile(filePath, csvData, 0o644); err != nil {
		s.logger.Error("Failed to write report file", slog.String("path", filePath), slog.String("error", err.Error()))
		return "", fmt.Errorf("write report file: %w", err)
	}

	s.logger.Info("Report file generated", slog.String("path", filePath), slog.String("run_id", runID))
	return filePath, nil
}

// GetListWithStatus возвращает все запуски с их сценариями и статусами.
func (s *TestResultsService) GetListWithStatus(ctx context.Context) ([]models.TestWithStatus, error) {
	list, err := s.repository.GetListWithStatus(ctx)
	if err != nil {
		s.logger.Error("GetListWithStatus failed", slog.String("error", err.Error()))
		return nil, ErrRepoError
	}
	if list == nil {
		list = []models.TestWithStatus{}
	}
	return list, nil
}

// loadRunData - общий хелпер: загружает run, scenario и call_results из БД.
func (s *TestResultsService) loadRunData(ctx context.Context, runID string) (
	*models.TestRun, *models.TestScenario, []models.CallResult, error,
) {
	run, scenario, err := s.repository.GetRunWithScenario(ctx, runID)
	if err != nil {
		s.logger.Error("GetRunWithScenario failed", slog.String("run_id", runID), slog.String("error", err.Error()))
		return nil, nil, nil, ErrRepoError
	}
	if run == nil {
		return nil, nil, nil, ErrRunNotFound
	}

	results, err := s.repository.GetCallResults(ctx, runID)
	if err != nil {
		s.logger.Error("GetCallResults failed", slog.String("run_id", runID), slog.String("error", err.Error()))
		return nil, nil, nil, ErrRepoError
	}

	return run, scenario, results, nil
}

// buildCSV строит CSV-файл из отчёта.
// Структура: секция Summary + секция Per-Request + сырые результаты.
func buildCSV(r models.TestReport) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// --- Заголовок ---
	_ = w.Write([]string{"Report for run", r.RunID})
	_ = w.Write([]string{"Scenario", r.ScenarioName})
	_ = w.Write([]string{"Start", r.StartTime.Format(time.RFC3339)})
	_ = w.Write([]string{"End", r.EndTime.Format(time.RFC3339)})
	_ = w.Write([]string{})

	// --- Summary ---
	_ = w.Write([]string{"=== SUMMARY ==="})
	_ = w.Write([]string{"Metric", "Value"})
	m := r.Summary
	_ = w.Write([]string{"total_requests", strconv.Itoa(m.TotalRequests)})
	_ = w.Write([]string{"success_count", strconv.Itoa(m.SuccessCount)})
	_ = w.Write([]string{"error_count", strconv.Itoa(m.ErrorCount)})
	_ = w.Write([]string{"error_rate", fmt.Sprintf("%.4f", m.ErrorRate)})
	_ = w.Write([]string{"rps", fmt.Sprintf("%.2f", m.RPS)})
	_ = w.Write([]string{"avg_latency_ms", strconv.FormatInt(m.AvgLatency.Milliseconds(), 10)})
	_ = w.Write([]string{"min_latency_ms", strconv.FormatInt(m.MinLatency.Milliseconds(), 10)})
	_ = w.Write([]string{"max_latency_ms", strconv.FormatInt(m.MaxLatency.Milliseconds(), 10)})
	_ = w.Write([]string{"p50_ms", strconv.FormatInt(m.P50.Milliseconds(), 10)})
	_ = w.Write([]string{"p90_ms", strconv.FormatInt(m.P90.Milliseconds(), 10)})
	_ = w.Write([]string{"p95_ms", strconv.FormatInt(m.P95.Milliseconds(), 10)})
	_ = w.Write([]string{"p99_ms", strconv.FormatInt(m.P99.Milliseconds(), 10)})

	_ = w.Write([]string{"total_bytes_in", strconv.FormatInt(m.TotalBytesIn, 10)})
	_ = w.Write([]string{"total_bytes_out", strconv.FormatInt(m.TotalBytesOut, 10)})
	_ = w.Write([]string{})

	// --- Per-Request ---
	_ = w.Write([]string{"=== PER REQUEST ==="})
	_ = w.Write([]string{
		"request", "total", "success", "errors", "error_rate",
		"avg_ms", "p50_ms", "p95_ms", "p99_ms", "max_ms", "rps",
	})
	for name, rm := range r.PerRequest {
		_ = w.Write([]string{
			name,
			strconv.Itoa(rm.TotalRequests),
			strconv.Itoa(rm.SuccessCount),
			strconv.Itoa(rm.ErrorCount),
			fmt.Sprintf("%.4f", rm.ErrorRate),
			strconv.FormatInt(rm.AvgLatency.Milliseconds(), 10),
			strconv.FormatInt(rm.P50.Milliseconds(), 10),
			strconv.FormatInt(rm.P95.Milliseconds(), 10),
			strconv.FormatInt(rm.P99.Milliseconds(), 10),
			strconv.FormatInt(rm.MaxLatency.Milliseconds(), 10),
			fmt.Sprintf("%.2f", rm.RPS),
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// normilizeMetrics - вспомогательная функция для возврата длительности в милисекундах вместо наносекунд.
func normalizeMetrics(m models.Metrics) models.Metrics {
	m.P50 = m.P50 / time.Millisecond
	m.P90 = m.P90 / time.Millisecond
	m.P95 = m.P95 / time.Millisecond
	m.P99 = m.P99 / time.Millisecond
	m.MinLatency = m.MinLatency / time.Millisecond
	m.MaxLatency = m.MaxLatency / time.Millisecond
	m.AvgLatency = m.AvgLatency / time.Millisecond
	return m
}
