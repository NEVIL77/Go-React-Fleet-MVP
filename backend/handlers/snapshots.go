// Package handlers exposes HTTP handlers for the fleet API.
package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func ServeSnapshot(dataDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eventID := r.PathValue("event_id")
		camera := r.PathValue("camera")

		suffix := ""
		switch camera {
		case "road_facing", "road":
			suffix = "road"
		case "driver_facing", "driver":
			suffix = "driver"
		default:
			http.Error(w, "invalid camera type", http.StatusBadRequest)
			return
		}

		filename := fmt.Sprintf("%s_%s.png", eventID, suffix)
		filePath := filepath.Join(dataDir, "snapshots", filename)

		data, err := os.ReadFile(filePath)
		if err != nil {
			http.Error(w, "snapshot not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	})
}
