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
	ChaosEvents []ChaosParams `json:"chaos_events,omitempty"`
}

// ChaosType - тип сбоя.
type ChaosType string

const (
	ChaosShutdown     ChaosType = "component_shutdown"
	ChaosNetworkDelay ChaosType = "network_delay"
	ChaosPacketLoss   ChaosType = "packet_loss"
	ChaosResource     ChaosType = "resource_limit"
)

// ChaosParams - параметры одного сбоя.
type ChaosParams struct {
	Type              ChaosType `json:"type"`             // Тип сбоя
	TargetContainerID string    `json:"target_container"` // ID контейнера
	StartDelay        int       `json:"start_delay"`      // Задержка от начала этапа (сек)
	Duration          int       `json:"duration"`         // Длительность сбоя (сек)

	// Специфичные параметры (зависят от типа)
	Delay       string `json:"delay,omitempty"`        // "100ms" (для network_delay)
	Jitter      string `json:"jitter,omitempty"`       // "10ms" (для network_delay)
	PacketLoss  int    `json:"packet_loss,omitempty"`  // 0-100 (для network_loss)
	CPUQuota    int64  `json:"cpu_quota,omitempty"`    // -1..100000 (для resource_limit)
	MemoryBytes int64  `json:"memory_bytes,omitempty"` // bytes (для resource_limit)

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

		// валидация сбоев при наличии.
		for j, chaos := range stage.ChaosEvents {
			if chaos.TargetContainerID == "" {
				return fmt.Errorf("stage #%d chaos #%d target_container is required", i+1, j+1)
			}
			if chaos.Duration <= 0 {
				return fmt.Errorf("stage #%d chaos #%d duration must be positive", i+1, j+1)
			}
			if chaos.StartDelay < 0 {
				return fmt.Errorf("stage #%d chaos #%d start_delay cannot be negative", i+1, j+1)
			}
			// Сбой не должен вылезать за пределы этапа.
			if chaos.StartDelay+chaos.Duration > stage.Duration {
				return fmt.Errorf("stage #%d chaos #%d exceeds stage duration", i+1, j+1)
			}

			// Валидация типов (опционально).
			if chaos.Type == ChaosNetworkDelay && chaos.Delay == "" {
				return fmt.Errorf("stage #%d chaos #%d network_delay requires 'delay' param", i+1, j+1)
			}
		}
	}

	// Если пользователь забыл указать TotalDuration, считаем сами.
	if s.TotalDuration == 0 {
		s.TotalDuration = totalDuration
	} else if s.TotalDuration < totalDuration {
		return fmt.Errorf("total_duration (%d) is less than sum of stages (%d)", s.TotalDuration, totalDuration)
	}

	return nil
}
