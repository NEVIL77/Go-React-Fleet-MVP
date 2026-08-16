// Package handlers exposes HTTP handlers for the fleet API.
package handlers

import (
	"encoding/json"
	"net/http"

	"fleet-backend/bus"
	"fleet-backend/models"
	"fleet-backend/store"
)

type IngestRequest struct {
	Frames []models.TelemetryFrame `json:"frames"`
}

type IngestResponse struct {
	Accepted   int `json:"accepted"`
	Duplicates int `json:"duplicates"`
	Rejected   int `json:"rejected"`
	Queued     int `json:"queued"`
}

func Ingest(st *store.Store, eventBus bus.EventBus) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req IngestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    "INVALID_REQUEST",
					"message": err.Error(),
					"details": map[string]any{},
				},
			})
			return
		}

		accepted, duplicates, newFrames := st.SaveFrames(req.Frames)

		// Publish newly accepted frames to event bus for async detection
		ctx := r.Context()
		queued := 0
		for i := range newFrames {
			if err := eventBus.Publish(ctx, &newFrames[i]); err == nil {
				queued++
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(IngestResponse{
			Accepted:   accepted,
			Duplicates: duplicates,
			Rejected:   0,
			Queued:     queued,
		})
	})
}
