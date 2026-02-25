package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sibhellyx/tester/internal/models"
	"github.com/sibhellyx/tester/internal/service"
)

// TestRunServiceInterface - интерфейс для взаимодействия хендлера с сервисом.
type TestRunServiceInterface interface {
	StartTest(ctx context.Context, scenarioID string) (string, error)
	StopTest(runID string) error
	GetTestStatus(ctx context.Context, runID string) (*models.TestWithStatus, error)
	ListRuns(ctx context.Context, scenarioID string) ([]models.TestRun, error)
	GetSummary(ctx context.Context, runID string) (*models.TestRunSummary, error)
}

// TestRunHandler обрабатывает HTTP-запросы управления тестовыми прогонами.
type TestRunHandler struct {
	logger  *slog.Logger
	service TestRunServiceInterface
}

// NewTestRunHandler - конструктор.
func NewTestRunHandler(logger *slog.Logger, service TestRunServiceInterface) *TestRunHandler {
	return &TestRunHandler{
		logger:  logger,
		service: service,
	}
}

// StartTest godoc
// @Summary      Запустить тест
// @Description  Запускает нагрузочное тестирование по ID сценария
// @Tags         runs
// @Produce      json
// @Param        id  path  string  true  "ID сценария"
// @Success      202  {object}  map[string]string  "id запуска"
// @Failure      404  {object}  map[string]string  "Сценарий не найден"
// @Failure      500  {object}  map[string]string  "Внутренняя ошибка"
// @Router       /api/v1/scenarios/{id}/runs [post]
func (h *TestRunHandler) StartTest(c *gin.Context) {
	scenarioID := c.Param("id")
	if scenarioID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scenario id is required"})
		return
	}

	runID, err := h.service.StartTest(c.Request.Context(), scenarioID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrScenarioNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "scenario not found"})
		case errors.Is(err, service.ErrRunAlreadyActive):
			c.JSON(http.StatusConflict, gin.H{"error": "test is already running"})
		default:
			h.logger.Error("StartTest failed", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	// 202 Accepted — тест запущен асинхронно, результат будет доступен позже.
	c.JSON(http.StatusAccepted, gin.H{"run_id": runID})
}

// StopTest godoc
// @Summary      Остановить тест
// @Description  Принудительно останавливает активный тестовый прогон
// @Tags         runs
// @Produce      json
// @Param        run_id  path  string  true  "ID запуска"
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string  "Запуск не найден"
// @Failure      409  {object}  map[string]string  "Тест уже не активен"
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/runs/{run_id}/stop [post]
func (h *TestRunHandler) StopTest(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_id is required"})
		return
	}

	if err := h.service.StopTest(runID); err != nil {
		switch {
		case errors.Is(err, service.ErrRunNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		case errors.Is(err, service.ErrRunNotActive):
			c.JSON(http.StatusConflict, gin.H{"error": "test run is not active"})
		default:
			h.logger.Error("StopTest failed", slog.String("run_id", runID), slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// GetStatus godoc
// @Summary      Статус запуска
// @Description  Возвращает текущий статус тестового прогона вместе со сценарием
// @Tags         runs
// @Produce      json
// @Param        run_id  path  string  true  "ID запуска"
// @Success      200  {object}  models.TestWithStatus
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/runs/{run_id} [get]
func (h *TestRunHandler) GetStatus(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_id is required"})
		return
	}

	status, err := h.service.GetTestStatus(c.Request.Context(), runID)
	if err != nil {
		if errors.Is(err, service.ErrRunNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, status)
}

// ListRuns godoc
// @Summary      История запусков
// @Description  Возвращает список всех запусков для данного сценария
// @Tags         runs
// @Produce      json
// @Param        id  path  string  true  "ID сценария"
// @Success      200  {array}   models.TestRun
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/scenarios/{id}/runs [get]
func (h *TestRunHandler) ListRuns(c *gin.Context) {
	scenarioID := c.Param("id")
	if scenarioID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scenario id is required"})
		return
	}

	runs, err := h.service.ListRuns(c.Request.Context(), scenarioID)
	if err != nil {
		h.logger.Error("ListRuns failed", slog.String("scenario_id", scenarioID), slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch runs"})
		return
	}

	if runs == nil {
		runs = []models.TestRun{}
	}

	c.JSON(http.StatusOK, runs)
}

// GetSummary godoc
// @Summary      Итоги запуска
// @Description  Возвращает агрегированную статистику завершённого запуска
// @Tags         runs
// @Produce      json
// @Param        run_id  path  string  true  "ID запуска"
// @Success      200  {object}  models.TestRunSummary
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/runs/{run_id}/summary [get]
func (h *TestRunHandler) GetSummary(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_id is required"})
		return
	}

	summary, err := h.service.GetSummary(c.Request.Context(), runID)
	if err != nil {
		if errors.Is(err, service.ErrRunNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, summary)
}
