package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/sibhellyx/tester/internal/core/chaos"
	"github.com/sibhellyx/tester/internal/models"
)

// Ошибки, которые могут возникнуть в бизнес-логике.
var (
	ErrScenarioNotFound = errors.New("scenario not found")
	ErrInvalidScenario  = errors.New("scenario validation failed")
	ErrRepoError        = errors.New("repository error")
)

type DockerClientInterface interface {
	ListContainers(ctx context.Context) ([]chaos.ContainerInfo, error)
}

// ScenarioRepository определяет методы работы с БД.
type ScenarioRepositoryInterface interface {
	Create(ctx context.Context, s models.TestScenario) error
	Get(ctx context.Context, id string) (*models.TestScenario, error)
	List(ctx context.Context) ([]models.TestScenario, error)
	Update(ctx context.Context, s models.TestScenario) error
	Delete(ctx context.Context, id string) error
}

// TestManagementService реализует логику управления сценариями.
type TestManagementService struct {
	logger *slog.Logger
	repo   ScenarioRepositoryInterface
	client DockerClientInterface
}

// NewTestManagementService - конструктор.
func NewTestManagementService(logger *slog.Logger, repo ScenarioRepositoryInterface, client DockerClientInterface) *TestManagementService {
	return &TestManagementService{
		logger: logger,
		repo:   repo,
		client: client,
	}
}

// CreateScenario создает новый сценарий.
func (s *TestManagementService) CreateScenario(ctx context.Context, scenario models.TestScenario) (string, error) {
	// Валидация (Бизнес-правила).
	if err := scenario.Validate(); err != nil {
		s.logger.Warn("Scenario validation failed", slog.String("error", err.Error()))
		// Оборачиваем ошибку, чтобы хендлер понял, что это 400 Bad Request
		return "", fmt.Errorf("%w: %s", ErrInvalidScenario, err.Error())
	}

	// Генерация ID, если не задан.
	if scenario.ID == "" {
		scenario.ID = uuid.New().String()
	}

	// Сохранение в БД.
	if err := s.repo.Create(ctx, scenario); err != nil {
		s.logger.Error("Failed to save scenario to repo", slog.String("error", err.Error()))
		return "", ErrRepoError
	}

	s.logger.Info("Scenario created", slog.String("id", scenario.ID))
	return scenario.ID, nil
}

// GetScenario получает сценарий по ID.
func (s *TestManagementService) GetScenario(ctx context.Context, id string) (*models.TestScenario, error) {
	scenario, err := s.repo.Get(ctx, id)
	if err != nil {
		// Репозиторий должен возвращать специфичную ошибку, если не нашел,
		// но для надежности проверим nil.
		s.logger.Error("Failed to get scenario", slog.String("id", id), slog.String("error", err.Error()))
		return nil, ErrRepoError
	}
	if scenario == nil {
		return nil, ErrScenarioNotFound
	}

	return scenario, nil
}

// ListScenarios возвращает список всех сценариев.
func (s *TestManagementService) ListScenarios(ctx context.Context) ([]models.TestScenario, error) {
	scenarios, err := s.repo.List(ctx)
	if err != nil {
		s.logger.Error("Failed to list scenarios", slog.String("error", err.Error()))
		return nil, ErrRepoError
	}
	return scenarios, nil
}

// UpdateScenario обновляет сценарий.
func (s *TestManagementService) UpdateScenario(ctx context.Context, scenario models.TestScenario) error {
	// Валидация.
	if err := scenario.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidScenario, err.Error())
	}

	// Проверка существования (опционально, зависит от логики репозитория).
	existing, err := s.repo.Get(ctx, scenario.ID)
	if err != nil {
		return ErrRepoError
	}
	if existing == nil {
		return ErrScenarioNotFound
	}

	// Обновление.
	if err := s.repo.Update(ctx, scenario); err != nil {
		s.logger.Error("Failed to update scenario", slog.String("id", scenario.ID), slog.String("error", err.Error()))
		return ErrRepoError
	}

	s.logger.Info("Scenario updated", slog.String("id", scenario.ID))
	return nil
}

// DeleteScenario удаляет сценарий.
func (s *TestManagementService) DeleteScenario(ctx context.Context, id string) error {
	// Проверка существования перед удалением (чтобы вернуть 404 если нет).
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return ErrRepoError
	}
	if existing == nil {
		return ErrScenarioNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete scenario", slog.String("id", id), slog.String("error", err.Error()))
		return ErrRepoError
	}

	s.logger.Info("Scenario deleted", slog.String("id", id))
	return nil
}

// ListContainers возвращает список запущенных в системе контейнеров.
func (s *TestManagementService) ListContainers(ctx context.Context) ([]chaos.ContainerInfo, error) {
	return s.client.ListContainers(ctx)
}
