package load

import (
	"context"
	"sync"

	"github.com/sibhellyx/tester/internal/models"
)

// virtualUser представляет одного активного виртуального пользователя.
type virtualUser struct {
	cancel context.CancelFunc
	done   <-chan struct{}
}

// UserPool управляет жизненным циклом виртуальных пользователей.
// Все workers пула разделяют один SharedGenerator — при смене этапа
// достаточно вызвать SetStage, чтобы все workers мгновенно перешли
// на новые запросы без перезапуска горутин.
type UserPool struct {
	mu        sync.Mutex
	users     []*virtualUser
	wg        sync.WaitGroup
	sharedGen *SharedGenerator
}

// NewUserPool создаёт пустой пул с инициализированным SharedGenerator.
func NewUserPool() *UserPool {
	return &UserPool{
		sharedGen: newSharedGenerator(),
	}
}

// SetStage атомарно обновляет генератор и stageID для всех workers пула.
// Должен вызываться перед каждым этапом — до Spawn/Kill.
func (p *UserPool) SetStage(stageID int, gen RequestGenerator) {
	p.sharedGen.Set(stageID, gen)
}

// Len возвращает текущее количество активных пользователей.
func (p *UserPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.users)
}

// Spawn запускает n новых виртуальных пользователей.
// Все они используют общий SharedGenerator пула.
//
// ctx должен быть родительским контекстом теста (не stageCtx),
// чтобы пользователи переживали смену этапов.
func (p *UserPool) Spawn(
	ctx context.Context,
	n int,
	attacker Shooter,
	results chan<- models.CallResult,
) {
	for i := 0; i < n; i++ {
		userCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})

		vu := &virtualUser{
			cancel: cancel,
			done:   done,
		}

		p.wg.Add(1)
		go func() {
			defer close(done)
			RunVirtualUser(userCtx, &p.wg, p.sharedGen, attacker, results)
		}()

		p.mu.Lock()
		p.users = append(p.users, vu)
		p.mu.Unlock()
	}
}

// Kill останавливает последних n пользователей из пула (LIFO) и ждёт их завершения.
func (p *UserPool) Kill(n int) {
	p.mu.Lock()

	current := len(p.users)
	if n > current {
		n = current
	}

	toKill := make([]*virtualUser, n)
	copy(toKill, p.users[current-n:])
	p.users = p.users[:current-n]

	p.mu.Unlock()

	for _, vu := range toKill {
		vu.cancel()
		<-vu.done
	}
}

// KillAll останавливает всех пользователей в пуле и ждёт завершения всех горутин.
func (p *UserPool) KillAll() {
	p.mu.Lock()
	users := p.users
	p.users = nil
	p.mu.Unlock()

	for _, vu := range users {
		vu.cancel()
	}

	for _, vu := range users {
		<-vu.done
	}

	p.wg.Wait()
}
