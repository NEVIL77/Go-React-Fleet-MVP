import type { Alert } from "../types";
import { snapshotUrl } from "../api/client";
import { severityColor } from "../utils/map";

interface Props {
  alert: Alert;
}

function Snapshot({ label, src }: { label: string; src: string | null }) {
  if (!src) return <div className="snapshot missing">No {label} still</div>;
  return (
    <figure className="snapshot">
      <img src={src} alt={label} onError={(e) => (e.currentTarget.style.display = "none")} />
      <figcaption>{label}</figcaption>
    </figure>
  );
}

export function EventPanel({ alert }: Props) {
  const road = alert.snapshots.road_facing
    ? snapshotUrl(alert.event_id, "road_facing")
    : null;
  const driver = alert.snapshots.driver_facing
    ? snapshotUrl(alert.event_id, "driver_facing")
    : null;

  return (
    <div className="event-panel">
      <div className="event-header">
        <span className="badge" style={{ background: severityColor[alert.severity] }}>
          {alert.severity}
        </span>
        <strong>{alert.type}</strong>
      </div>
      <p>{alert.message}</p>
      <dl className="meta">
        <dt>Speed</dt>
        <dd>{alert.speed_kph.toFixed(1)} kph</dd>
        <dt>Peak G</dt>
        <dd>{alert.g_force.toFixed(2)}</dd>
        {alert.impact_direction && (
          <>
            <dt>Direction</dt>
            <dd>{alert.impact_direction}</dd>
          </>
        )}
        {alert.corroborated_by && alert.corroborated_by.length > 0 && (
          <>
            <dt>Corroborated</dt>
            <dd>{alert.corroborated_by.join(", ")}</dd>
          </>
        )}
      </dl>
      <div className="snapshots">
        <Snapshot label="Road" src={road} />
        <Snapshot label="Driver" src={driver} />
      </div>
    </div>
  );
}
