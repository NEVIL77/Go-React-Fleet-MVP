// Package detection runs a background consumer that reads frames from the bus.
package detection

import (
	"context"

	"fleet-backend/bus"
	"fleet-backend/store"
)

func StartConsumer(ctx context.Context, eventBus bus.EventBus, st *store.Store) {
	engine := NewEngine()
	ch := eventBus.Subscribe(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-ch:
				if !ok {
					return
				}
				_ = engine.EvaluateFrame(st, frame)
			}
		}
	}()
}
