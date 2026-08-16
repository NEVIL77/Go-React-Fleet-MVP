# API contract

Base path `/api/v1`. JSON in, JSON out. All timestamps RFC 3339 UTC (`...Z`).

**Auth:** every request carries `Authorization: Bearer <token>`. The token is in
`data/organization.json` (`nxc_demo_token_9f2b41c7`). Compare it and reject anything else with
`401`. That's the whole requirement — there is no multi-tenancy in this exercise, and `org_id`
in the data is descriptive, not a scoping key. Unknown resource ids return `404`.

**Errors:** non-2xx responses use one shape:

```json
{ "error": { "code": "INVALID_FRAME", "message": "seq must be >= 0", "details": {} } }
```

You may add endpoints. The ones below are the ones we exercise.

---

## Ingest

### `POST /api/v1/ingest/telemetry`

The only endpoint the device simulator calls.

```json
{ "frames": [ { /* telemetry frame, see EVENT_SPEC.md */ } ] }
```

- Batch size is 5 in practice; accept 1–500.
- **Idempotent on `frame_id`.** A replayed batch must not create duplicate frames or
  duplicate alerts.
- Partial success is allowed — report it rather than failing the whole batch.
- Respond `202` **before detection has run**. The handler validates, persists, publishes to
  `fleet:telemetry`, and returns. See [ARCHITECTURE.md](ARCHITECTURE.md).

```json
{ "accepted": 5, "duplicates": 0, "rejected": 0, "queued": 5 }
```

`alerts_raised` is no longer meaningful in the response — detection hasn't happened yet when
you reply. Report what you enqueued instead.

---

## Vehicles

### `GET /api/v1/vehicles`

Query: `status`, `type`, `depot_id`, `q` (free text over registration/make/model),
`page`/`page_size` or `cursor`/`limit`, `sort`.

Each item should carry enough for the list view without an N+1 from the frontend:

```json
{
  "vehicle_id": "VEH-003",
  "registration_no": "TS9XG4417",
  "make": "Ashok Leyland", "model": "Dost+", "vehicle_type": "LCV",
  "status": "ACTIVE",
  "current_driver": { "driver_id": "DRV-003", "name": "Anitha Reddy" },
  "current_trip_id": null,
  "device": {
    "device_id": "DCAM-3F7A21B9", "status": "OFFLINE", "last_heartbeat_at": "...",
    "road_camera_status": "ONLINE", "driver_camera_status": "ONLINE", "crash_sensor_status": "ONLINE"
  },
  "last_known_location": { "lat": 17.55, "lon": 78.47, "recorded_at": "..." },
  "odometer_km": 84213.66,
  "open_alert_count": 1
}
```

Derive `device.status` from the last heartbeat (e.g. `ONLINE` if seen in the last 120 s), and
the three per-sensor statuses from the last frame. A vehicle can be reporting while one of its
three sensors is degraded, so the fleet view should distinguish "device offline" from "one
sensor unhealthy" — every sensor is `ONLINE` in this dataset, so this is about the model, not
about handling data we gave you.

### `GET /api/v1/vehicles/{vehicle_id}`

Detail, plus recent trips and recent alerts.

---

## Trips

### `GET /api/v1/trips`

Query: `vehicle_id`, `driver_id`, `status`, `from`, `to`, `has_alerts=true`, paging, sort.

```json
{
  "items": [{
    "trip_id": "TRIP-1003",
    "vehicle": { "vehicle_id": "VEH-003", "registration_no": "TS9XG4417" },
    "driver":  { "driver_id": "DRV-003", "name": "Anitha Reddy" },
    "route_name": "Secunderabad - Medchal",
    "status": "COMPLETED_WITH_INCIDENT",
    "started_at": "...", "ended_at": "...",
    "distance_km": 18.4, "duration_s": 1144,
    "frame_count": 572,
    "alert_counts": { "CRITICAL": 1, "HIGH": 0, "MEDIUM": 0 }
  }],
  "page": 1, "page_size": 20, "total": 12
}
```

`status` ∈ `SCHEDULED | IN_PROGRESS | COMPLETED | COMPLETED_WITH_INCIDENT | CANCELLED`.

### `GET /api/v1/trips/{trip_id}`

Summary + aggregates: max/avg speed, harsh-braking count, idle time, alert list.

### `GET /api/v1/trips/{trip_id}/route`

The polyline for the map. Query: `from_seq`, `to_seq`, `resolution` (`full` | `simplified`).

```json
{
  "trip_id": "TRIP-1003",
  "point_count": 572,
  "bbox": [17.4399, 78.4790, 17.6280, 78.4983],
  "points": [
    { "seq": 0, "lat": 17.43990, "lon": 78.49830, "speed_kph": 0.0, "recorded_at": "..." }
  ],
  "alerts": [ { "alert_id": "...", "type": "COLLISION", "seq": 540, "lat": 17.55021, "lon": 78.47934 } ]
}
```

572 points is fine to send raw; 50,000 is not. Show us you thought about it —
downsampling, `resolution`, or paging by `seq` all count.

### `GET /api/v1/trips/{trip_id}/frames`

Raw frames with `from_seq`/`limit`. Optional — useful if your trip detail screen wants the
full telemetry behind a point on the map.

---

## Alerts

### `GET /api/v1/alerts`

Query: `severity`, `type`, `trip_id`, `vehicle_id`, `driver_id`, `acknowledged`,
`from`, `to`, paging. Default sort: `detected_at` desc.

```json
{ "items": [ /* alert objects, see EVENT_SPEC.md §4 */ ],
  "unacknowledged_count": 17, "page": 1, "page_size": 20, "total": 17 }
```

`severity` filtering matters here: the safety team's default view is `CRITICAL,HIGH`, while
the review queue is `LOW` — that is how `UNVERIFIED_IMPACT` stays out of the way until
someone looks at it. This endpoint is the only way the dashboard learns about alerts, so make
it good.

### `POST /api/v1/alerts/{alert_id}/acknowledge`

```json
{ "acknowledged_by": "safety@nexcart.in", "note": "Called the driver, minor bump" }
```

Idempotent.

### `GET /api/v1/snapshots/{event_id}/{camera}`

`camera` ∈ `road_facing | driver_facing`. Returns `image/png` bytes from your own storage.
`404` if the event is unknown, or if that camera captured nothing for the event
(`DROWSY_DRIVER` has no road still; `IDLE_NO_MOTION` has no cabin still).

A single-path variant (`/snapshots/{event_id}`) returning a list of available images is
also acceptable — just be consistent with what the alert object advertises.

---

## Ops

### `GET /healthz`

```json
{ "status": "ok", "db": "ok", "redis": "ok", "uptime_s": 3812,
  "frames_ingested": 11584, "stream_pending": 0, "alerts_raised": 17 }
```
