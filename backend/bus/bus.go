// Package bus defines the event bus seam between ingest and detection.
package bus

import (
	"context"

	"fleet-backend/models"
)

type EventBus interface {
	Publish(ctx context.Context, frame *models.TelemetryFrame) error
	Subscribe(ctx context.Context) <-chan *models.TelemetryFrame
}
