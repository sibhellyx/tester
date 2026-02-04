package load

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
)

// TestNewWeightedGenerator_Success проверяет успешное создание генератора.
func TestNewWeightedGenerator_Success(t *testing.T) {
	requests := []TestRequest{
		{Name: "Login", Weight: 50},
		{Name: "GetUsers", Weight: 30},
		{Name: "CreateOrder", Weight: 20},
	}

	gen, err := NewWeightedGenerator(requests)

	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if gen == nil {
		t.Fatal("Generator should not be nil")
	}

	if gen.totalWeight != 100 {
		t.Errorf("Expected totalWeight 100, got %d", gen.totalWeight)
	}

	expectedCumulative := []int{50, 80, 100}
	for i, expected := range expectedCumulative {
		if gen.cumulativeWeights[i] != expected {
			t.Errorf("cumulativeWeights[%d]: expected %d, got %d", i, expected, gen.cumulativeWeights[i])
		}
	}
}

// TestNewWeightedGenerator_EmptyRequests проверяет ошибку при пустом списке запросов.
func TestNewWeightedGenerator_EmptyRequests(t *testing.T) {
	requests := []TestRequest{}

	gen, err := NewWeightedGenerator(requests)

	if err == nil {
		t.Fatal("Expected error for empty requests list")
	}

	if gen != nil {
		t.Error("Generator should be nil on error")
	}

	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("Expected 'cannot be empty' error, got: %s", err)
	}
}

// TestNewWeightedGenerator_NegativeWeight проверяет ошибку при отрицательном весе.
func TestNewWeightedGenerator_NegativeWeight(t *testing.T) {
	requests := []TestRequest{
		{Name: "Request1", Weight: 50},
		{Name: "BadRequest", Weight: -10},
		{Name: "Request2", Weight: 60},
	}

	gen, err := NewWeightedGenerator(requests)

	if err == nil {
		t.Fatal("Expected error for negative weight")
	}

	if gen != nil {
		t.Error("Generator should be nil on error")
	}

	if !strings.Contains(err.Error(), "negative weight") {
		t.Errorf("Expected 'negative weight' error, got: %s", err)
	}

	if !strings.Contains(err.Error(), "BadRequest") {
		t.Errorf("Error should mention request name, got: %s", err)
	}
}

// TestNewWeightedGenerator_ZeroTotalWeight проверяет ошибку, когда сумма весов = 0.
func TestNewWeightedGenerator_ZeroTotalWeight(t *testing.T) {
	requests := []TestRequest{
		{Name: "Request1", Weight: 0},
		{Name: "Request2", Weight: 0},
		{Name: "Request3", Weight: 0},
	}

	gen, err := NewWeightedGenerator(requests)

	if err == nil {
		t.Fatal("Expected error for zero total weight")
	}

	if gen != nil {
		t.Error("Generator should be nil on error")
	}

	if !strings.Contains(err.Error(), "total weight is 0") {
		t.Errorf("Expected 'total weight is 0' error, got: %s", err)
	}
}

// TestNewWeightedGenerator_PartialZeroWeights проверяет работу с частичными нулевыми весами.
// Запросы с весом 0 не должны выбираться.
func TestNewWeightedGenerator_PartialZeroWeights(t *testing.T) {
	requests := []TestRequest{
		{Name: "Active1", Weight: 50},
		{Name: "Disabled", Weight: 0}, // Отключенный запрос.
		{Name: "Active2", Weight: 50},
	}

	gen, err := NewWeightedGenerator(requests)

	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	// Генерируем 1000 запросов и проверяем, что "Disabled" никогда не выбирается.
	counts := make(map[string]int)
	for i := 0; i < 1000; i++ {
		req := gen.Next()
		counts[req.Name]++
	}

	if counts["Disabled"] > 0 {
		t.Errorf("Request with weight 0 should never be selected, got %d times", counts["Disabled"])
	}

	if counts["Active1"] == 0 || counts["Active2"] == 0 {
		t.Error("Active requests should be selected")
	}
}

// TestWeightedGenerator_Next_BasicDistribution проверяет базовое распределение вероятностей.
func TestWeightedGenerator_Next_BasicDistribution(t *testing.T) {
	requests := []TestRequest{
		{Name: "Request1", Weight: 50},
		{Name: "Request2", Weight: 30},
		{Name: "Request3", Weight: 20},
	}

	gen, err := NewWeightedGenerator(requests)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	samples := 10000
	counts := make(map[string]int)

	for i := 0; i < samples; i++ {
		req := gen.Next()
		if req == nil {
			t.Fatal("Next() returned nil")
		}
		counts[req.Name]++
	}

	// Проверяем распределение с допуском ±2%.
	expectedDistribution := map[string]float64{
		"Request1": 50.0,
		"Request2": 30.0,
		"Request3": 20.0,
	}

	for name, expectedPercent := range expectedDistribution {
		actualCount := counts[name]
		actualPercent := float64(actualCount) / float64(samples) * 100
		deviation := math.Abs(actualPercent - expectedPercent)

		if deviation > 2.0 {
			t.Errorf("%s: expected ~%.1f%%, got %.1f%% (deviation %.2f%%)",
				name, expectedPercent, actualPercent, deviation)
		}
	}
}

// TestWeightedGenerator_Next_HighPercentage проверяет работу с сильным перекосом (95%/3%/2%).
func TestWeightedGenerator_Next_HighPercentage(t *testing.T) {
	requests := []TestRequest{
		{Name: "DominantRequest", Weight: 95},
		{Name: "RareRequest1", Weight: 3},
		{Name: "RareRequest2", Weight: 2},
	}

	gen, err := NewWeightedGenerator(requests)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	samples := 100000 // Большая выборка для точности.
	counts := make(map[string]int)

	for i := 0; i < samples; i++ {
		req := gen.Next()
		counts[req.Name]++
	}

	// Проверяем распределение с допуском ±0.5% для большой выборки.
	expectedDistribution := map[string]float64{
		"DominantRequest": 95.0,
		"RareRequest1":    3.0,
		"RareRequest2":    2.0,
	}

	for name, expectedPercent := range expectedDistribution {
		actualCount := counts[name]
		actualPercent := float64(actualCount) / float64(samples) * 100
		deviation := math.Abs(actualPercent - expectedPercent)

		if deviation > 0.5 {
			t.Errorf("%s: expected ~%.1f%%, got %.1f%% (deviation %.2f%%)",
				name, expectedPercent, actualPercent, deviation)
		}

		t.Logf("%s: expected %.1f%%, got %.1f%% (count: %d, deviation: %.3f%%)",
			name, expectedPercent, actualPercent, actualCount, deviation)
	}
}

// TestWeightedGenerator_Next_ExtremePercentage проверяет экстремальный перекос (99%/0.5%/0.5%).
func TestWeightedGenerator_Next_ExtremePercentage(t *testing.T) {
	requests := []TestRequest{
		{Name: "AlmostAlways", Weight: 990},
		{Name: "VeryRare1", Weight: 5},
		{Name: "VeryRare2", Weight: 5},
	}

	gen, err := NewWeightedGenerator(requests)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	samples := 100000
	counts := make(map[string]int)

	for i := 0; i < samples; i++ {
		req := gen.Next()
		counts[req.Name]++
	}

	expectedDistribution := map[string]float64{
		"AlmostAlways": 99.0,
		"VeryRare1":    0.5,
		"VeryRare2":    0.5,
	}

	for name, expectedPercent := range expectedDistribution {
		actualCount := counts[name]
		actualPercent := float64(actualCount) / float64(samples) * 100
		deviation := math.Abs(actualPercent - expectedPercent)

		// Для редких событий (0.5%) допускаем чуть больше отклонение.
		maxDeviation := 0.3
		if expectedPercent < 1.0 {
			maxDeviation = 0.5
		}

		if deviation > maxDeviation {
			t.Errorf("%s: expected ~%.1f%%, got %.1f%% (deviation %.2f%%)",
				name, expectedPercent, actualPercent, deviation)
		}

		t.Logf("%s: expected %.1f%%, got %.1f%% (count: %d)",
			name, expectedPercent, actualPercent, actualCount)
	}

	// Проверяем, что редкие запросы все же выполнились.
	if counts["VeryRare1"] == 0 {
		t.Error("VeryRare1 should have been selected at least once in 100k samples")
	}
	if counts["VeryRare2"] == 0 {
		t.Error("VeryRare2 should have been selected at least once in 100k samples")
	}
}

// TestWeightedGenerator_Next_SingleRequest проверяет работу с одним запросом.
func TestWeightedGenerator_Next_SingleRequest(t *testing.T) {
	requests := []TestRequest{
		{Name: "OnlyOne", Weight: 100},
	}

	gen, err := NewWeightedGenerator(requests)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	for i := 0; i < 100; i++ {
		req := gen.Next()
		if req.Name != "OnlyOne" {
			t.Errorf("Expected 'OnlyOne', got '%s'", req.Name)
		}
	}
}

// TestWeightedGenerator_Next_TwoRequests проверяет работу с двумя запросами (70%/30%).
func TestWeightedGenerator_Next_TwoRequests(t *testing.T) {
	requests := []TestRequest{
		{Name: "Primary", Weight: 70},
		{Name: "Secondary", Weight: 30},
	}

	gen, err := NewWeightedGenerator(requests)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	samples := 10000
	counts := make(map[string]int)

	for i := 0; i < samples; i++ {
		req := gen.Next()
		counts[req.Name]++
	}

	primaryPercent := float64(counts["Primary"]) / float64(samples) * 100
	secondaryPercent := float64(counts["Secondary"]) / float64(samples) * 100

	if math.Abs(primaryPercent-70.0) > 2.0 {
		t.Errorf("Primary: expected ~70%%, got %.1f%%", primaryPercent)
	}

	if math.Abs(secondaryPercent-30.0) > 2.0 {
		t.Errorf("Secondary: expected ~30%%, got %.1f%%", secondaryPercent)
	}
}

// TestWeightedGenerator_Next_NonStandardWeights проверяет работу с произвольными весами (не %).
func TestWeightedGenerator_Next_NonStandardWeights(t *testing.T) {
	requests := []TestRequest{
		{Name: "Request1", Weight: 1},
		{Name: "Request2", Weight: 2},
		{Name: "Request3", Weight: 3},
	}

	gen, err := NewWeightedGenerator(requests)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Общий вес: 1 + 2 + 3 = 6.
	// Ожидаемое распределение: 16.67%, 33.33%, 50%.
	samples := 60000 // Кратно 6 для удобства.
	counts := make(map[string]int)

	for i := 0; i < samples; i++ {
		req := gen.Next()
		counts[req.Name]++
	}

	expectedDistribution := map[string]float64{
		"Request1": 16.67,
		"Request2": 33.33,
		"Request3": 50.0,
	}

	for name, expectedPercent := range expectedDistribution {
		actualCount := counts[name]
		actualPercent := float64(actualCount) / float64(samples) * 100
		deviation := math.Abs(actualPercent - expectedPercent)

		if deviation > 1.0 {
			t.Errorf("%s: expected ~%.2f%%, got %.2f%% (deviation %.2f%%)",
				name, expectedPercent, actualPercent, deviation)
		}
	}
}

// TestWeightedGenerator_Concurrency проверяет потокобезопасность генератора.
func TestWeightedGenerator_Concurrency(t *testing.T) {
	requests := []TestRequest{
		{Name: "Request1", Weight: 50},
		{Name: "Request2", Weight: 50},
	}

	gen, err := NewWeightedGenerator(requests)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	var wg sync.WaitGroup
	goroutines := 10
	requestsPerGoroutine := 1000
	results := make(chan string, goroutines*requestsPerGoroutine)

	// Запускаем несколько горутин, которые параллельно вызывают Next().
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				req := gen.Next()
				results <- req.Name
			}
		}()
	}

	wg.Wait()
	close(results)

	// Подсчитываем результаты.
	counts := make(map[string]int)
	totalSamples := 0
	for name := range results {
		counts[name]++
		totalSamples++
	}

	if totalSamples != goroutines*requestsPerGoroutine {
		t.Errorf("Expected %d total samples, got %d", goroutines*requestsPerGoroutine, totalSamples)
	}

	// Проверяем распределение ~50/50 с допуском ±5% (из-за параллелизма).
	for name, count := range counts {
		actualPercent := float64(count) / float64(totalSamples) * 100
		deviation := math.Abs(actualPercent - 50.0)

		if deviation > 5.0 {
			t.Errorf("%s: expected ~50%%, got %.1f%% (deviation %.2f%%)",
				name, actualPercent, deviation)
		}
	}

	t.Logf("Concurrency test passed with %d goroutines, %d total requests",
		goroutines, totalSamples)
}

// TestWeightedGenerator_ManyRequests проверяет работу с большим количеством разных запросов.
func TestWeightedGenerator_ManyRequests(t *testing.T) {
	// Создаем 20 запросов с равными весами.
	requests := make([]TestRequest, 20)
	for i := 0; i < 20; i++ {
		requests[i] = TestRequest{
			Name:   fmt.Sprintf("Request%d", i+1),
			Weight: 5, // Каждый по 5% (всего 100%).
		}
	}

	gen, err := NewWeightedGenerator(requests)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	samples := 100000
	counts := make(map[string]int)

	for i := 0; i < samples; i++ {
		req := gen.Next()
		counts[req.Name]++
	}

	// Каждый запрос должен быть выбран примерно 5000 раз (5% от 100000).
	expectedCount := samples / 20
	tolerance := float64(expectedCount) * 0.15 // ±15% допуск.

	for i := 1; i <= 20; i++ {
		name := fmt.Sprintf("Request%d", i)
		actualCount := counts[name]

		if actualCount == 0 {
			t.Errorf("%s was never selected", name)
			continue
		}

		deviation := math.Abs(float64(actualCount - expectedCount))
		if deviation > tolerance {
			t.Errorf("%s: expected ~%d, got %d (deviation %.0f, tolerance %.0f)",
				name, expectedCount, actualCount, deviation, tolerance)
		}
	}

	t.Logf("Many requests test: 20 requests, each ~5%%, passed with %d samples", samples)
}

// TestWeightedGenerator_CumulativeWeightsCalculation проверяет правильность расчета накопительных весов.
func TestWeightedGenerator_CumulativeWeightsCalculation(t *testing.T) {
	tests := []struct {
		name               string
		weights            []int
		expectedCumulative []int
		expectedTotal      int
	}{
		{
			name:               "Simple (10/20/30)",
			weights:            []int{10, 20, 30},
			expectedCumulative: []int{10, 30, 60},
			expectedTotal:      60,
		},
		{
			name:               "With zeros (50/0/50)",
			weights:            []int{50, 0, 50},
			expectedCumulative: []int{50, 50, 100},
			expectedTotal:      100,
		},
		{
			name:               "Large numbers (1000/2000/3000)",
			weights:            []int{1000, 2000, 3000},
			expectedCumulative: []int{1000, 3000, 6000},
			expectedTotal:      6000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := make([]TestRequest, len(tt.weights))
			for i, w := range tt.weights {
				requests[i] = TestRequest{
					Name:   fmt.Sprintf("Req%d", i+1),
					Weight: w,
				}
			}

			gen, err := NewWeightedGenerator(requests)
			if err != nil {
				t.Fatalf("Failed to create generator: %v", err)
			}

			if gen.totalWeight != tt.expectedTotal {
				t.Errorf("totalWeight: expected %d, got %d", tt.expectedTotal, gen.totalWeight)
			}

			for i, expected := range tt.expectedCumulative {
				if gen.cumulativeWeights[i] != expected {
					t.Errorf("cumulativeWeights[%d]: expected %d, got %d",
						i, expected, gen.cumulativeWeights[i])
				}
			}
		})
	}
}
