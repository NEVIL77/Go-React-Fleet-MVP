import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { alertCount, fetchTrips, formatDuration } from "../api/trips";
import type { TripListItem } from "../types";

export function TripsPage() {
  const [trips, setTrips] = useState<TripListItem[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchTrips()
      .then((res) => setTrips(res.items))
      .catch((e) => setError(String(e)));
  }, []);

  if (error) return <p className="error">{error}</p>;

  return (
    <div>
      <h1>Trips</h1>
      <table className="trips-table">
        <thead>
          <tr>
            <th>Route</th>
            <th>Status</th>
            <th>Duration</th>
            <th>Events</th>
          </tr>
        </thead>
        <tbody>
          {trips.map((trip) => (
            <tr key={trip.trip_id}>
              <td>
                <Link to={`/trips/${trip.trip_id}`}>{trip.route_name}</Link>
              </td>
              <td>{trip.status}</td>
              <td>{formatDuration(trip.duration_s)}</td>
              <td>{alertCount(trip)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
