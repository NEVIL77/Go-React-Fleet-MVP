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
  impact_direction?: string;
  snapshots: { road_facing: string | null; driver_facing: string | null };
  corroborated_by?: string[];
  message: string;
}

export interface RoutePoint {
  seq: number;
  lat: number;
  lon: number;
  speed_kph: number;
  recorded_at: string;
}

export interface RouteAlert {
  alert_id: string;
  type: AlertType;
  severity: Severity;
  seq: number;
  lat: number;
  lon: number;
}

export interface TripRoute {
  trip_id: string;
  point_count: number;
  bbox: [number, number, number, number];
  points: RoutePoint[];
  alerts: RouteAlert[];
}

export interface TripListItem {
  trip_id: string;
  route_name: string;
  status: string;
  started_at: string;
  ended_at: string | null;
  duration_s: number;
  alert_counts: Partial<Record<Severity, number>>;
}

export interface TripsResponse {
  items: TripListItem[];
  page: number;
  page_size: number;
  total: number;
}
