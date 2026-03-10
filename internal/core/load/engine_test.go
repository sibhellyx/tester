package load

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sibhellyx/tester/internal/models"
)

type MockAttacker struct {
	duration time.Duration
	mu       sync.Mutex
	calls    int
}

func (m *MockAttacker) Shoot(r models.TestRequest) models.CallResult {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if m.duration > 0 {
		time.Sleep(m.duration)
	}
	return models.CallResult{RequestName: r.Name, Status: 200, Timestamp: time.Now(), Duration: m.duration}
}

func (m *MockAttacker) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestEngine() (*Engine, *MockAttacker) {
	attacker := &MockAttacker{}
	return NewEngine(newTestLogger(), attacker), attacker
}

// makeResults создаёт буферизованный канал для результатов.
// Правило закрытия: всегда вызывайте pool.KillAll() перед close(results),
// иначе живые горутины пользователей запаникуют на записи в закрытый канал.
func makeResults() chan models.CallResult {
	return make(chan models.CallResult, 500)
}

func drainResults(ch chan models.CallResult) int {
	var n int
	for range ch {
		n++
	}
	return n
}

func makeRequests() []models.TestRequest {
	return []models.TestRequest{
		{Name: "req", Method: "GET", Path: "http://example.com/test", Weight: 100},
	}
}


func TestNewEngine(t *testing.T) {
	engine, attacker := newTestEngine()
	if engine == nil {
		t.Fatal("engine is nil")
	}
	if engine.attacker != attacker {
		t.Error("attacker not set")
	}
}

// ── StageSteady ───────────────────────────────────────────────────────────────

// Пользователи запускаются и производят результаты в течение длительности этапа.
func TestExecuteStage_Steady_ProducesResults(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	stage := models.Stage{
		ID: 1, Type: models.StageSteady, Duration: 1, TargetUsers: 3,
		Requests: makeRequests(),
	}

	engine.ExecuteStage(context.Background(), stage, pool, results)

	// Сначала останавливаем всех пользователей — они прекращают писать в results.
	// Только после этого безопасно закрывать канал.
	pool.KillAll()
	close(results)

	if n := drainResults(results); n == 0 {
		t.Error("expected results, got 0")
	}
}

// Пул должен содержать ровно TargetUsers после завершения этапа.
func TestExecuteStage_Steady_PoolSize(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	stage := models.Stage{
		ID: 1, Type: models.StageSteady, Duration: 1, TargetUsers: 5,
		Requests: makeRequests(),
	}

	engine.ExecuteStage(context.Background(), stage, pool, results)

	if got := pool.Len(); got != 5 {
		t.Errorf("expected pool size 5, got %d", got)
	}

	pool.KillAll()
}

// Если пользователей уже больше TargetUsers — лишние должны быть убиты.
func TestExecuteStage_Steady_KillsExcessUsers(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	// Сначала запускаем 5 пользователей.
	stage1 := models.Stage{
		ID: 1, Type: models.StageSteady, Duration: 1, TargetUsers: 5,
		Requests: makeRequests(),
	}
	engine.ExecuteStage(context.Background(), stage1, pool, results)
	if got := pool.Len(); got != 5 {
		t.Fatalf("setup: expected 5 users, got %d", got)
	}

	// Потом снижаем до 2.
	stage2 := models.Stage{
		ID: 2, Type: models.StageSteady, Duration: 1, TargetUsers: 2,
		Requests: makeRequests(),
	}
	engine.ExecuteStage(context.Background(), stage2, pool, results)

	if got := pool.Len(); got != 2 {
		t.Errorf("expected pool size 2 after kill, got %d", got)
	}

	pool.KillAll()
}

// Если количество пользователей уже равно TargetUsers — ничего не меняется.
func TestExecuteStage_Steady_NoChangeWhenAtTarget(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	stage := models.Stage{
		ID: 1, Type: models.StageSteady, Duration: 1, TargetUsers: 3,
		Requests: makeRequests(),
	}
	engine.ExecuteStage(context.Background(), stage, pool, results)
	before := pool.Len()

	engine.ExecuteStage(context.Background(), stage, pool, results)
	after := pool.Len()

	if before != after {
		t.Errorf("pool size changed from %d to %d, expected no change", before, after)
	}

	pool.KillAll()
}

// ── StageRampUp ───────────────────────────────────────────────────────────────

// После RampUp пул должен содержать TargetUsers пользователей.
func TestExecuteStage_RampUp_ReachesTarget(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	stage := models.Stage{
		ID: 1, Type: models.StageRampUp, Duration: 1, TargetUsers: 4,
		Requests: makeRequests(),
	}

	engine.ExecuteStage(context.Background(), stage, pool, results)

	if got := pool.Len(); got != 4 {
		t.Errorf("expected 4 users after ramp-up, got %d", got)
	}

	pool.KillAll()
}

// Пользователи из RampUp переживают этап и продолжают работу на следующем Steady.
func TestExecuteStage_RampUp_UsersSurviveIntoSteady(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	rampUp := models.Stage{
		ID: 1, Type: models.StageRampUp, Duration: 1, TargetUsers: 3,
		Requests: makeRequests(),
	}
	steady := models.Stage{
		ID: 2, Type: models.StageSteady, Duration: 1, TargetUsers: 3,
		Requests: makeRequests(),
	}

	engine.ExecuteStage(context.Background(), rampUp, pool, results)
	afterRampUp := pool.Len()

	engine.ExecuteStage(context.Background(), steady, pool, results)
	afterSteady := pool.Len()

	// Steady не должен убивать и пересоздавать пользователей — дельта = 0.
	if afterRampUp != 3 {
		t.Errorf("expected 3 users after ramp-up, got %d", afterRampUp)
	}
	if afterSteady != 3 {
		t.Errorf("expected 3 users after steady (no change), got %d", afterSteady)
	}

	pool.KillAll()
}

// Если пул уже >= TargetUsers — RampUp ничего не добавляет.
func TestExecuteStage_RampUp_NoOpWhenAtOrAboveTarget(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	// Сначала steady до 5.
	setup := models.Stage{
		ID: 1, Type: models.StageSteady, Duration: 1, TargetUsers: 5,
		Requests: makeRequests(),
	}
	engine.ExecuteStage(context.Background(), setup, pool, results)

	// RampUp до 3 — должен быть no-op.
	rampUp := models.Stage{
		ID: 2, Type: models.StageRampUp, Duration: 1, TargetUsers: 3,
		Requests: makeRequests(),
	}
	engine.ExecuteStage(context.Background(), rampUp, pool, results)

	if got := pool.Len(); got != 5 {
		t.Errorf("expected pool size unchanged at 5, got %d", got)
	}

	pool.KillAll()
}

// ── StageRampDown ─────────────────────────────────────────────────────────────

// После RampDown пул должен содержать TargetUsers пользователей.
func TestExecuteStage_RampDown_ReachesTarget(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	setup := models.Stage{
		ID: 1, Type: models.StageSteady, Duration: 1, TargetUsers: 5,
		Requests: makeRequests(),
	}
	engine.ExecuteStage(context.Background(), setup, pool, results)

	rampDown := models.Stage{
		ID: 2, Type: models.StageRampDown, Duration: 1, TargetUsers: 2,
		Requests: makeRequests(),
	}
	engine.ExecuteStage(context.Background(), rampDown, pool, results)

	if got := pool.Len(); got != 2 {
		t.Errorf("expected 2 users after ramp-down, got %d", got)
	}

	pool.KillAll()
}

// Если пул уже <= TargetUsers — RampDown ничего не убивает.
func TestExecuteStage_RampDown_NoOpWhenAtOrBelowTarget(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	setup := models.Stage{
		ID: 1, Type: models.StageSteady, Duration: 1, TargetUsers: 2,
		Requests: makeRequests(),
	}
	engine.ExecuteStage(context.Background(), setup, pool, results)

	rampDown := models.Stage{
		ID: 2, Type: models.StageRampDown, Duration: 1, TargetUsers: 5,
		Requests: makeRequests(),
	}
	engine.ExecuteStage(context.Background(), rampDown, pool, results)

	if got := pool.Len(); got != 2 {
		t.Errorf("expected pool size unchanged at 2, got %d", got)
	}

	pool.KillAll()
}

// ── Полный сценарий RampUp → Steady → RampDown ────────────────────────────────

func TestExecuteStage_FullLifecycle(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	ctx := context.Background()

	stages := []models.Stage{
		{ID: 1, Type: models.StageRampUp, Duration: 1, TargetUsers: 5, Requests: makeRequests()},
		{ID: 2, Type: models.StageSteady, Duration: 1, TargetUsers: 5, Requests: makeRequests()},
		{ID: 3, Type: models.StageRampDown, Duration: 1, TargetUsers: 0, Requests: makeRequests()},
	}

	for _, stage := range stages {
		engine.ExecuteStage(ctx, stage, pool, results)
	}

	if got := pool.Len(); got != 0 {
		t.Errorf("expected 0 users after full ramp-down, got %d", got)
	}
}

// ── Отмена контекста ──────────────────────────────────────────────────────────

// При отмене ctx ExecuteStage должен завершиться быстро.
func TestExecuteStage_ContextCancel_ExitsQuickly(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	ctx, cancel := context.WithCancel(context.Background())

	stage := models.Stage{
		ID: 1, Type: models.StageSteady, Duration: 30, TargetUsers: 3,
		Requests: makeRequests(),
	}

	time.AfterFunc(200*time.Millisecond, cancel)

	start := time.Now()
	engine.ExecuteStage(ctx, stage, pool, results)
	elapsed := time.Since(start)

	if elapsed >= 2*time.Second {
		t.Errorf("expected fast exit on cancel, took %v", elapsed)
	}

	pool.KillAll()
}

// ── Неизвестный тип этапа ─────────────────────────────────────────────────────

func TestExecuteStage_UnknownType_NoResults(t *testing.T) {
	engine, attacker := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	stage := models.Stage{
		ID: 1, Type: "UNKNOWN", Duration: 1, TargetUsers: 3,
		Requests: makeRequests(),
	}

	engine.ExecuteStage(context.Background(), stage, pool, results)
	close(results)

	if attacker.Calls() != 0 {
		t.Errorf("expected 0 calls for unknown stage type, got %d", attacker.Calls())
	}
	if got := pool.Len(); got != 0 {
		t.Errorf("expected empty pool for unknown stage type, got %d", got)
	}
}

// ── Невалидный генератор ──────────────────────────────────────────────────────

// Если список запросов пустой — ExecuteStage должен вернуться без паники.
func TestExecuteStage_EmptyRequests_NoOp(t *testing.T) {
	engine, _ := newTestEngine()
	pool := NewUserPool()
	results := makeResults()

	stage := models.Stage{
		ID: 1, Type: models.StageSteady, Duration: 1, TargetUsers: 3,
		Requests: []models.TestRequest{}, // пустой список
	}

	// Не должно быть паники.
	engine.ExecuteStage(context.Background(), stage, pool, results)
	close(results)

	if got := pool.Len(); got != 0 {
		t.Errorf("expected empty pool on generator error, got %d", got)
	}
}
