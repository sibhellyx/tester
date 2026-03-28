package checker

import (
	"testing"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

func TestThresholdChecker_NilConditions(t *testing.T) {
	tc := NewThresholdChecker(nil)
	tc.Record(models.CallResult{Error: "fail"})
	if violated, _ := tc.IsViolated(); violated {
		t.Error("expected no violation with nil conditions")
	}
}

func TestThresholdChecker_NoViolation_BelowAllThresholds(t *testing.T) {
	errRate := 50.0
	maxLatency := 1.0
	tc := NewThresholdChecker(&models.StopConditions{
		ErrorRatePercent:   &errRate,
		MaxResponseTimeSec: &maxLatency,
	})
	tc.Record(models.CallResult{Status: 200, Duration: 500 * time.Millisecond})
	if violated, _ := tc.IsViolated(); violated {
		t.Error("expected no violation when below all thresholds")
	}
}

func TestThresholdChecker_Latency_BelowThreshold_NoViolation(t *testing.T) {
	maxLatency := 1.0
	tc := NewThresholdChecker(&models.StopConditions{
		MaxResponseTimeSec: &maxLatency,
	})
	// Exactly at threshold — not violated because check is strictly >
	tc.Record(models.CallResult{Status: 200, Duration: 1 * time.Second})
	if violated, _ := tc.IsViolated(); violated {
		t.Error("expected no violation when duration equals threshold (not >)")
	}
}

func TestThresholdChecker_Latency_ExceedsThreshold_Violation(t *testing.T) {
	maxLatency := 1.0
	tc := NewThresholdChecker(&models.StopConditions{
		MaxResponseTimeSec: &maxLatency,
	})
	tc.Record(models.CallResult{Status: 200, Duration: 1001 * time.Millisecond})

	violated, reason := tc.IsViolated()
	if !violated {
		t.Fatal("expected latency violation")
	}
	if reason == "" {
		t.Error("expected non-empty violation reason")
	}
}

func TestThresholdChecker_Latency_PersistsAfterFirstBreach(t *testing.T) {
	maxLatency := 0.5
	tc := NewThresholdChecker(&models.StopConditions{
		MaxResponseTimeSec: &maxLatency,
	})
	tc.Record(models.CallResult{Status: 200, Duration: 1 * time.Second})   // breach
	tc.Record(models.CallResult{Status: 200, Duration: 10 * time.Millisecond}) // well below

	if violated, _ := tc.IsViolated(); !violated {
		t.Error("expected violation to persist after first breach")
	}
}

func TestThresholdChecker_ErrorRate_BelowMinCalls_NoViolation(t *testing.T) {
	errRate := 5.0
	tc := NewThresholdChecker(&models.StopConditions{
		ErrorRatePercent: &errRate,
	})
	// All calls fail but fewer than minCallsForErrorRate (100)
	for i := 0; i < 99; i++ {
		tc.Record(models.CallResult{Error: "fail"})
	}
	if violated, _ := tc.IsViolated(); violated {
		t.Error("expected no violation below minCallsForErrorRate threshold")
	}
}

func TestThresholdChecker_ErrorRate_Violation(t *testing.T) {
	errRate := 10.0
	tc := NewThresholdChecker(&models.StopConditions{
		ErrorRatePercent: &errRate,
	})
	// 80 successes + 20 failures = 20% error rate > 10%
	for i := 0; i < 80; i++ {
		tc.Record(models.CallResult{Status: 200})
	}
	for i := 0; i < 20; i++ {
		tc.Record(models.CallResult{Error: "network error"})
	}

	violated, reason := tc.IsViolated()
	if !violated {
		t.Fatal("expected error rate violation")
	}
	if reason == "" {
		t.Error("expected non-empty violation reason")
	}
}

func TestThresholdChecker_ErrorRate_NoViolation(t *testing.T) {
	errRate := 30.0
	tc := NewThresholdChecker(&models.StopConditions{
		ErrorRatePercent: &errRate,
	})
	// 90 successes + 10 failures = 10% error rate < 30%
	for i := 0; i < 90; i++ {
		tc.Record(models.CallResult{Status: 200})
	}
	for i := 0; i < 10; i++ {
		tc.Record(models.CallResult{Error: "fail"})
	}
	if violated, _ := tc.IsViolated(); violated {
		t.Error("expected no violation when error rate is below threshold")
	}
}

func TestThresholdChecker_ErrorRate_ExactThreshold_Violation(t *testing.T) {
	errRate := 10.0
	tc := NewThresholdChecker(&models.StopConditions{
		ErrorRatePercent: &errRate,
	})
	// 90 successes + 10 failures = exactly 10% — check is >=, so violated
	for i := 0; i < 90; i++ {
		tc.Record(models.CallResult{Status: 200})
	}
	for i := 0; i < 10; i++ {
		tc.Record(models.CallResult{Error: "fail"})
	}
	if violated, _ := tc.IsViolated(); !violated {
		t.Error("expected violation when error rate equals threshold (>=)")
	}
}

func TestThresholdChecker_Status0_CountsAsError(t *testing.T) {
	errRate := 5.0
	tc := NewThresholdChecker(&models.StopConditions{
		ErrorRatePercent: &errRate,
	})
	// 90 successes + 10 status=0 = 10% > 5%
	for i := 0; i < 90; i++ {
		tc.Record(models.CallResult{Status: 200})
	}
	for i := 0; i < 10; i++ {
		tc.Record(models.CallResult{Status: 0})
	}
	if violated, _ := tc.IsViolated(); !violated {
		t.Error("expected violation: status=0 must count as error")
	}
}
