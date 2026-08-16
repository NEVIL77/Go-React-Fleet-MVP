// Package bus provides an in-process channel implementation of EventBus.
package bus

import (
	"context"

	"fleet-backend/models"
)

type ChannelBus struct {
	ch chan *models.TelemetryFrame
}

func NewChannelBus(buffer int) *ChannelBus {
	if buffer <= 0 {
		buffer = 256
	}
	return &ChannelBus{ch: make(chan *models.TelemetryFrame, buffer)}
}

func (b *ChannelBus) Publish(_ context.Context, frame *models.TelemetryFrame) error {
	b.ch <- frame
	return nil
}

func (b *ChannelBus) Subscribe(_ context.Context) <-chan *models.TelemetryFrame {
	return b.ch
}
