package chaos

import "context"

// Injector описывает поведение любого хаос-воздействия.
type Injector interface {
	// Inject применяет сбой (ломает систему).
	Inject(ctx context.Context) error
	// Recover устраняет сбой (восстанавливает систему).
	Recover(ctx context.Context) error
	// String возвращает описание сбоя для логов (например: "NetworkDelay 100ms on container ab123").
	String() string
}
