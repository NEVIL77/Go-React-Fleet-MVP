# Architectural Notes & Decisions

### 1. How did you decide an impact was real?
We distinguish real collisions from surface-level anomalies (such as potholes or speed breakers) by requiring corroborating evidence alongside the crash sensor's high-frequency impact reading. Specifically, while any event with `impact_detected: true` and `peak_g >= 3.5` registers as an impact, it is only classified as a `COLLISION` (Severity: `CRITICAL`) if corroborated by either:
1. An ADAS forward-collision warning (`forward_collision_warning: true`) on the road-facing camera within the current frame or either of the two preceding frames ($t-4\text{s}$ to $t$), OR
2. A sudden deceleration of $\ge 10.0\text{ kph}$ from the immediate preceding frame.
If a $\ge 3.5\text{g}$ trigger lacks both camera corroboration and significant wheel-speed loss (e.g. 4.2g vertical spike over a pothole at constant speed), it is classified as `UNVERIFIED_IMPACT` (Severity: `LOW`) and routed to an offline review queue without paging on-call safety operators.

### 2. Where does your detection state live, and what breaks if a trip's frames get handled by two workers?
Detection state (recent frame history for $\Delta v$/FCW checks, last drowsy alert timestamp/sequence for the 60-second debounce, and the continuous idle duration counter) currently lives in-memory within an engine struct keyed by `trip_id` protected by a mutex. If frames for a single trip were distributed arbitrarily across two concurrent workers without sticky routing, the state would fragment:
- Worker B evaluating frame $N$ would lack frame $N-1$ and $N-2$ processed on Worker A, missing speed drops and pre-impact FCW warnings (turning collisions into false unverified impacts).
- The 60-second drowsiness debounce and 180-second idle run counters would race or reset independently, resulting in duplicate alerts or missed threshold crossings.
To prevent this in a multi-worker setup, stream messages must be partitioned by `trip_id` (or vehicle ID) so that a given trip's frames are guaranteed to be processed in-order by the same worker instance, or state transitions must be centralized in an atomic store like Redis with Lua scripts/hashes.

### 3. What would you swap the in-process bus for in production, and what changes when the API runs as three instances?
In production, the in-process Go channel would be replaced by **Redis Streams** (using consumer group `detector` and stream `fleet:telemetry`) or Apache Kafka. Redis Streams provide message durability, consumer groups with automatic message leasing, pending-entry recovery (`XPENDING` / `XCLAIM`), and at-least-once delivery semantics so that crashes during processing do not drop telemetry. When scaling the API to three instances behind a load balancer, each API instance simply executes `XADD fleet:telemetry * ...` and returns `202 Accepted` immediately. If low-latency live dashboard updates were later needed, detector workers would publish confirmed alerts to a Redis Pub/Sub topic or a second stream (`fleet:alerts`), and API instances would subscribe to Redis to push events downstream to connected browser WebSockets or SSE channels.

### 4. What did you cut, and what would you do with another four hours?
To stay within the four-hour constraint and prioritize sensor fusion correctness and clean boundary separation, we cut:
- Redis infrastructure / Docker dependencies in favor of an in-process buffered channel bus interface and thread-safe in-memory store.
- Interactive map tile providers (Mapbox/Google Maps/Leaflet) in favor of lightweight projected SVG polyline rendering with color-coded severity markers.
- Acknowledgement mutations, GPS quality filtering (`hdop`/`satellites`), camera obstruction degradation handling, and secondary live update transports.

With another four hours, we would:
1. Implement Redis Streams integration with `go-redis`, partition-key consumer assignment per trip, and stream crash recovery tests.
2. Add full SQLite / Postgres persistence using `sqlc` or GORM with the production schema and migration scripts.
3. Integrate Leaflet / MapLibre GL for full interactive tile zoom and pan on real Hyderabad road network basemaps with cluster markers.
4. Implement alert acknowledgement workflows (`POST /api/v1/alerts/{id}/acknowledge`) and operator audit logging.
