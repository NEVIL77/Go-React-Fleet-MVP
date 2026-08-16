import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { fetchAlertDetail, fetchTripRoute } from "../api/trips";
import { EventPanel } from "../components/EventPanel";
import { RouteMap } from "../components/RouteMap";
import type { Alert, RouteAlert, TripRoute } from "../types";
import { severityColor } from "../utils/map";

export function TripDetailPage() {
  const { tripId } = useParams<{ tripId: string }>();
  const [route, setRoute] = useState<TripRoute | null>(null);
  const [selected, setSelected] = useState<RouteAlert | null>(null);
  const [detail, setDetail] = useState<Alert | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!tripId) return;
    fetchTripRoute(tripId)
      .then(setRoute)
      .catch((e) => setError(String(e)));
  }, [tripId]);

  useEffect(() => {
    if (!selected) {
      setDetail(null);
      return;
    }
    fetchAlertDetail(selected.alert_id)
      .then(setDetail)
      .catch(() => setDetail(null));
  }, [selected]);

  if (error) return <p className="error">{error}</p>;
  if (!route) return <p>Loading…</p>;

  return (
    <div>
      <p>
        <Link to="/">← Trips</Link>
      </p>
      <h1>{route.trip_id}</h1>
      <div className="detail-layout">
        <RouteMap
          points={route.points}
          alerts={route.alerts}
          bbox={route.bbox}
          selectedAlertId={selected?.alert_id}
          onSelectAlert={setSelected}
        />
        <aside>
          <h2>Events</h2>
          <ul className="event-list">
            {route.alerts.map((a) => (
              <li key={a.alert_id}>
                <button
                  type="button"
                  className={selected?.alert_id === a.alert_id ? "active" : ""}
                  onClick={() => setSelected(a)}
                >
                  <span className="dot" style={{ background: severityColor[a.severity] }} />
                  {a.type} — {a.severity}
                </button>
              </li>
            ))}
          </ul>
          {detail && <EventPanel alert={detail} />}
        </aside>
      </div>
    </div>
  );
}
