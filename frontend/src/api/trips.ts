import { getJson } from "./client";
import type { Alert, TripRoute, TripsResponse } from "../types";

export function fetchTrips() {
  return getJson<TripsResponse>("/trips", "/mock/trips.json");
}

export function fetchTripRoute(tripId: string) {
  return getJson<TripRoute>(`/trips/${tripId}/route`, `/mock/routes/${tripId}.json`);
}

export async function fetchAlertDetail(alertId: string): Promise<Alert | null> {
  const res = await getJson<{ items: Alert[] }>("/alerts", "/mock/alerts.json");
  return res.items.find((a) => a.alert_id === alertId) ?? null;
}

export function alertCount(trip: TripsResponse["items"][0]): number {
  return Object.values(trip.alert_counts).reduce((sum, n) => sum + (n ?? 0), 0);
}

export function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}m ${s}s`;
}
