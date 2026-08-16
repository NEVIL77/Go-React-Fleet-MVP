export function projectPoint(
  lat: number,
  lon: number,
  bbox: [number, number, number, number],
  width: number,
  height: number,
  padding = 24,
) {
  const [minLat, minLon, maxLat, maxLon] = bbox;
  const x = padding + ((lon - minLon) / (maxLon - minLon || 1)) * (width - padding * 2);
  const y = padding + ((maxLat - lat) / (maxLat - minLat || 1)) * (height - padding * 2);
  return { x, y };
}

export const severityColor: Record<string, string> = {
  CRITICAL: "#dc2626",
  HIGH: "#ea580c",
  MEDIUM: "#ca8a04",
  LOW: "#6b7280",
};

export const severityRadius: Record<string, number> = {
  CRITICAL: 10,
  HIGH: 8,
  MEDIUM: 7,
  LOW: 6,
};
