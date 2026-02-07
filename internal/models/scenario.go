package models

import "fmt"

// TestScenario - корневая структура, описывающая весь тест.
type TestScenario struct {
	ID            string  `json:"id"`             // Уникальный ID запуска (генерируется сервисом).
	Name          string  `json:"name"`           // Читаемое имя теста (напр. "Checkout Load Test").
	BaseURL       string  `json:"base_url"`       // Базовый URL (напр. "https://api.myshop.com").
	TotalDuration int     `json:"total_duration"` // Общая длительность в секундах (защита от зависания).
	Stages        []Stage `json:"stages"`         // Список этапов выполнения.
}

// StageType - тип этапа нагрузки.
type StageType string

const (
	StageRampUp   StageType = "ramp_up"   // Плавный разгон (линейный рост пользователей).
	StageSteady   StageType = "steady"    // Постоянная нагрузка (фиксированное число пользователей).
	StageRampDown StageType = "ramp_down" // Плавное снижение (линейное уменьшение).
)

// Stage - один этап нагрузочного тестирования.
type Stage struct {
	ID          int           `json:"id"`           // Порядковый номер этапа (1, 2, 3...).
	Type        StageType     `json:"type"`         // Тип этапа (ramp_up, steady, ramp_down).
	Duration    int           `json:"duration"`     // Длительность этапа в секундах.
	TargetUsers int           `json:"target_users"` // Целевое количество пользователей (VU) к концу этапа.
	Requests    []TestRequest `json:"requests"`     // Набор запросов для этого этапа (с весами).

	// Хаос-инжиниринг (будет добавлено позже).
}

// Validate проверяет корректность сценария перед запуском.
func (s *TestScenario) Validate() error {
	if s.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if len(s.Stages) == 0 {
		return fmt.Errorf("scenario must have at least one stage")
	}

	totalDuration := 0
	for i, stage := range s.Stages {
		if stage.Duration <= 0 {
			return fmt.Errorf("stage #%d duration must be positive", i+1)
		}
		if stage.TargetUsers <= 0 {
			return fmt.Errorf("stage #%d target_users must be positive", i+1)
		}
		if len(stage.Requests) == 0 {
			return fmt.Errorf("stage #%d must have at least one request", i+1)
		}

		// Проверка весов запросов.
		weightSum := 0
		for _, req := range stage.Requests {
			if req.Weight < 0 {
				return fmt.Errorf("request '%s' in stage #%d has negative weight", req.Name, i+1)
			}
			weightSum += req.Weight
		}
		if weightSum == 0 {
			return fmt.Errorf("stage #%d requests total weight must be > 0", i+1)
		}

		totalDuration += stage.Duration
	}

	// Если пользователь забыл указать TotalDuration, считаем сами.
	if s.TotalDuration == 0 {
		s.TotalDuration = totalDuration
	} else if s.TotalDuration < totalDuration {
		return fmt.Errorf("total_duration (%d) is less than sum of stages (%d)", s.TotalDuration, totalDuration)
	}

	return nil
}
