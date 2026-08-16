package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fleet-backend/bus"
	"fleet-backend/config"
	"fleet-backend/detection"
	"fleet-backend/handlers"
	"fleet-backend/middleware"
	"fleet-backend/store"
)

func main() {
	dataDir := envOr("DATA_DIR", "../data")

	cfg, err := config.Load(dataDir)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	st := store.New()
	st.LoadReferenceData(cfg)
	log.Printf("loaded %d trips, %d drivers, %d vehicles",
		st.TripCount(), st.DriverCount(), st.VehicleCount())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	eventBus := bus.NewChannelBus(1024)
	detection.StartConsumer(ctx, eventBus, st)

	mux := http.NewServeMux()

	// Ingest endpoint
	mux.Handle("POST /api/v1/ingest/telemetry",
		middleware.Auth(cfg.Token, handlers.Ingest(st, eventBus)))

	// Read endpoints
	mux.Handle("GET /api/v1/trips",
		middleware.Auth(cfg.Token, handlers.ListTrips(st)))
	mux.Handle("GET /api/v1/trips/{id}/route",
		middleware.Auth(cfg.Token, handlers.GetTripRoute(st)))
	mux.Handle("GET /api/v1/alerts",
		middleware.Auth(cfg.Token, handlers.ListAlerts(st)))

	// Snapshots endpoints (accessible for <img> tags)
	mux.Handle("GET /api/v1/snapshots/{event_id}/{camera}",
		handlers.ServeSnapshot(dataDir))

	// Health check endpoint
	startTime := time.Now()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "ok",
			"uptime_s": int(time.Since(startTime).Seconds()),
			"trips":    st.TripCount(),
			"alerts":   len(st.GetAlerts("", "")),
		})
	})

	handlerWithCORS := middleware.CORS(mux)

	addr := envOr("ADDR", ":8080")
	log.Printf("listening on %s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: handlerWithCORS,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
