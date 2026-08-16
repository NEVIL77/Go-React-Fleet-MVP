// Package config loads reference JSON from the data directory at startup.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Organization struct {
	APIToken string `json:"api_token"`
}

type Trip struct {
	TripID     string  `json:"trip_id"`
	VehicleID  string  `json:"vehicle_id"`
	DriverID   string  `json:"driver_id"`
	RouteName  string  `json:"route_name"`
	Status     string  `json:"status"`
	StartedAt  string  `json:"started_at"`
	EndedAt    *string `json:"ended_at"`
	DurationS  int     `json:"duration_s"`
	FrameCount int     `json:"frame_count"`
}

type Driver struct {
	DriverID string `json:"driver_id"`
	Name     string `json:"name"`
}

type Vehicle struct {
	VehicleID      string `json:"vehicle_id"`
	RegistrationNo string `json:"registration_no"`
}

type Config struct {
	Token    string
	Trips    []Trip
	Drivers  []Driver
	Vehicles []Vehicle
}

func Load(dataDir string) (*Config, error) {
	org, err := readJSON[Organization](filepath.Join(dataDir, "organization.json"))
	if err != nil {
		return nil, fmt.Errorf("organization.json: %w", err)
	}

	trips, err := readJSON[[]Trip](filepath.Join(dataDir, "trips.json"))
	if err != nil {
		return nil, fmt.Errorf("trips.json: %w", err)
	}

	drivers, err := readJSON[[]Driver](filepath.Join(dataDir, "drivers.json"))
	if err != nil {
		return nil, fmt.Errorf("drivers.json: %w", err)
	}

	vehicles, err := readJSON[[]Vehicle](filepath.Join(dataDir, "vehicles.json"))
	if err != nil {
		return nil, fmt.Errorf("vehicles.json: %w", err)
	}

	return &Config{
		Token:    org.APIToken,
		Trips:    *trips,
		Drivers:  *drivers,
		Vehicles: *vehicles,
	}, nil
}

func readJSON[T any](path string) (*T, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
