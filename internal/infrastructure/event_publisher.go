package infrastructure

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"ssm-portway/internal/domain"
)

// wailsEventPublisher emite eventos hacia el frontend usando el
// runtime de Wails. El contexto solo esta disponible una vez que la
// app termina de arrancar, por lo que se inyecta despues via
// SetContext en vez del constructor.
type wailsEventPublisher struct {
	mu  sync.RWMutex
	ctx context.Context
}

func NewWailsEventPublisher() domain.EventPublisher {
	return &wailsEventPublisher{}
}

func (p *wailsEventPublisher) SetContext(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ctx = ctx
}

func (p *wailsEventPublisher) Publish(event string, payload any) {
	p.mu.RLock()
	ctx := p.ctx
	p.mu.RUnlock()

	if ctx != nil {
		runtime.EventsEmit(ctx, event, payload)
	}
}
