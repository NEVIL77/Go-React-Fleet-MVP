// Package detection implements async frame processing and alert rules.
package detection

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"fleet-backend/models"
	"fleet-backend/store"
)

var eventIDRegex = regexp.MustCompile(`evt_[a-f0-9]+`)

type IdleState struct {
	Active     bool
	StartSeq   int
	StartTime  time.Time
	FiredAlert bool
}

type TripDetectionState struct {
	History        []models.TelemetryFrame
	LastDrowsyTime time.Time
	LastDrowsySeq  int
	Idle           IdleState
}

type Engine struct {
	mu     sync.Mutex
	states map[string]*TripDetectionState
}

func NewEngine() *Engine {
	return &Engine{
		states: make(map[string]*TripDetectionState),
	}
}

func (e *Engine) getTripState(tripID string) *TripDetectionState {
	if state, exists := e.states[tripID]; exists {
		return state
	}
	state := &TripDetectionState{
		History:       make([]models.TelemetryFrame, 0, 10),
		LastDrowsySeq: -999,
	}
	e.states[tripID] = state
	return state
}

func parseTime(ts string) time.Time {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return t
}

func extractEventID(frame *models.TelemetryFrame) string {
	if frame.Cameras.RoadFacing.SnapshotUrl != nil {
		if match := eventIDRegex.FindString(*frame.Cameras.RoadFacing.SnapshotUrl); match != "" {
			return match
		}
	}
	if frame.Cameras.DriverFacing.SnapshotUrl != nil {
		if match := eventIDRegex.FindString(*frame.Cameras.DriverFacing.SnapshotUrl); match != "" {
			return match
		}
	}
	return fmt.Sprintf("evt_%s_%d", frame.TripId, frame.Seq)
}

func makeSnapshotURLs(eventID string, road bool, driver bool) models.Snapshots {
	var roadURL, driverURL *string
	if road {
		r := fmt.Sprintf("/api/v1/snapshots/%s/road_facing", eventID)
		roadURL = &r
	}
	if driver {
		d := fmt.Sprintf("/api/v1/snapshots/%s/driver_facing", eventID)
		driverURL = &d
	}
	return models.Snapshots{
		RoadFacing:   roadURL,
		DriverFacing: driverURL,
	}
}

// EvaluateFrame processes a single frame statefully against all detection rules.
func (e *Engine) EvaluateFrame(st *store.Store, frame *models.TelemetryFrame) []models.Alert {
	e.mu.Lock()
	defer e.mu.Unlock()

	state := e.getTripState(frame.TripId)
	alerts := make([]models.Alert, 0)
	frameTime := parseTime(frame.RecordedAt)

	// --- 1. Impact Detection: Rule 3.1 (COLLISION) & Rule 3.2 (UNVERIFIED_IMPACT) ---
	if frame.Sensors.CrashSensor.ImpactDetected && frame.Sensors.CrashSensor.PeakG >= 3.5 {
		corroboratedBy := make([]string, 0)
		corroboratedBy = append(corroboratedBy, "CRASH_SENSOR")

		// Check FCW on current frame or prior 2 frames
		fcwFound := frame.Cameras.RoadFacing.ForwardCollisionWarning
		if !fcwFound && len(state.History) > 0 {
			lastIdx := len(state.History) - 1
			if state.History[lastIdx].Cameras.RoadFacing.ForwardCollisionWarning {
				fcwFound = true
			} else if lastIdx >= 1 && state.History[lastIdx-1].Cameras.RoadFacing.ForwardCollisionWarning {
				fcwFound = true
			}
		}

		if fcwFound {
			corroboratedBy = append(corroboratedBy, "ROAD_CAMERA_FCW")
		}

		// Check speed drop: previous_frame.speed - current.speed >= 10.0 kph
		if len(state.History) > 0 {
			prevSpeed := state.History[len(state.History)-1].SpeedKph
			if prevSpeed-frame.SpeedKph >= 10.0 {
				corroboratedBy = append(corroboratedBy, "SPEED_DROP")
			}
		}

		eventID := extractEventID(frame)
		impactDir := frame.Sensors.CrashSensor.ImpactDirection

		if len(corroboratedBy) > 1 {
			// Rule 3.1: COLLISION (CRITICAL)
			alert := models.Alert{
				AlertID:         fmt.Sprintf("alt_%s", eventID),
				EventID:         eventID,
				Type:            "COLLISION",
				Severity:        "CRITICAL",
				TripID:          frame.TripId,
				VehicleID:       frame.VehicleId,
				DriverID:        frame.DriverId,
				DetectedAt:      frame.RecordedAt,
				TriggerFrameID:  frame.FrameId,
				TriggerSeq:      frame.Seq,
				Location:        models.AlertLocation{Lat: frame.Location.Lat, Lon: frame.Location.Lon},
				SpeedKph:        frame.SpeedKph,
				GForce:          frame.Sensors.CrashSensor.PeakG,
				ImpactDirection: &impactDir,
				Snapshots:       makeSnapshotURLs(eventID, true, true),
				CorroboratedBy:  corroboratedBy,
				Message: fmt.Sprintf("Collision detected: %.1fg %s impact (delta-v: %.1f kph)",
					frame.Sensors.CrashSensor.PeakG, impactDir, frame.Sensors.CrashSensor.DeltaVKph),
			}
			alerts = append(alerts, alert)
		} else {
			// Rule 3.2: UNVERIFIED_IMPACT (LOW)
			alert := models.Alert{
				AlertID:         fmt.Sprintf("alt_%s", eventID),
				EventID:         eventID,
				Type:            "UNVERIFIED_IMPACT",
				Severity:        "LOW",
				TripID:          frame.TripId,
				VehicleID:       frame.VehicleId,
				DriverID:        frame.DriverId,
				DetectedAt:      frame.RecordedAt,
				TriggerFrameID:  frame.FrameId,
				TriggerSeq:      frame.Seq,
				Location:        models.AlertLocation{Lat: frame.Location.Lat, Lon: frame.Location.Lon},
				SpeedKph:        frame.SpeedKph,
				GForce:          frame.Sensors.CrashSensor.PeakG,
				ImpactDirection: &impactDir,
				Snapshots:       makeSnapshotURLs(eventID, true, true),
				CorroboratedBy:  corroboratedBy,
				Message: fmt.Sprintf("Unverified impact: %.1fg %s impact without camera or speed corroboration (likely road surface)",
					frame.Sensors.CrashSensor.PeakG, impactDir),
			}
			alerts = append(alerts, alert)
		}
	}

	// --- 2. Drowsy Driver: Rule 3.4 (DROWSY_DRIVER) ---
	driverState := frame.Cameras.DriverFacing.DriverState
	isDrowsyState := driverState == "DROWSY" || driverState == "EYES_CLOSED"
	if isDrowsyState && frame.Cameras.DriverFacing.EyeClosureMs >= 1500 && frame.SpeedKph > 20.0 {
		// 60-second debounce check (or >= 30 frames)
		debounced := false
		if !state.LastDrowsyTime.IsZero() && frameTime.Sub(state.LastDrowsyTime) < 60*time.Second {
			debounced = true
		} else if frame.Seq-state.LastDrowsySeq < 30 {
			debounced = true
		}

		if !debounced {
			state.LastDrowsyTime = frameTime
			state.LastDrowsySeq = frame.Seq

			eventID := extractEventID(frame)
			alert := models.Alert{
				AlertID:        fmt.Sprintf("alt_%s", eventID),
				EventID:        eventID,
				Type:           "DROWSY_DRIVER",
				Severity:       "HIGH",
				TripID:         frame.TripId,
				VehicleID:      frame.VehicleId,
				DriverID:       frame.DriverId,
				DetectedAt:     frame.RecordedAt,
				TriggerFrameID: frame.FrameId,
				TriggerSeq:     frame.Seq,
				Location:       models.AlertLocation{Lat: frame.Location.Lat, Lon: frame.Location.Lon},
				SpeedKph:       frame.SpeedKph,
				GForce:         frame.Sensors.AccelerometerG.Magnitude,
				Snapshots:      makeSnapshotURLs(eventID, false, true),
				CorroboratedBy: []string{"DRIVER_CAMERA_DMS"},
				Message: fmt.Sprintf("Drowsy driver detected: %s with eye closure %d ms at %.1f kph",
					driverState, frame.Cameras.DriverFacing.EyeClosureMs, frame.SpeedKph),
			}
			alerts = append(alerts, alert)
		}
	}

	// --- 3. Idle No Motion: Rule 3.5 (IDLE_NO_MOTION) ---
	isIdleFrame := frame.EngineOn && frame.SpeedKph < 2.0 && !frame.Context.TrafficCongestion
	if isIdleFrame {
		if !state.Idle.Active {
			state.Idle.Active = true
			state.Idle.StartSeq = frame.Seq
			state.Idle.StartTime = frameTime
			state.Idle.FiredAlert = false
		}

		durationSec := frameTime.Sub(state.Idle.StartTime).Seconds()
		seqDiff := frame.Seq - state.Idle.StartSeq

		if !state.Idle.FiredAlert && (durationSec >= 180 || seqDiff >= 90) {
			state.Idle.FiredAlert = true
			eventID := extractEventID(frame)
			alert := models.Alert{
				AlertID:        fmt.Sprintf("alt_%s", eventID),
				EventID:        eventID,
				Type:           "IDLE_NO_MOTION",
				Severity:       "MEDIUM",
				TripID:         frame.TripId,
				VehicleID:      frame.VehicleId,
				DriverID:       frame.DriverId,
				DetectedAt:     frame.RecordedAt,
				TriggerFrameID: frame.FrameId,
				TriggerSeq:     frame.Seq,
				Location:       models.AlertLocation{Lat: frame.Location.Lat, Lon: frame.Location.Lon},
				SpeedKph:       frame.SpeedKph,
				GForce:         frame.Sensors.AccelerometerG.Magnitude,
				Snapshots:      makeSnapshotURLs(eventID, true, false),
				CorroboratedBy: []string{"ENGINE_TELEMETRY", "WHEEL_SPEED"},
				Message:        "Vehicle idle for ≥ 180s with engine on outside traffic congestion",
			}
			alerts = append(alerts, alert)
		}
	} else {
		// Condition broken: reset idle tracker
		state.Idle.Active = false
		state.Idle.FiredAlert = false
	}

	// Maintain history (keep last 5 frames)
	state.History = append(state.History, *frame)
	if len(state.History) > 5 {
		state.History = state.History[len(state.History)-5:]
	}

	// Save alerts to store
	for _, a := range alerts {
		st.SaveAlert(a)
	}

	return alerts
}
