# Data dictionary

Reference for the seed files under `data/`. All IDs are strings; treat them as opaque.

---

## `organization.json`

| Field | Type | Notes |
|---|---|---|
| `org_id` | string | `org_nxc` — the single tenant in this dataset |
| `name` | string | |
| `timezone` | string | IANA name; display in local time, store UTC |
| `base_city` | string | |
| `depots[]` | array | `depot_id`, `name`, `lat`, `lon` |
| `api_token` | string | the bearer token your API must require |

`org_id` appears throughout the data because real devices report it. It is descriptive only —
this exercise has one tenant and no org scoping.

## `drivers.json` — 10 rows

| Field | Type | Notes |
|---|---|---|
| `driver_id` | string | `DRV-001` … |
| `name`, `phone`, `licence_no`, `licence_expiry` | | |
| `status` | enum | `ACTIVE` / `ON_LEAVE` |
| `safety_score` | int | 0–100, seeded; recomputing it from alerts is a bonus |
| `joined_on` | date | |

## `vehicles.json` — 14 rows

| Field | Type | Notes |
|---|---|---|
| `vehicle_id` | string | `VEH-001` … |
| `registration_no` | string | |
| `make`, `model`, `vehicle_type`, `payload_capacity_kg`, `fuel_type`, `year` | | `vehicle_type` ∈ `MINI_TRUCK`/`PICKUP`/`LCV`/`TRUCK` |
| `odometer_km` | float | reading at the **start** of this dataset |
| `status` | enum | `ACTIVE` (12) / `IN_MAINTENANCE` (VEH-013) / `DECOMMISSIONED` (VEH-014) |
| `home_depot_id` | string | |
| `device` | object | `device_id`, `model`, `firmware`, `sim_iccid`, `last_heartbeat_at`, `sensors` |
| `device.sensors` | object | capability declaration for the three sources: `road_facing_camera` (resolution, FOV, ADAS features), `driver_facing_camera` (resolution, IR, DMS features), `crash_sensor` (model, axes, `trigger_threshold_g`, `sample_hz`) |
| `insurance_expiry`, `fitness_expiry` | date | good material for an expiry widget |

Note `VEH-013` is `IN_MAINTENANCE` yet has a `SCHEDULED` trip (`TRIP-1011`) against it.
That is deliberate. Surfacing it is a bonus; crashing on it is not.

## `trips.json` — 12 rows

| Field | Type | Notes |
|---|---|---|
| `trip_id` | string | `TRIP-1001` … `TRIP-1012` |
| `vehicle_id`, `driver_id` | string | |
| `route_name` | string | human label |
| `origin`, `destination` | object | `lat`, `lon` |
| `scheduled_start_at`, `started_at`, `ended_at` | timestamp | `null` where not applicable |
| `status` | enum | see API contract |
| `distance_km`, `duration_s`, `frame_count` | | derived from telemetry — recompute rather than trusting these |
| `telemetry_file` | string | path to the NDJSON, `null` for the scheduled trip |
| `cargo` | object | `manifest_id`, `weight_kg`, `type` |
| `live_replay` | bool | `true` for the two `IN_PROGRESS` trips (TRIP-1009, TRIP-1010). Load them with the simulator's `--live` flag; the name is historical, nothing is streamed |

## `telemetry/TRIP-XXXX.ndjson`

One frame per line, 2 s apart. Full schema in [`EVENT_SPEC.md`](EVENT_SPEC.md) §2.

| Trip | Frames | Duration | Character |
|---|---|---|---|
| TRIP-1001 | 1140 | 38 min | pothole strike at 30 min — crash sensor fires, nothing else does |
| TRIP-1002 | 1260 | 42 min | drowsy episode + engine-on idling |
| TRIP-1003 |  572 | 19 min | ends in a real collision (5.2 g, FRONT, 43 kph delta-v) |
| TRIP-1004 | 1380 | 46 min | two forward-collision near misses |
| TRIP-1005 | 1200 | 40 min | two long idles + a pothole strike |
| TRIP-1006 | 1320 | 44 min | clean night run |
| TRIP-1007 | 1080 | 36 min | two drowsy episodes, night |
| TRIP-1008 | 1200 | 40 min | near miss + drowsy |
| TRIP-1009 | 452 | 15 min | in progress — real collision ~14 min in |
| TRIP-1010 | 960 | 32 min | in progress — drowsy, then idling |
| TRIP-1012 | 1020 | 34 min | idling at the depot |

11,584 frames total, ~2 s apart — **6.4 hours of driving**, each frame ~1 kB of JSON,
~9 MB on disk. Shortest trip 15 min, longest 46 min.

Traps worth knowing about, all intentional:

- **Four frames have `crash_sensor.impact_detected: true` and only two are collisions.**
  The two pothole strikes report 4.2 g `VERTICAL` with ~0 kph delta-v and no camera
  corroboration; the two real collisions report 5.2 g `FRONT` with 43 kph delta-v and a
  forward-collision warning in the preceding 4 s. A `peak_g >= 3.5` check alone gets this
  wrong, and gets it wrong in the direction that pages people at 3 a.m.
- Many frames have `speed_kph < 2` with `traffic_congestion: true` — legitimate stops in
  traffic. They must not raise `IDLE_NO_MOTION`.
- Drowsy episodes span 15–30 consecutive frames. Without the 60 s debounce you will produce
  ~20 alerts instead of 5.
- `TRIP-1003` and `TRIP-1009` stop dead after the impact — speed is 0 for the last ~60 s
  with the engine on. Ordering your rules correctly keeps that from becoming a bogus
  `IDLE_NO_MOTION`.
- GPS quality varies (`hdop`, `satellites`); no frame is bad enough to discard, but a
  quality filter is a reasonable thing to build.

## `snapshots/evt_*_road.png`, `snapshots/evt_*_driver.png`

320×180 PNG stills, **24 files across 17 events**. `_road` is the road-facing view;
`_driver` is the IR cabin view (eyes drawn closed on drowsiness events). Border colour
encodes alert type.

How many stills each event carries:

| Alert | road | driver |
|---|---|---|
| `COLLISION`, `UNVERIFIED_IMPACT`, `NEAR_MISS` | ✓ | ✓ |
| `DROWSY_DRIVER` | — | ✓ |
| `IDLE_NO_MOTION` | ✓ | — |

Referenced by `cameras.<which>.snapshot_url` on trigger frames. Serve them from your own
API — the frontend must not read `data/` directly.
