import type { RouteAlert, RoutePoint } from "../types";
import { projectPoint, severityColor, severityRadius } from "../utils/map";

interface Props {
  points: RoutePoint[];
  alerts: RouteAlert[];
  bbox: [number, number, number, number];
  selectedAlertId?: string;
  onSelectAlert: (alert: RouteAlert) => void;
}

const W = 800;
const H = 500;

export function RouteMap({ points, alerts, bbox, selectedAlertId, onSelectAlert }: Props) {
  const polyline = points
    .map((p) => {
      const { x, y } = projectPoint(p.lat, p.lon, bbox, W, H);
      return `${x},${y}`;
    })
    .join(" ");

  return (
    <svg viewBox={`0 0 ${W} ${H}`} className="route-map">
      <rect width={W} height={H} fill="#f8fafc" />
      <polyline points={polyline} fill="none" stroke="#2563eb" strokeWidth={2} />
      {alerts.map((alert) => {
        const { x, y } = projectPoint(alert.lat, alert.lon, bbox, W, H);
        const selected = alert.alert_id === selectedAlertId;
        return (
          <circle
            key={alert.alert_id}
            cx={x}
            cy={y}
            r={severityRadius[alert.severity] ?? 6}
            fill={severityColor[alert.severity] ?? "#666"}
            stroke={selected ? "#111" : "#fff"}
            strokeWidth={selected ? 3 : 2}
            style={{ cursor: "pointer" }}
            onClick={() => onSelectAlert(alert)}
          />
        );
      })}
    </svg>
  );
}
