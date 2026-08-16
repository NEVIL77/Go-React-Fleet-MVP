const API_BASE = import.meta.env.VITE_API_BASE ?? "http://localhost:8080/api/v1";
const TOKEN = import.meta.env.VITE_API_TOKEN ?? "nxc_demo_token_9f2b41c7";
const USE_MOCK = import.meta.env.VITE_USE_MOCK !== "false";

export const isMockMode = () => USE_MOCK;

async function apiFetch<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { Authorization: `Bearer ${TOKEN}` },
  });
  if (!res.ok) throw new Error(`API ${path}: ${res.status}`);
  return res.json() as Promise<T>;
}

async function mockFetch<T>(path: string): Promise<T> {
  const res = await fetch(path);
  if (!res.ok) throw new Error(`Mock ${path}: ${res.status}`);
  return res.json() as Promise<T>;
}

export async function getJson<T>(apiPath: string, mockPath: string): Promise<T> {
  return USE_MOCK ? mockFetch<T>(mockPath) : apiFetch<T>(apiPath);
}

export function snapshotUrl(eventId: string, camera: "road_facing" | "driver_facing"): string {
  if (USE_MOCK) return `/mock/snapshots/${eventId}_${camera}.png`;
  return `${API_BASE}/snapshots/${eventId}/${camera}`;
}
