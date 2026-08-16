// Package store holds in-memory frames, alerts, and reference data.
package store

import (
	"fmt"
	"sort"
	"sync"

	"fleet-backend/config"
	"fleet-backend/models"
)

type TripVehicleInfo struct {
	VehicleID      string `json:"vehicle_id"`
	RegistrationNo string `json:"registration_no"`
}

type TripDriverInfo struct {
	DriverID string `json:"driver_id"`
	Name     string `json:"name"`
}

type TripListItem struct {
	TripID      string          `json:"trip_id"`
	Vehicle     TripVehicleInfo `json:"vehicle"`
	Driver      TripDriverInfo  `json:"driver"`
	RouteName   string          `json:"route_name"`
	Status      string          `json:"status"`
	StartedAt   string          `json:"started_at"`
	EndedAt     *string         `json:"ended_at"`
	DurationS   int             `json:"duration_s"`
	FrameCount  int             `json:"frame_count"`
	AlertCounts map[string]int  `json:"alert_counts"`
}

type TripsResponse struct {
	Items    []TripListItem `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Total    int            `json:"total"`
}

type RoutePoint struct {
	Seq        int     `json:"seq"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	SpeedKph   float64 `json:"speed_kph"`
	RecordedAt string  `json:"recorded_at"`
}

type RouteAlert struct {
	AlertID  string  `json:"alert_id"`
	Type     string  `json:"type"`
	Severity string  `json:"severity"`
	Seq      int     `json:"seq"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
}

type TripRouteResponse struct {
	TripID     string       `json:"trip_id"`
	PointCount int          `json:"point_count"`
	BBox       [4]float64   `json:"bbox"` // [minLat, minLon, maxLat, maxLon]
	Points     []RoutePoint `json:"points"`
	Alerts     []RouteAlert `json:"alerts"`
}

type Store struct {
	mu         sync.RWMutex
	trips      map[string]config.Trip
	tripOrder  []string
	drivers    map[string]config.Driver
	vehicles   map[string]config.Vehicle
	frames     map[string]models.TelemetryFrame
	tripFrames map[string][]models.TelemetryFrame
	alerts     []models.Alert
	alertKeys  map[string]bool
}

func New() *Store {
	return &Store{
		trips:      make(map[string]config.Trip),
		tripOrder:  make([]string, 0),
		drivers:    make(map[string]config.Driver),
		vehicles:   make(map[string]config.Vehicle),
		frames:     make(map[string]models.TelemetryFrame),
		tripFrames: make(map[string][]models.TelemetryFrame),
		alerts:     make([]models.Alert, 0),
		alertKeys:  make(map[string]bool),
	}
}

func (s *Store) LoadReferenceData(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range cfg.Trips {
		s.trips[t.TripID] = t
		s.tripOrder = append(s.tripOrder, t.TripID)
	}
	for _, d := range cfg.Drivers {
		s.drivers[d.DriverID] = d
	}
	for _, v := range cfg.Vehicles {
		s.vehicles[v.VehicleID] = v
	}
}

func (s *Store) SaveFrames(incoming []models.TelemetryFrame) (accepted int, duplicates int, newFrames []models.TelemetryFrame) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, f := range incoming {
		if _, exists := s.frames[f.FrameId]; exists {
			duplicates++
			continue
		}
		s.frames[f.FrameId] = f
		s.tripFrames[f.TripId] = append(s.tripFrames[f.TripId], f)
		newFrames = append(newFrames, f)
		accepted++
	}

	return accepted, duplicates, newFrames
}

func (s *Store) SaveAlert(alert models.Alert) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%d", alert.TripID, alert.Type, alert.TriggerSeq)
	if s.alertKeys[key] {
		return false
	}
	s.alertKeys[key] = true
	s.alerts = append(s.alerts, alert)
	return true
}

func (s *Store) GetAlerts(severity, tripID string) []models.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Alert, 0, len(s.alerts))
	// Return in reverse chronological order
	for i := len(s.alerts) - 1; i >= 0; i-- {
		a := s.alerts[i]
		if severity != "" && a.Severity != severity {
			continue
		}
		if tripID != "" && a.TripID != tripID {
			continue
		}
		result = append(result, a)
	}
	return result
}

func (s *Store) GetAlertByID(alertID string) (models.Alert, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, a := range s.alerts {
		if a.AlertID == alertID {
			return a, true
		}
	}
	return models.Alert{}, false
}

func (s *Store) GetTrips() TripsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]TripListItem, 0, len(s.tripOrder))

	for _, tripID := range s.tripOrder {
		trip := s.trips[tripID]
		veh := s.vehicles[trip.VehicleID]
		drv := s.drivers[trip.DriverID]

		// Compute alert counts
		counts := map[string]int{
			"CRITICAL": 0,
			"HIGH":     0,
			"MEDIUM":   0,
			"LOW":      0,
		}

		for _, a := range s.alerts {
			if a.TripID == tripID {
				counts[a.Severity]++
			}
		}

		status := trip.Status
		if status == "COMPLETED" && (counts["CRITICAL"] > 0 || counts["HIGH"] > 0) {
			status = "COMPLETED_WITH_INCIDENT"
		}

		frames := s.tripFrames[tripID]
		frameCount := len(frames)
		if frameCount == 0 && trip.FrameCount > 0 {
			frameCount = trip.FrameCount
		}

		durationS := trip.DurationS
		if len(frames) > 1 {
			durationS = len(frames) * 2
		}

		items = append(items, TripListItem{
			TripID: trip.TripID,
			Vehicle: TripVehicleInfo{
				VehicleID:      trip.VehicleID,
				RegistrationNo: veh.RegistrationNo,
			},
			Driver: TripDriverInfo{
				DriverID: trip.DriverID,
				Name:     drv.Name,
			},
			RouteName:   trip.RouteName,
			Status:      status,
			StartedAt:   trip.StartedAt,
			EndedAt:     trip.EndedAt,
			DurationS:   durationS,
			FrameCount:  frameCount,
			AlertCounts: counts,
		})
	}

	return TripsResponse{
		Items:    items,
		Page:     1,
		PageSize: len(items),
		Total:    len(items),
	}
}

func (s *Store) GetTripRoute(tripID string) (*TripRouteResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trip, exists := s.trips[tripID]
	if !exists {
		return nil, fmt.Errorf("trip %s not found", tripID)
	}
	_ = trip

	frames := s.tripFrames[tripID]
	// Sort frames by Seq
	sortedFrames := make([]models.TelemetryFrame, len(frames))
	copy(sortedFrames, frames)
	sort.Slice(sortedFrames, func(i, j int) bool {
		return sortedFrames[i].Seq < sortedFrames[j].Seq
	})

	points := make([]RoutePoint, len(sortedFrames))
	var minLat, minLon, maxLat, maxLon float64
	if len(sortedFrames) > 0 {
		minLat, maxLat = sortedFrames[0].Location.Lat, sortedFrames[0].Location.Lat
		minLon, maxLon = sortedFrames[0].Location.Lon, sortedFrames[0].Location.Lon
	}

	for i, f := range sortedFrames {
		points[i] = RoutePoint{
			Seq:        f.Seq,
			Lat:        f.Location.Lat,
			Lon:        f.Location.Lon,
			SpeedKph:   f.SpeedKph,
			RecordedAt: f.RecordedAt,
		}
		if f.Location.Lat < minLat {
			minLat = f.Location.Lat
		}
		if f.Location.Lat > maxLat {
			maxLat = f.Location.Lat
		}
		if f.Location.Lon < minLon {
			minLon = f.Location.Lon
		}
		if f.Location.Lon > maxLon {
			maxLon = f.Location.Lon
		}
	}

	routeAlerts := make([]RouteAlert, 0)
	for _, a := range s.alerts {
		if a.TripID == tripID {
			routeAlerts = append(routeAlerts, RouteAlert{
				AlertID:  a.AlertID,
				Type:     a.Type,
				Severity: a.Severity,
				Seq:      a.TriggerSeq,
				Lat:      a.Location.Lat,
				Lon:      a.Location.Lon,
			})
		}
	}

	return &TripRouteResponse{
		TripID:     tripID,
		PointCount: len(points),
		BBox:       [4]float64{minLat, minLon, maxLat, maxLon},
		Points:     points,
		Alerts:     routeAlerts,
	}, nil
}

func (s *Store) TripCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.trips)
}

func (s *Store) DriverCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.drivers)
}

func (s *Store) VehicleCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.vehicles)
}
