// Package handlers exposes HTTP handlers for the fleet API.
package handlers

import (
	"encoding/json"
	"net/http"

	"fleet-backend/store"
)

func ListTrips(st *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trips := st.GetTrips()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(trips)
	})
}

func GetTripRoute(st *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tripID := r.PathValue("id")
		if tripID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    "INVALID_TRIP_ID",
					"message": "trip id required",
					"details": map[string]any{},
				},
			})
			return
		}

		route, err := st.GetTripRoute(tripID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    "TRIP_NOT_FOUND",
					"message": err.Error(),
					"details": map[string]any{},
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(route)
	})
}
