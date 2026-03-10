package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sibhellyx/tester/internal/core/chaos"
	"github.com/sibhellyx/tester/internal/models"
	"github.com/sibhellyx/tester/internal/service"
)

// TestManagementServiceInterface определяет бизнес-логику управления сценариями.
type TestManagementServiceInterface interface {
	CreateScenario(ctx context.Context, s models.TestScenario) (string, error)
	DeleteScenario(ctx context.Context, id string) error
	ListScenarios(ctx context.Context) ([]models.TestScenario, error)
	GetScenario(ctx context.Context, id string) (*models.TestScenario, error)
	UpdateScenario(ctx context.Context, s models.TestScenario) error
	ListContainers(ctx context.Context) ([]chaos.ContainerInfo, error)
}

// ScenarioHandler обрабатывает HTTP-запросы, связанные со сценариями.
type ScenarioHandler struct {
	logger  *slog.Logger
	service TestManagementServiceInterface
}

// NewScenarioHandler - конструктор хендлера.
func NewScenarioHandler(logger *slog.Logger, service TestManagementServiceInterface) *ScenarioHandler {
	return &ScenarioHandler{
		logger:  logger,
		service: service,
	}
}

// CreateScenario godoc
// @Summary      Создать сценарий
// @Description  Создает и сохраняет новый сценарий тестирования
// @Tags         scenarios
// @Accept       json
// @Produce      json
// @Param        input body models.TestScenario true "Тело сценария"
// @Success      201  {object}  map[string]string "id созданного сценария"
// @Failure      400  {object}  map[string]string "Ошибка валидации"
// @Failure      500  {object}  map[string]string "Внутренняя ошибка сервера"
// @Router       /api/v1/scenarios [post]
func (h *ScenarioHandler) CreateScenario(c *gin.Context) {
	var scenario models.TestScenario

	// Парсинг JSON.
	if err := c.ShouldBindJSON(&scenario); err != nil {
		h.logger.Error("Failed to bind JSON", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Вызов бизнес-логики.
	id, err := h.service.CreateScenario(c.Request.Context(), scenario)
	if err != nil {
		if errors.Is(err, service.ErrInvalidScenario) {
			// Ошибка валидации бизнес-логики -> 400.
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			// Любая другая ошибка (БД, сеть) -> 500.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	// Ответ.
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// GetScenario godoc
// @Summary      Получить сценарий
// @Description  Возвращает сценарий по его ID
// @Tags         scenarios
// @Produce      json
// @Param        id   path      string  true  "ID сценария"
// @Success      200  {object}  models.TestScenario
// @Failure      404  {object}  map[string]string "Сценарий не найден"
// @Failure      500  {object}  map[string]string "Внутренняя ошибка"
// @Router       /api/v1/scenarios/{id} [get]
func (h *ScenarioHandler) GetScenario(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	scenario, err := h.service.GetScenario(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrScenarioNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scenario not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	if scenario == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scenario not found"})
		return
	}

	c.JSON(http.StatusOK, scenario)
}

// ListScenarios godoc
// @Summary      Список сценариев
// @Description  Возвращает список всех доступных сценариев
// @Tags         scenarios
// @Produce      json
// @Success      200  {array}   models.TestScenario
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/scenarios [get]
func (h *ScenarioHandler) ListScenarios(c *gin.Context) {
	scenarios, err := h.service.ListScenarios(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to list scenarios", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scenarios"})
		return
	}

	// Если сценариев нет, возвращаем пустой массив, а не null
	if scenarios == nil {
		scenarios = []models.TestScenario{}
	}

	c.JSON(http.StatusOK, scenarios)
}

// DeleteScenario godoc
// @Summary      Удалить сценарий
// @Description  Удаляет сценарий по ID
// @Tags         scenarios
// @Param        id   path      string  true  "ID сценария"
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/scenarios/{id} [delete]
func (h *ScenarioHandler) DeleteScenario(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	err := h.service.DeleteScenario(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrScenarioNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scenario not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateScenario godoc
// @Summary      Обновить сценарий
// @Description  Обновляет существующий сценарий по ID
// @Tags         scenarios
// @Accept       json
// @Produce      json
// @Param        id     path      string               true  "ID сценария"
// @Param        input  body      models.TestScenario  true  "Обновленные данные"
// @Success      200    {object}  models.TestScenario
// @Failure      400    {object}  map[string]string
// @Failure      404    {object}  map[string]string
// @Failure      500    {object}  map[string]string
// @Router       /api/v1/scenarios/{id} [put]
func (h *ScenarioHandler) UpdateScenario(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	var scenario models.TestScenario
	// Парсим JSON.
	if err := c.ShouldBindJSON(&scenario); err != nil {
		h.logger.Error("Failed to bind JSON for update", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Принудительно ставим ID из URL, чтобы он совпадал с объектом.
	scenario.ID = id

	// Вызываем сервис обновления.
	err := h.service.UpdateScenario(c.Request.Context(), scenario)
	if err != nil {
		if errors.Is(err, service.ErrInvalidScenario) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if errors.Is(err, service.ErrScenarioNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scenario not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	c.JSON(http.StatusOK, scenario)
}

// ListContainers godoc
// @Summary      Список контейнеров
// @Description  Возвращает список запущенных Docker-контейнеров доступных для chaos-тестирования
// @Tags         chaos
// @Produce      json
// @Success      200  {array}   chaos.ContainerInfo
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/containers [get]
func (h *ScenarioHandler) ListContainers(c *gin.Context) {
	containers, err := h.service.ListContainers(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to list containers", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch containers"})
		return
	}

	if containers == nil {
		containers = []chaos.ContainerInfo{}
	}

	c.JSON(http.StatusOK, containers)
}
