package checker

import (
	"fmt"
	"sync"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

const minCallsForErrorRate = 100 // не останавливаем тест по первым N запросам

type ThresholdChecker struct {
	conditions *models.StopConditions

	mu              sync.Mutex
	totalCalls      int64
	failedCalls     int64
	latencyBreached bool
}

func NewThresholdChecker(sc *models.StopConditions) *ThresholdChecker {
	return &ThresholdChecker{conditions: sc}
}

// Record вызывается для каждого результата в цикле executeRun.
func (t *ThresholdChecker) Record(result models.CallResult) {
	if t.conditions == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalCalls++

	// Ошибкой считаем: сетевую ошибку, статус 0, или статус >= 500.
	// Статусы из ExpectedStatusCodes уже обработаны в attacker.go —
	// там result.Error будет заполнен если код не ожидался.
	if result.Error != "" || result.Status == 0 {
		t.failedCalls++
	}

	if t.conditions.MaxResponseTimeSec != nil && !t.latencyBreached {
		limit := time.Duration(*t.conditions.MaxResponseTimeSec * float64(time.Second))
		if result.Duration > limit {
			t.latencyBreached = true
		}
	}
}

// IsViolated возвращает (true, причина) если критерий нарушен.
func (t *ThresholdChecker) IsViolated() (bool, string) {
	if t.conditions == nil {
		return false, ""
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.latencyBreached {
		return true, fmt.Sprintf(
			"max response time %.2fs exceeded",
			*t.conditions.MaxResponseTimeSec,
		)
	}

	if t.conditions.ErrorRatePercent != nil && t.totalCalls >= minCallsForErrorRate {
		rate := float64(t.failedCalls) / float64(t.totalCalls) * 100
		if rate >= *t.conditions.ErrorRatePercent {
			return true, fmt.Sprintf(
				"error rate %.1f%% exceeded threshold %.1f%%",
				rate, *t.conditions.ErrorRatePercent,
			)
		}
	}

	return false, ""
}
