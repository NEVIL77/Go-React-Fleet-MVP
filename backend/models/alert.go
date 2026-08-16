// Package models holds JSON types for alerts produced by detection.
package models

type AlertLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Snapshots struct {
	RoadFacing   *string `json:"road_facing"`
	DriverFacing *string `json:"driver_facing"`
}

type Alert struct {
	AlertID         string        `json:"alert_id"`
	EventID         string        `json:"event_id"`
	Type            string        `json:"type"`
	Severity        string        `json:"severity"`
	TripID          string        `json:"trip_id"`
	VehicleID       string        `json:"vehicle_id"`
	DriverID        string        `json:"driver_id"`
	DetectedAt      string        `json:"detected_at"`
	TriggerFrameID  string        `json:"trigger_frame_id"`
	TriggerSeq      int           `json:"trigger_seq"`
	Location        AlertLocation `json:"location"`
	SpeedKph        float64       `json:"speed_kph"`
	GForce          float64       `json:"g_force"`
	ImpactDirection *string       `json:"impact_direction,omitempty"`
	Snapshots       Snapshots     `json:"snapshots"`
	CorroboratedBy  []string      `json:"corroborated_by,omitempty"`
	Message         string        `json:"message"`
	Acknowledged    bool          `json:"acknowledged"`
	AcknowledgedBy  *string       `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *string       `json:"acknowledged_at,omitempty"`
}
