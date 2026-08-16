// Package handlers exposes HTTP handlers for the fleet API.
package handlers

import (
	"encoding/json"
	"net/http"

	"fleet-backend/models"
	"fleet-backend/store"
)

type AlertsResponse struct {
	Items               []models.Alert `json:"items"`
	UnacknowledgedCount int            `json:"unacknowledged_count"`
	Page                int            `json:"page"`
	PageSize            int            `json:"page_size"`
	Total               int            `json:"total"`
}

func ListAlerts(st *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		severity := r.URL.Query().Get("severity")
		tripID := r.URL.Query().Get("trip_id")

		items := st.GetAlerts(severity, tripID)
		unack := 0
		for _, a := range items {
			if !a.Acknowledged {
				unack++
			}
		}

		resp := AlertsResponse{
			Items:               items,
			UnacknowledgedCount: unack,
			Page:                1,
			PageSize:            len(items),
			Total:               len(items),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}
