package detection

import (
	"testing"
	"time"

	"fleet-backend/models"
	"fleet-backend/store"
)

func TestUncorroboratedImpactDoesNotProduceCollision(t *testing.T) {
	engine := NewEngine()
	st := store.New()

	// Frame with crash sensor trigger at 4.2g, but NO forward collision warning and NO speed drop
	frame := models.TelemetryFrame{
		FrameId:    "frm_test_pothole",
		TripId:     "TRIP-TEST",
		VehicleId:  "VEH-001",
		DriverId:   "DRV-001",
		Seq:        100,
		RecordedAt: "2026-08-11T04:30:00Z",
		Location:   models.Location{Lat: 17.55, Lon: 78.47},
		SpeedKph:   40.0,
		Sensors: models.Sensors{
			CrashSensor: models.CrashSensor{
				Status:          "ONLINE",
				ImpactDetected:  true,
				PeakG:           4.2,
				DeltaVKph:       0.2,
				ImpactDirection: "VERTICAL",
			},
		},
		Cameras: models.Cameras{
			RoadFacing: models.RoadFacing{
				Status:                  "ONLINE",
				ForwardCollisionWarning: false,
			},
			DriverFacing: models.DriverFacing{
				Status:      "ONLINE",
				DriverState: "ALERT",
			},
		},
	}

	alerts := engine.EvaluateFrame(st, &frame)

	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.Type == "COLLISION" {
		t.Fatalf("CRITICAL FAILURE: uncorroborated impact produced a COLLISION alert")
	}

	if alert.Type != "UNVERIFIED_IMPACT" {
		t.Errorf("expected UNVERIFIED_IMPACT, got %s", alert.Type)
	}

	if alert.Severity != "LOW" {
		t.Errorf("expected LOW severity, got %s", alert.Severity)
	}
}

func TestCorroboratedImpactProducesCollision(t *testing.T) {
	engine := NewEngine()
	st := store.New()

	// 1. Frame preceding impact with FCW
	prevFrame := models.TelemetryFrame{
		FrameId:    "frm_test_pre",
		TripId:     "TRIP-TEST-2",
		VehicleId:  "VEH-002",
		DriverId:   "DRV-002",
		Seq:        50,
		RecordedAt: "2026-08-11T04:35:58Z",
		SpeedKph:   55.0,
		Cameras: models.Cameras{
			RoadFacing: models.RoadFacing{
				Status:                  "ONLINE",
				ForwardCollisionWarning: true,
			},
			DriverFacing: models.DriverFacing{
				Status:      "ONLINE",
				DriverState: "ALERT",
			},
		},
	}
	engine.EvaluateFrame(st, &prevFrame)

	// 2. Impact frame with 5.2g peak g and speed drop
	impactFrame := models.TelemetryFrame{
		FrameId:    "frm_test_impact",
		TripId:     "TRIP-TEST-2",
		VehicleId:  "VEH-002",
		DriverId:   "DRV-002",
		Seq:        51,
		RecordedAt: "2026-08-11T04:36:00Z",
		SpeedKph:   12.0, // 55.0 -> 12.0 is a 43 kph drop
		Sensors: models.Sensors{
			CrashSensor: models.CrashSensor{
				Status:          "ONLINE",
				ImpactDetected:  true,
				PeakG:           5.2,
				DeltaVKph:       43.0,
				ImpactDirection: "FRONT",
			},
		},
		Cameras: models.Cameras{
			RoadFacing: models.RoadFacing{
				Status:                  "ONLINE",
				ForwardCollisionWarning: false,
			},
			DriverFacing: models.DriverFacing{
				Status:      "ONLINE",
				DriverState: "ALERT",
			},
		},
	}

	alerts := engine.EvaluateFrame(st, &impactFrame)

	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.Type != "COLLISION" {
		t.Errorf("expected COLLISION alert, got %s", alert.Type)
	}

	if alert.Severity != "CRITICAL" {
		t.Errorf("expected CRITICAL severity, got %s", alert.Severity)
	}
}

func TestDrowsyDriverDebounce(t *testing.T) {
	engine := NewEngine()
	st := store.New()

	baseTime := time.Date(2026, 8, 11, 4, 0, 0, 0, time.UTC)

	// First drowsy frame
	frame1 := models.TelemetryFrame{
		FrameId:    "frm_drowsy_1",
		TripId:     "TRIP-DROWSY",
		Seq:        10,
		RecordedAt: baseTime.Format(time.RFC3339),
		SpeedKph:   50.0,
		Cameras: models.Cameras{
			DriverFacing: models.DriverFacing{
				Status:       "ONLINE",
				DriverState:  "DROWSY",
				EyeClosureMs: 1800,
			},
		},
	}
	alerts1 := engine.EvaluateFrame(st, &frame1)
	if len(alerts1) != 1 || alerts1[0].Type != "DROWSY_DRIVER" {
		t.Fatalf("expected 1 DROWSY_DRIVER alert, got %d", len(alerts1))
	}

	// Immediate subsequent drowsy frame 2 seconds later (must be debounced)
	frame2 := models.TelemetryFrame{
		FrameId:    "frm_drowsy_2",
		TripId:     "TRIP-DROWSY",
		Seq:        11,
		RecordedAt: baseTime.Add(2 * time.Second).Format(time.RFC3339),
		SpeedKph:   50.0,
		Cameras: models.Cameras{
			DriverFacing: models.DriverFacing{
				Status:       "ONLINE",
				DriverState:  "DROWSY",
				EyeClosureMs: 1900,
			},
		},
	}
	alerts2 := engine.EvaluateFrame(st, &frame2)
	if len(alerts2) != 0 {
		t.Fatalf("expected 0 alerts due to 60s debounce, got %d", len(alerts2))
	}
}

func TestIdleNoMotionInCongestionDoesNotTrigger(t *testing.T) {
	engine := NewEngine()
	st := store.New()

	baseTime := time.Date(2026, 8, 11, 4, 0, 0, 0, time.UTC)

	// Stationary vehicle in traffic congestion for 200 seconds
	for i := 0; i <= 100; i++ {
		frame := models.TelemetryFrame{
			FrameId:    "frm_cong",
			TripId:     "TRIP-CONGESTION",
			Seq:        i,
			RecordedAt: baseTime.Add(time.Duration(i*2) * time.Second).Format(time.RFC3339),
			SpeedKph:   0.0,
			EngineOn:   true,
			Context: models.Context{
				TrafficCongestion: true, // in traffic!
			},
		}
		alerts := engine.EvaluateFrame(st, &frame)
		if len(alerts) > 0 {
			t.Fatalf("expected 0 alerts for stationary in traffic congestion, got %d on seq %d", len(alerts), i)
		}
	}
}
