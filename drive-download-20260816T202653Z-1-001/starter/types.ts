// Reference types for the telemetry frame and the alert you will produce.
// Generated from the dataset, so the field names match exactly.
// Adapt freely or delete - this is a convenience, not a requirement.


export interface TelemetryFrame {
  frame_id: string;
  org_id: string;
  trip_id: string;
  vehicle_id: string;
  driver_id: string;
  device_id: string;
  seq: number;
  recorded_at: string;
  location: {
    lat: number;
    lon: number;
    heading_deg: number;
    hdop: number;
    satellites: number;
  };
  speed_kph: number;
  engine_on: boolean;
  rpm: number;
  odometer_km: number;
  fuel_level_pct: number;
  coolant_temp_c: number;
  sensors: {
    accelerometer_g: {
      x: number;
      y: number;
      z: number;
      magnitude: number;
    };
    gyroscope_dps: {
      x: number;
      y: number;
      z: number;
    };
    crash_sensor: {
      status: string;
      impact_detected: boolean;
      peak_g: number;
      delta_v_kph: number;
      impact_direction: string;
      rollover_detected: boolean;
      trigger_source: string;
    };
  };
  cameras: {
    road_facing: {
      status: string;
      forward_collision_warning: boolean;
      lane_departure_warning: boolean;
      headway_s: number | null;
      pedestrian_detected: boolean;
      detected_speed_limit_kph: number;
      obstruction_score: number;
      snapshot_url: string | null;
    };
    driver_facing: {
      status: string;
      driver_state: string;
      eye_closure_ms: number;
      yawn_count_1m: number;
      head_pitch_deg: number;
      phone_use_detected: boolean;
      smoking_detected: boolean;
      seatbelt_fastened: boolean;
      snapshot_url: string | null;
    };
  };
  context: {
    traffic_congestion: boolean;
    road_speed_limit_kph: number;
    geofence: string;
    ambient: string;
  };
}

export type AlertType =
  | "COLLISION"
  | "UNVERIFIED_IMPACT"
  | "DROWSY_DRIVER"
  | "IDLE_NO_MOTION";

export type Severity = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

export interface Alert {
  alert_id: string;
  event_id: string;
  type: AlertType;
  severity: Severity;
  trip_id: string;
  vehicle_id: string;
  driver_id: string;
  detected_at: string;
  trigger_frame_id: string;
  trigger_seq: number;
  location: { lat: number; lon: number };
  speed_kph: number;
  g_force: number;
  snapshots: { road_facing: string | null; driver_facing: string | null };
  corroborated_by?: string[];
  message: string;
}
