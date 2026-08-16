# Telemetry & event specification

Everything in this file is normative. The detection rules are graded against an answer key.

---

## 1. The three signal sources

Every vehicle carries one telematics unit exposing three independent sources. They can and
do disagree — reconciling them is the core of this exercise.

| Source | What it reports | Fails by |
|---|---|---|
| **Road-facing camera** | ADAS: forward-collision warning, headway, lane departure, pedestrians, speed-limit signs | Being blocked, dirty, or blinded at night |
| **Driver-facing camera** | DMS: eye closure, yawns, head pose, phone use, seatbelt | Occlusion, sunglasses, driver out of frame |
| **Crash sensor** | 6-axis impact detection: peak g, delta-v, impact direction, rollover | **False positives** — potholes, speed breakers, kerb strikes, door slams |

The crash sensor is the only source that can detect an impact the cameras miss, and it is
also the noisiest. A production system never pages a safety team on the crash sensor alone.

## 2. Telemetry frame

One JSON object per line in `data/telemetry/*.ndjson`. Devices emit one frame every
**2 seconds** while the ignition is on. The simulator posts them in batches of 5.

```json
{
  "frame_id": "frm_9c1a4e7b2d5f8031",
  "org_id": "org_nxc",
  "trip_id": "TRIP-1003",
  "vehicle_id": "VEH-003",
  "driver_id": "DRV-003",
  "device_id": "DCAM-3F7A21B9",
  "seq": 540,
  "recorded_at": "2026-08-11T04:36:00Z",
  "location": {
    "lat": 17.55021, "lon": 78.47934,
    "heading_deg": 12.4, "hdop": 0.94, "satellites": 11
  },
  "speed_kph": 12.5,
  "engine_on": true,
  "rpm": 2210,
  "odometer_km": 84213.66,
  "fuel_level_pct": 63.4,
  "coolant_temp_c": 89.2,

  "sensors": {
    "accelerometer_g": { "x": 2.71, "y": -3.94, "z": 2.05, "magnitude": 5.2 },
    "gyroscope_dps": { "x": -1.4, "y": 0.9, "z": 6.2 },
    "crash_sensor": {
      "status": "ONLINE",
      "impact_detected": true,
      "peak_g": 5.2,
      "delta_v_kph": 43.2,
      "impact_direction": "FRONT",
      "rollover_detected": false,
      "trigger_source": "IMPACT_THRESHOLD"
    }
  },

  "cameras": {
    "road_facing": {
      "status": "ONLINE",
      "forward_collision_warning": true,
      "lane_departure_warning": false,
      "headway_s": 0.34,
      "pedestrian_detected": false,
      "detected_speed_limit_kph": 60,
      "obstruction_score": 0.03,
      "snapshot_url": "/snapshots/evt_657a1de657b4b82c_road.png"
    },
    "driver_facing": {
      "status": "ONLINE",
      "driver_state": "ALERT",
      "eye_closure_ms": 0,
      "yawn_count_1m": 0,
      "head_pitch_deg": 2.1,
      "phone_use_detected": false,
      "smoking_detected": false,
      "seatbelt_fastened": true,
      "snapshot_url": "/snapshots/evt_657a1de657b4b82c_driver.png"
    }
  },

  "context": {
    "traffic_congestion": false,
    "road_speed_limit_kph": 60,
    "geofence": "HIGHWAY",
    "ambient": "DAY"
  }
}
```

Notes that matter for implementation:

- `frame_id` is globally unique and stable — it is your **idempotency key**.
- `seq` is monotonic **within a trip**, starting at 0. Use it for ordering; do not assume
  frames arrive in order, and do not assume `recorded_at` is strictly increasing across
  retries.
- `speed_kph` is the wheel-speed reading, not derived from GPS.
- `accelerometer_g.magnitude` = ‖(x, y, z)‖, in g. At rest it sits at ~1.0 (gravity).
- `crash_sensor.peak_g` is the peak over the 400 Hz window the frame summarises — it is
  higher-fidelity than the 0.5 Hz accelerometer field, so **use `peak_g` for impact logic**.
- `crash_sensor.impact_direction` ∈ `FRONT | REAR | LEFT | RIGHT | VERTICAL | NONE`.
- `driver_state` ∈ `ALERT | DISTRACTED | DROWSY | EYES_CLOSED | ABSENT`.
- Camera `status` ∈ `ONLINE | DEGRADED | OBSTRUCTED | OFFLINE`. Both cameras are `ONLINE`
  throughout this dataset, so no rule below depends on it — but the field is real, and when a
  camera is not `ONLINE` its detection fields are stale rather than negative. Handling that
  (and the `obstruction_score`) is listed under bonus work, not required.
- `headway_s` is time-to-vehicle-ahead; `null` when stationary or nothing is ahead.
- `context.traffic_congestion` is the device's own inference from surrounding traffic —
  treat it as truth.
- `snapshot_url` is non-null **only** on frames where that camera captured a still. Each
  camera has its own, so an event may carry zero, one, or two images.

---

## 3. Alert types and detection rules

Five rules. Evaluate every ingested frame; rules use the current frame and the previous
frames of the **same trip**.

### 3.1 `COLLISION` — severity `CRITICAL`

A crash-sensor impact **corroborated by a second source**. Fire when **all** hold:

- `crash_sensor.impact_detected == true`, and
- `crash_sensor.peak_g >= 3.5`, and
- at least one corroboration:
  - `road_facing.forward_collision_warning == true` on **this frame or either of the two
    preceding frames** (the ADAS warns before the impact it could not prevent), **or**
  - `previous_frame.speed_kph - current.speed_kph >= 10.0` (a real loss of speed)

Snapshots: both cameras. Notification: immediate, highest priority.

### 3.2 `UNVERIFIED_IMPACT` — severity `LOW`

The same crash-sensor trigger with **no corroboration** — i.e. the first two conditions of
3.1 hold and the third does not. This is a pothole, a speed breaker, or a kerb strike.

It goes to a review queue. It must **not** page anyone, must **not** be styled as a crash,
and must **not** change the trip's status.

Snapshots: both cameras.

> This is the rule the exercise is built around. Implementing only `peak_g >= 3.5` yields
> **4 collisions instead of 2**, and the safety team gets woken up twice for road surface.

### 3.3 `NEAR_MISS` — severity `HIGH`

Fire when **both** hold on the same frame:

- `road_facing.forward_collision_warning == true`, and
- `previous_frame.speed_kph - current.speed_kph >= 25.0` (a ≥25 kph drop in 2 s)

**Precedence:** if a frame qualifies as `COLLISION` or `UNVERIFIED_IMPACT`, do **not** also
raise `NEAR_MISS` for it. An impact frame is one event, not two.

Snapshots: both cameras.

### 3.4 `DROWSY_DRIVER` — severity `HIGH`

Fire when **all** hold:

- `driver_facing.driver_state` ∈ {`DROWSY`, `EYES_CLOSED`}, and
- `driver_facing.eye_closure_ms >= 1500`, and
- `speed_kph > 20.0` (drowsiness only matters while moving)

**Debounce:** at most one `DROWSY_DRIVER` alert per trip per **60 seconds** (30 frames).
A sustained drowsy episode is one alert, not thirty.

Snapshots: driver-facing only.

### 3.5 `IDLE_NO_MOTION` — severity `MEDIUM`

Fire when a run of consecutive frames in the same trip all satisfy:

- `engine_on == true`, and
- `speed_kph < 2.0`, and
- `context.traffic_congestion == false`

...and the run has lasted **≥ 180 seconds** (`seq` difference from the first frame of the
run ≥ 90). Fire **once** per run, at the frame where the run crosses 180 s. Any frame that
breaks a condition resets the run.

A vehicle stationary in reported congestion must **not** fire it — that negative case is
graded.

Snapshots: road-facing only.

---

## 4. Alert object

Shape your API response like this (field names are checked loosely; the data is not):

```json
{
  "alert_id": "alt_01J8X...",
  "event_id": "evt_657a1de657b4b82c",
  "type": "COLLISION",
  "severity": "CRITICAL",
  "org_id": "org_nxc",
  "trip_id": "TRIP-1003",
  "vehicle_id": "VEH-003",
  "driver_id": "DRV-003",
  "detected_at": "2026-08-11T04:36:00Z",
  "ingested_at": "2026-08-13T09:12:44Z",
  "trigger_frame_id": "frm_9c1a4e7b2d5f8031",
  "trigger_seq": 540,
  "location": { "lat": 17.55021, "lon": 78.47934 },
  "speed_kph": 12.5,
  "g_force": 5.2,
  "snapshots": {
    "road_facing": "/api/v1/snapshots/evt_657a1de657b4b82c/road_facing",
    "driver_facing": "/api/v1/snapshots/evt_657a1de657b4b82c/driver_facing"
  },
  "corroborated_by": ["CRASH_SENSOR", "ROAD_CAMERA_FCW", "SPEED_DROP"],
  "message": "Frontal impact 5.2g, delta-v 43.2 kph, corroborated by road camera FCW",
  "acknowledged": false,
  "acknowledged_by": null,
  "acknowledged_at": null
}
```

`detected_at` is the telemetry timestamp; `ingested_at` is when your system saw it — keep
both. `corroborated_by` is what justifies the severity; an operator disputing an alert will
ask for exactly this, so make it explicit rather than implied.

`snapshots` values may be `null`. Model it as a set of images per event, not one image.

---

## 5. Expected results

Across the whole dataset the rules above produce **17 alerts**:

| Trip | Alerts |
|---|---|
| TRIP-1001 | `UNVERIFIED_IMPACT` |
| TRIP-1002 | `DROWSY_DRIVER`, `IDLE_NO_MOTION` |
| TRIP-1003 | `COLLISION` |
| TRIP-1004 | `NEAR_MISS` ×2 |
| TRIP-1005 | `IDLE_NO_MOTION` ×2, `UNVERIFIED_IMPACT` |
| TRIP-1006 | — |
| TRIP-1007 | `DROWSY_DRIVER` ×2 |
| TRIP-1008 | `NEAR_MISS`, `DROWSY_DRIVER` |
| TRIP-1009 (in progress) | `COLLISION` |
| TRIP-1010 (in progress) | `DROWSY_DRIVER`, `IDLE_NO_MOTION` |
| TRIP-1011 | — (scheduled, no telemetry) |
| TRIP-1012 | `IDLE_NO_MOTION` |

Totals: `COLLISION` 2 · `UNVERIFIED_IMPACT` 2 · `NEAR_MISS` 3 · `DROWSY_DRIVER` 5 ·
`IDLE_NO_MOTION` 5.

There are **4 crash-sensor triggers** in the dataset. Exactly 2 are collisions.

Re-ingesting the same trip must not change these numbers.

---

## 6. How alerts must be surfaced

When an alert is raised:

1. Persist it.
2. Materialise every snapshot it references so they are servable from your API.
3. Make it available on `GET /api/v1/alerts`, newest first, filterable by severity.

Routing by severity is a UI concern, and it is graded:

- `CRITICAL` — impossible to miss in the alerts list.
- `HIGH` — clearly flagged.
- `MEDIUM` / `LOW` — present but quiet. **`UNVERIFIED_IMPACT` must not be presented as a
  crash.** A reviewer should be able to open it, look at the two stills, and decide whether it
  was a real hit or a pothole.

There is no live push in this exercise — the dashboard loads alerts over REST. See
[ARCHITECTURE.md](ARCHITECTURE.md) §4.
