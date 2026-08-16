# Take-home exercise — Fleet Safety Dashboard (Go + React)

**Plan for about four hours.** Submit within 3 days.
**Stack:** Go for the backend, React (TypeScript preferred) for the frontend.

This is deliberately small. We've cut the infrastructure, the screen count, and the CRUD so
that your four hours go into the one thing we actually want to read: how you decide which
sensor events are real, and where you put the boundary between accepting data and acting on it.

**If you hit four hours and aren't done, stop.** Write up where you got to in `NOTES.md` and
send it. We mean that — see §6.

---

## 1. The scenario

Nexcart Logistics runs light commercial vehicles around Hyderabad. Each carries a telematics
unit reporting a **telemetry frame every 2 seconds** from three independent sources:

- a **road-facing camera** running ADAS — forward-collision warning, headway, lane departure,
- a **driver-facing camera** running driver monitoring — eye closure, yawns, head pose,
- a **crash sensor** — a 6-axis impact detector reporting peak g, delta-v, impact direction.

Those three disagree. The crash sensor fires on potholes and speed breakers as readily as on
real collisions, and it's also the only source that can detect an impact at all — so you can't
just trust it and you can't just ignore it. **Deciding what's real before you alert anyone is
the core of this exercise.**

## 2. What to build

### Backend (Go)

1. **Ingest** — `POST /api/v1/ingest/telemetry`, accepting batches from the supplied
   simulator. Idempotent on `frame_id`: the device retries, so frames arrive twice. It must
   **not** run detection inline — see point 2.

2. **An event-driven seam** — the ingest handler publishes and returns; a consumer picks the
   frames up, runs the rules, and records alerts. Read
   [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) §1, §5 and §7 for the shape and the reasoning.

   **Implement the bus over a Go channel.** Define it as an interface, ship the in-process
   version, and describe the Redis Streams version in `NOTES.md`. We're scoring the seam and
   the reasoning, not the infrastructure — do not spend your four hours on Redis.

3. **Detection** — rules **3.1, 3.2, 3.4 and 3.5** from
   [`docs/EVENT_SPEC.md`](docs/EVENT_SPEC.md). Rule 3.3 (`NEAR_MISS`) is out of scope and this
   dataset has no events for it.

4. **Three read endpoints:**
   - `GET /api/v1/trips` — for the trips list
   - `GET /api/v1/trips/{id}/route` — the polyline plus that trip's alerts, for the detail screen
   - `GET /api/v1/alerts` — every event across the fleet. No screen uses this one; **we call it
     to check your detection output against our answer key**, so please include it.

5. **Serve the dashcam stills.** Serving them straight off disk from `data/snapshots/` is
   fine here — we're not testing file handling.

6. **Auth** — compare against the single token in `data/organization.json`, `401` otherwise.

**Storage: in-memory is fine.** No database, no migrations. Describe the schema you'd use in
production in `NOTES.md` instead — we read that as carefully as we'd read the DDL.

**No Docker.** Two documented commands to start the backend and the frontend.

### Frontend (React) — two screens, and one of them is trivial

1. **Trips** — a plain list: route name, status, duration, and a count of events on that trip.
   It exists so you can click into a trip. No filters, no sorting, no pagination.

2. **Trip detail** — this is the screen, and the whole point of the exercise is visible here.
   Three things, in this order of importance:

   **a. The route the trip actually took**, drawn from its telemetry. A trip is 450–1,300
   points; draw all of them or downsample, your call.

   **b. Every event on that trip, plotted at the place it happened.** Not a list beside the
   map — markers *on* the route, at the event's own `lat`/`lon`, colour-coded by severity.
   `GET /trips/{id}/route` hands you the polyline and that trip's alerts together, so this is
   one request. A `CRITICAL` collision and a `LOW` unverified impact must be
   distinguishable at a glance, without clicking.

   **c. Clicking an event** reveals its dashcam still(s) and the metadata behind it — peak g,
   impact direction, speed, and what corroborated it. Two stills side by side where both
   exist.

   The test: a reviewer opens a trip, sees immediately *where* on the route something
   happened and *how serious* it was, and can click through to decide whether an
   `UNVERIFIED_IMPACT` was a real hit or a pothole. If your screen does that, it's done.

   A list of events beside the map, in addition, is fine and probably useful — but the
   markers are the requirement, not the list.

For the map: [React Leaflet](https://react-leaflet.js.org/) with OpenStreetMap needs no API
key and gives you markers for free. **A hand-rolled SVG is equally acceptable** — project
lat/lon onto a viewBox from the route's bounding box, draw a `<polyline>`, and place a
`<circle>` per event. That's often faster than fighting map tiles, and it scores the same. We
care that the route and the events are legible, not which library drew them. Don't lose 40
minutes to map setup.

### Out of scope — do not build these

A vehicles page. A separate alerts queue. Acknowledge actions. Filters, sorting, search,
pagination. Live updates of any kind — no WebSocket, no SSE, no polling. Login screens, user
management, multi-tenancy. Docker, Redis, migrations, object storage. Mobile layouts. Visual
polish. The `NEAR_MISS` rule.

Pages fetch over REST when they load. That's the whole interaction model.

---

## 3. The data

```
data/
  organization.json          the API token
  drivers.json               10 drivers
  vehicles.json              14 vehicles
  trips.json                 6 trips (5,584 frames, ~3 h of driving)
  telemetry/TRIP-XXXX.ndjson one frame per line, 2 s apart
  snapshots/evt_*.png        road-facing and driver-facing stills
starter/
  types.ts                   TypeScript interfaces for the frame and alert
  frame.go                   the same as Go structs
tools/
  simulate_live.py           replays telemetry into YOUR ingest endpoint (stdlib Python)
docs/
  EVENT_SPEC.md              §2 frame schema, §3 the detection rules — the important one
  ARCHITECTURE.md            the seam we ask for (§1, §5, §7)
  API_CONTRACT.md            reference shapes — follow loosely, not a conformance test
  DATA_DICTIONARY.md         field-by-field reference
```

`starter/` exists so you don't spend half an hour transcribing a JSON schema into structs.
Adapt it however you like, or delete it — it's a convenience, not a requirement.

Load everything through your ingest API with one command:

```bash
python3 tools/simulate_live.py --all --speed max \
  --url http://localhost:8080/api/v1/ingest/telemetry --token nxc_demo_token_9f2b41c7
```

---

## 4. What must be true when you're done

The dataset contains **10 events**:

| Alert type          | Count | Severity |
|---------------------|-------|----------|
| `COLLISION`         | 2     | CRITICAL |
| `UNVERIFIED_IMPACT` | 2     | LOW      |
| `DROWSY_DRIVER`     | 2     | HIGH     |
| `IDLE_NO_MOTION`    | 4     | MEDIUM   |

There are **four crash-sensor triggers** in this data and only **two** are collisions. If you
report 4 CRITICAL alerts, you've paged the safety team twice for a bad road. This is the first
thing we check.

Re-running the load must not duplicate frames or alerts.

---

## 5. Deliverables

1. A Git repo (or zip) with `backend/` and `frontend/`, and a README with the two commands.
2. **`NOTES.md`** — four short answers, a paragraph each. Not an essay:
   - How did you decide an impact was real?
   - Where does your detection state live (previous frame, debounce window, idle run), and
     what breaks if a trip's frames get handled by two workers?
   - What would you swap the in-process bus for in production, and what changes when the API
     runs as three instances?
   - What did you cut, and what would you do with another four hours?
3. **One unit test:** an uncorroborated crash-sensor trigger must not produce a `COLLISION`.
4. Screenshots of the trip detail screen — route with the events marked on it — and of an
   event's dashcam stills.

## 6. If you run out of time

Build in this order and stop wherever you land:

1. Ingest + the publish/consume seam
2. Detection rules — especially getting 2 collisions and not 4
3. `GET /trips/{id}/route` and `GET /alerts`
4. Trip detail: the route drawn, with the events marked on it
5. The stills and metadata on click
6. The trips list

A submission that stops after step 4 with a clear `NOTES.md` scores better than one that
reaches step 6 with the fusion logic wrong. We are not counting features.

## 7. How we grade

| Area | Weight |
|---|---|
| Detection correctness — especially 2 collisions, not 4 | 35% |
| Go: the publish/consume seam, package layout, error handling, no data races | 25% |
| `NOTES.md` — reasoning and honesty about trade-offs | 20% |
| React: the route and its events rendered legibly, severity distinguishable at a glance, component boundaries | 15% |
| Craft — the one test, README, commit history | 5% |

## 8. Rules

- Any libraries you like — list them in `NOTES.md`.
- AI assistants are fine. You'll be asked to explain and modify this code on a 30-minute call,
  so don't submit anything you can't defend.
- Don't modify anything under `data/`.
- Questions are welcome — ask rather than guess silently.
