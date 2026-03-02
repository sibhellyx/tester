package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sibhellyx/tester/internal/models"
	"github.com/sibhellyx/tester/internal/service"
)

// TestResultsServiceInterface - интерфейс сервиса результатов для хендлера.
type TestResultsServiceInterface interface {
	GetReport(ctx context.Context, runID string) (*models.TestReport, error)
	GetChartData(ctx context.Context, runID string) ([]models.ChartData, error)
	GenerateReportFile(ctx context.Context, runID string) (string, error)
	GetListWithStatus(ctx context.Context) ([]models.TestWithStatus, error)
}

// TestResultsHandler обрабатывает HTTP-запросы для получения результатов тестов.
type TestResultsHandler struct {
	logger  *slog.Logger
	service TestResultsServiceInterface
}

// NewTestResultsHandler - конструктор.
func NewTestResultsHandler(logger *slog.Logger, service TestResultsServiceInterface) *TestResultsHandler {
	return &TestResultsHandler{
		logger:  logger,
		service: service,
	}
}

// GetReport godoc
// @Summary      Полный отчёт по запуску
// @Description  Возвращает TestReport: summary-метрики, per-request метрики и данные графиков
// @Tags         results
// @Produce      json
// @Param        run_id  path  string  true  "ID запуска"
// @Success      200  {object}  models.TestReport
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/runs/{run_id}/report [get]
func (h *TestResultsHandler) GetReport(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_id is required"})
		return
	}

	rep, err := h.service.GetReport(c.Request.Context(), runID)
	if err != nil {
		h.handleServiceError(c, err, runID, "GetReport")
		return
	}

	c.JSON(http.StatusOK, rep)
}

// GetChartData godoc
// @Summary      Данные для графиков
// @Description  Возвращает только ChartData без полного отчёта (для отдельной загрузки графиков)
// @Tags         results
// @Produce      json
// @Param        run_id  path  string  true  "ID запуска"
// @Success      200  {array}   models.ChartData
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/runs/{run_id}/charts [get]
func (h *TestResultsHandler) GetChartData(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_id is required"})
		return
	}

	charts, err := h.service.GetChartData(c.Request.Context(), runID)
	if err != nil {
		h.handleServiceError(c, err, runID, "GetChartData")
		return
	}

	c.JSON(http.StatusOK, charts)
}

// GetReportFile godoc
// @Summary      Скачать отчёт (CSV)
// @Description  Генерирует CSV-файл отчёта и отдаёт его как вложение
// @Tags         results
// @Produce      text/csv
// @Param        run_id  path  string  true  "ID запуска"
// @Success      200  {file}    csv
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/runs/{run_id}/report/download [get]
func (h *TestResultsHandler) GetReportFile(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_id is required"})
		return
	}

	filePath, err := h.service.GenerateReportFile(c.Request.Context(), runID)
	if err != nil {
		h.handleServiceError(c, err, runID, "GenerateReportFile")
		return
	}

	filename := filepath.Base(filePath)

	// Отдаём файл как вложение — браузер/клиент скачает его.
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.File(filePath)
}

// GetListWithStatus godoc
// @Summary      Список запусков с результатами
// @Description  Возвращает все запуски со сценариями и статусами (дашборд)
// @Tags         results
// @Produce      json
// @Success      200  {array}   models.TestWithStatus
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/runs [get]
func (h *TestResultsHandler) GetListWithStatus(c *gin.Context) {
	list, err := h.service.GetListWithStatus(c.Request.Context())
	if err != nil {
		h.logger.Error("GetListWithStatus failed", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch runs"})
		return
	}

	c.JSON(http.StatusOK, list)
}

// handleServiceError - централизованная обработка ошибок сервисного слоя.
func (h *TestResultsHandler) handleServiceError(c *gin.Context, err error, runID, method string) {
	switch {
	case errors.Is(err, service.ErrRunNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
	default:
		h.logger.Error("Results service error",
			slog.String("method", method),
			slog.String("run_id", runID),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
