package load

import (
	"context"
	"sync"

	"github.com/sibhellyx/tester/internal/models"
)

// virtualUser представляет одного активного виртуального пользователя.
// Каждый пользователь работает в своей горутине и может быть остановлен
// индивидуально через cancel, либо дождаться его завершения через done.
type virtualUser struct {
	cancel context.CancelFunc
	done   <-chan struct{} // закрывается когда горутина пользователя завершилась
}

// userPool управляет жизненным циклом виртуальных пользователей.
// Pool живёт на уровне Engine и сохраняет пользователей между этапами теста.
// Это позволяет пользователям, запущенным на RampUp, продолжать работу на Steady.
type UserPool struct {
	mu    sync.Mutex
	users []*virtualUser
	wg    sync.WaitGroup // единый WaitGroup на весь пул
}

// newUserPool создаёт пустой пул пользователей.
func NewUserPool() *UserPool {
	return &UserPool{}
}

// Len возвращает текущее количество активных пользователей.
// Безопасен для конкурентного вызова.
func (p *UserPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.users)
}

// Spawn запускает n новых виртуальных пользователей и добавляет их в пул.
//
// Важно: ctx должен быть родительским контекстом теста, а НЕ stageCtx.
// Это позволяет пользователям пережить завершение этапа (например RampUp → Steady).
// Пользователь завершится только когда:
//   - будет вызван его индивидуальный cancel (через Kill/KillAll)
//   - будет отменён родительский ctx (StopTest / глобальный таймаут)
func (p *UserPool) Spawn(
	ctx context.Context,
	n int,
	generator RequestGeneratorInterface,
	attacker AttackerToolInterface,
	results chan<- models.CallResult,
) {
	for i := 0; i < n; i++ {
		// Каждый пользователь получает свой дочерний контекст.
		// При отмене родительского ctx — все пользователи тоже завершатся.
		userCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})

		vu := &virtualUser{
			cancel: cancel,
			done:   done,
		}

		p.wg.Add(1)
		go func() {
			// done закрывается строго после выхода из RunVirtualUser.
			// Это гарантирует что при чтении <-vu.done горутина уже не работает.
			defer close(done)
			RunVirtualUser(userCtx, &p.wg, generator, attacker, results)
		}()

		p.mu.Lock()
		p.users = append(p.users, vu)
		p.mu.Unlock()
	}
}

// Kill останавливает последних n пользователей из пула и ждёт их завершения.
// Пользователи удаляются с конца слайса (LIFO) — последние добавленные уходят первыми.
// Если n больше текущего размера пула — останавливаются все пользователи.
//
// Метод блокируется до полного завершения всех отменённых горутин.
// Это гарантирует что после возврата из Kill горутины не работают.
func (p *UserPool) Kill(n int) {
	p.mu.Lock()

	current := len(p.users)
	if n > current {
		n = current
	}

	// Вырезаем последних n пользователей из пула.
	toKill := make([]*virtualUser, n)
	copy(toKill, p.users[current-n:])
	p.users = p.users[:current-n]

	p.mu.Unlock()

	// Отменяем контексты и ждём завершения горутин.
	// Делаем это вне lock чтобы не держать мьютекс во время ожидания.
	for _, vu := range toKill {
		vu.cancel()
		<-vu.done // блокируемся до полного завершения горутины
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
		<-vu.done
	}

	// Дожидаемся что все wg.Done() были вызваны.
	p.wg.Wait()
}
