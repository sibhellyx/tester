package load

import (
	"sync"

	"github.com/sibhellyx/tester/internal/models"
)

// RequestGenerator - интерфейс генератора, предоставляет метод для получения следующего запроса.
type RequestGenerator interface {
	Next() *models.TestRequest
}

// SharedGenerator — потокобезопасный переключаемый генератор, общий для всех workers пула.
// Вызов Set атомарно заменяет источник запросов и stageID, поэтому существующие workers
// мгновенно переходят на запросы нового этапа без перезапуска горутин.
type SharedGenerator struct {
	mu      sync.RWMutex
	gen     RequestGenerator
	stageID int
}

func newSharedGenerator() *SharedGenerator {
	return &SharedGenerator{}
}

// Set атомарно заменяет текущий генератор и stageID.
func (g *SharedGenerator) Set(stageID int, gen RequestGenerator) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.gen = gen
	g.stageID = stageID
}

// Next возвращает следующий запрос из текущего генератора.
func (g *SharedGenerator) Next() *models.TestRequest {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.gen == nil {
		return nil
	}
	return g.gen.Next()
}

// CurrentStageID возвращает stageID, установленный последним вызовом Set.
func (g *SharedGenerator) CurrentStageID() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.stageID
}
