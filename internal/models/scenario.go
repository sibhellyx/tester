package models

import "fmt"

// TestScenario - корневая структура, описывающая весь тест.
type TestScenario struct {
	ID             string          `json:"id"`                        // Уникальный ID запуска (генерируется сервисом).
	Name           string          `json:"name"`                      // Читаемое имя теста (напр. "Checkout Load Test").
	BaseURL        string          `json:"base_url"`                  // Базовый URL (напр. "https://api.myshop.com").
	TotalDuration  int             `json:"total_duration"`            // Общая длительность в секундах (защита от зависания).
	Stages         []Stage         `json:"stages"`                    // Список этапов выполнения.
	StopConditions *StopConditions `json:"stop_conditions,omitempty"` // Критерии остановки для теста.
}

// StageType - тип этапа нагрузки.
type StageType string

const (
	StageRampUp   StageType = "ramp_up"   // Плавный разгон (линейный рост пользователей).
	StageSteady   StageType = "steady"    // Постоянная нагрузка (фиксированное число пользователей).
	StageRampDown StageType = "ramp_down" // Плавное снижение (линейное уменьшение).
	StagePeak     StageType = "spike"     // Стрессовый пик: мгновенный скачок до TargetUsers, удержание, возврат к базовому уровню.
)

// Stage - один этап нагрузочного тестирования.
type Stage struct {
	ID          int           `json:"id"`           // Порядковый номер этапа (1, 2, 3...).
	Type        StageType     `json:"type"`         // Тип этапа (ramp_up, steady, ramp_down, spike).
	Duration    int           `json:"duration"`     // Длительность этапа в секундах.
	TargetUsers int           `json:"target_users"` // Целевое количество пользователей (VU) к концу этапа.
	Requests    []TestRequest `json:"requests"`     // Набор запросов для этого этапа (с весами).

	// Параметры стрессового пика (только для type=spike).
	// Количество пользователей, к которому нужно вернуться после пика.
	// Если 0 — возврат к числу пользователей, которое было в пуле до начала этапа.
	BaselineUsers int `json:"baseline_users,omitempty"`

	// Хаос-инжиниринг.
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

// StopConditions - параметры для остановки теста.
type StopConditions struct {
	TargetContainerID  string   `json:"target_container"`                // ID контейнера
	ErrorRatePercent   *float64 `json:"error_rate_percent,omitempty"`    // порог ошибок в процентах (1–100)
	MaxResponseTimeSec *float64 `json:"max_response_time_sec,omitempty"` // максимально допустимое время отклика в секундах
	// MaxCPUPercent и MaxRAMPercent требует TargetContainerID
	MaxCPUPercent *float64 `json:"max_cpu_percent,omitempty"` // максимально допустимая загрузка CPU контейнера (1–95)
	MaxRAMPercent *float64 `json:"max_ram_percent,omitempty"` // максимально допустимое использование RAM контейнера (1–95)
}

// Validate проверяет корректность сценария перед запуском.
func (s *TestScenario) Validate() error {
	if s.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if len(s.Stages) == 0 {
		return fmt.Errorf("scenario must have at least one stage")
	}
	// Валидация этапов сценария.
	totalDuration, err := validateStages(s.Stages)
	if err != nil {
		return err
	}
	// Валидация критериев остановки.
	err = validateStopConditions(s.StopConditions)
	if err != nil {
		return err
	}

	// Если пользователь забыл указать TotalDuration, считаем сами.
	if s.TotalDuration == 0 {
		s.TotalDuration = totalDuration
	} else if s.TotalDuration < totalDuration {
		return fmt.Errorf("total_duration (%d) is less than sum of stages (%d)", s.TotalDuration, totalDuration)
	}

	return nil
}

// validateStopConditions проверяте корректность критериев остановки.
func validateStopConditions(sc *StopConditions) error {
	if sc == nil {
		return nil // критерии не заданы — ок
	}

	if sc.ErrorRatePercent != nil {
		v := *sc.ErrorRatePercent
		if v < 1 || v > 100 {
			return fmt.Errorf("stop_conditions: error_rate_percent must be between 1 and 100, got %.1f", v)
		}
	}

	if sc.MaxResponseTimeSec != nil && *sc.MaxResponseTimeSec <= 0 {
		return fmt.Errorf("stop_conditions: max_response_time_sec must be positive")
	}

	if sc.MaxCPUPercent != nil {
		v := *sc.MaxCPUPercent
		if v < 1 || v > 95 {
			return fmt.Errorf("stop_conditions: max_cpu_percent must be between 1 and 95, got %.1f", v)
		}
	}

	if sc.MaxRAMPercent != nil {
		v := *sc.MaxRAMPercent
		if v < 1 || v > 95 {
			return fmt.Errorf("stop_conditions: max_ram_percent must be between 1 and 95, got %.1f", v)
		}
	}

	// CPU или RAM заданы — контейнер обязателен
	if (sc.MaxCPUPercent != nil || sc.MaxRAMPercent != nil) && sc.TargetContainerID == "" {
		return fmt.Errorf("stop_conditions: target_container_id is required when max_cpu_percent or max_ram_percent is set")
	}

	return nil
}

// validateStages проверят корректность этапов тестирования.
func validateStages(stages []Stage) (int, error) {
	totalDuration := 0
	for i, stage := range stages {
		if stage.Duration <= 0 {
			return -1, fmt.Errorf("stage #%d duration must be positive", i+1)
		}
		if stage.Type == StageRampDown {
			if stage.TargetUsers < 0 {
				return -1, fmt.Errorf("stage #%d (ramp_down) target_users cannot be negative", i+1)
			}
		} else {
			if stage.TargetUsers <= 0 {
				return -1, fmt.Errorf("stage #%d target_users must be positive", i+1)
			}
		}
		if len(stage.Requests) == 0 {
			return -1, fmt.Errorf("stage #%d must have at least one request", i+1)
		}

		// Проверка весов запросов.
		weightSum := 0
		for _, req := range stage.Requests {
			if req.Weight < 0 {
				return -1, fmt.Errorf("request '%s' in stage #%d has negative weight", req.Name, i+1)
			}
			weightSum += req.Weight
		}
		if weightSum == 0 {
			return -1, fmt.Errorf("stage #%d requests total weight must be > 0", i+1)
		}

		totalDuration += stage.Duration

		// Валидация параметров стрессового пика.
		if stage.Type == StagePeak {
			if stage.BaselineUsers < 0 {
				return -1, fmt.Errorf("stage #%d (spike): baseline_users cannot be negative", i+1)
			}
		}

		// валидация сбоев при наличии.
		for j, chaos := range stage.ChaosEvents {
			if chaos.TargetContainerID == "" {
				return -1, fmt.Errorf("stage #%d chaos #%d target_container is required", i+1, j+1)
			}
			if chaos.Duration <= 0 {
				return -1, fmt.Errorf("stage #%d chaos #%d duration must be positive", i+1, j+1)
			}
			if chaos.StartDelay < 0 {
				return -1, fmt.Errorf("stage #%d chaos #%d start_delay cannot be negative", i+1, j+1)
			}
			// Сбой не должен вылезать за пределы этапа.
			if chaos.StartDelay+chaos.Duration > stage.Duration {
				return -1, fmt.Errorf("stage #%d chaos #%d exceeds stage duration", i+1, j+1)
			}

			// Валидация типов (опционально).
			if chaos.Type == ChaosNetworkDelay && chaos.Delay == "" {
				return -1, fmt.Errorf("stage #%d chaos #%d network_delay requires 'delay' param", i+1, j+1)
			}
		}
	}
	return totalDuration, nil
}
