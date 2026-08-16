// Package models holds JSON types for telemetry frames.
package models

type Location struct {
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	HeadingDeg float64 `json:"heading_deg"`
	Hdop       float64 `json:"hdop"`
	Satellites int     `json:"satellites"`
}

type AccelerometerG struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Z         float64 `json:"z"`
	Magnitude float64 `json:"magnitude"`
}

type GyroscopeDps struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type CrashSensor struct {
	Status          string  `json:"status"`
	ImpactDetected  bool    `json:"impact_detected"`
	PeakG           float64 `json:"peak_g"`
	DeltaVKph       float64 `json:"delta_v_kph"`
	ImpactDirection string  `json:"impact_direction"`
	RolloverDetected bool   `json:"rollover_detected"`
	TriggerSource   string  `json:"trigger_source"`
}

type Sensors struct {
	AccelerometerG AccelerometerG `json:"accelerometer_g"`
	GyroscopeDps   GyroscopeDps   `json:"gyroscope_dps"`
	CrashSensor    CrashSensor    `json:"crash_sensor"`
}

type RoadFacing struct {
	Status                  string   `json:"status"`
	ForwardCollisionWarning bool     `json:"forward_collision_warning"`
	LaneDepartureWarning    bool     `json:"lane_departure_warning"`
	HeadwayS                *float64 `json:"headway_s"`
	PedestrianDetected      bool     `json:"pedestrian_detected"`
	DetectedSpeedLimitKph   int      `json:"detected_speed_limit_kph"`
	ObstructionScore        float64  `json:"obstruction_score"`
	SnapshotUrl             *string  `json:"snapshot_url"`
}

type DriverFacing struct {
	Status           string  `json:"status"`
	DriverState      string  `json:"driver_state"`
	EyeClosureMs     int     `json:"eye_closure_ms"`
	YawnCount1m      int     `json:"yawn_count_1m"`
	HeadPitchDeg     float64 `json:"head_pitch_deg"`
	PhoneUseDetected bool    `json:"phone_use_detected"`
	SmokingDetected  bool    `json:"smoking_detected"`
	SeatbeltFastened bool    `json:"seatbelt_fastened"`
	SnapshotUrl      *string `json:"snapshot_url"`
}

type Cameras struct {
	RoadFacing   RoadFacing   `json:"road_facing"`
	DriverFacing DriverFacing `json:"driver_facing"`
}

type Context struct {
	TrafficCongestion bool   `json:"traffic_congestion"`
	RoadSpeedLimitKph int    `json:"road_speed_limit_kph"`
	Geofence          string `json:"geofence"`
	Ambient           string `json:"ambient"`
}

type TelemetryFrame struct {
	FrameId      string   `json:"frame_id"`
	OrgId        string   `json:"org_id"`
	TripId       string   `json:"trip_id"`
	VehicleId    string   `json:"vehicle_id"`
	DriverId     string   `json:"driver_id"`
	DeviceId     string   `json:"device_id"`
	Seq          int      `json:"seq"`
	RecordedAt   string   `json:"recorded_at"`
	Location     Location `json:"location"`
	SpeedKph     float64  `json:"speed_kph"`
	EngineOn     bool     `json:"engine_on"`
	Rpm          int      `json:"rpm"`
	OdometerKm   float64  `json:"odometer_km"`
	FuelLevelPct float64  `json:"fuel_level_pct"`
	CoolantTempC float64  `json:"coolant_temp_c"`
	Sensors      Sensors  `json:"sensors"`
	Cameras      Cameras  `json:"cameras"`
	Context      Context  `json:"context"`
}
