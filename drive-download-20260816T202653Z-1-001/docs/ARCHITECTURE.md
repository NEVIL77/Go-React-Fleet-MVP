# Required architecture — event-driven pipeline

This is the one structural constraint we place on your backend. Everything else (packages,
storage, libraries) is your call.

---

## 1. The shape

```
  simulator
      │
      │ POST /ingest/telemetry
      ▼
┌─────────────┐   XADD    ┌──────────────┐ XREADGROUP ┌──────────────┐
│  ingest API │──────────▶│ fleet:telem  │───────────▶│   detector   │
│  validate + │   stream  │   (Redis     │  consumer  │    worker    │
│  persist    │           │   Stream)    │   group    │  runs rules  │
└─────────────┘           └──────────────┘            └──────┬───────┘
      │ 202 immediately                                      │
      ▼                                                      ▼
   (client)                                           ┌──────────────┐
                                                      │    store     │
                                                      │ frames+alerts│
                                                      └──────────────┘
                                                             ▲
                                                             │ GET /alerts
                                                        (dashboard, on load)
```

**The ingest handler must not run detection inline.** It validates, persists, publishes to the
stream, and returns `202`. Detection happens in a consumer. That separation is the point of
this section.

The dashboard reads alerts over REST when a page loads. There is no live push in this
exercise — see §4.

## 2. Why we ask for it

Two things fall out of it, and they are what we're actually testing:

1. **Ingest latency is decoupled from detection cost.** Devices retry on timeout; a slow
   detector must not turn into a retry storm on the vehicles.
2. **Detection scales independently.** At 5,000 vehicles the detector is the bottleneck, not
   the HTTP handler. A consumer group lets you add workers without touching the API, and
   without two workers doing the same frame twice.

## 3. Redis specifics

Redis is in `docker-compose.yml`. Use these names so we can inspect them:

| Purpose | Type | Name |
|---|---|---|
| Telemetry frames awaiting detection | **Stream** | `fleet:telemetry` |
| Detector consumer group | Group | `detector` |

Telemetry needs durability and replay: if the detector crashes mid-batch those frames must
still be processed. Streams give you consumer groups, a pending-entries list, and `XACK`.

Redis Pub/Sub would be the wrong choice here — it drops anything published while no consumer
is attached, so restarting your detector would silently lose frames. Being able to say why is
worth more to us than the code that follows from it.

### Delivery semantics

Redis Streams give you **at-least-once**. Your detector will see the same frame twice — after
a crash, after a rebalance, after a redelivery from the pending list. Combined with
device-level retries, that means:

- the frame store must be idempotent on `frame_id`,
- **alert creation must be idempotent too.** Deduplicate on something stable such as
  `(trip_id, type, trigger_seq)`. Re-running the bulk load must not double your alert count.

`XACK` only once the work is durable, not on receipt.

## 4. No live push in this exercise

Out of scope: WebSocket, SSE, socket.io, and polling loops. The dashboard fetches alerts and
routes over REST when the user opens a page. Don't build a transport.

We do want the thinking, in `NOTES.md`, in a short paragraph:

> If we later wanted alerts to appear on an open dashboard within two seconds of ingest, where
> would that hook into the pipeline above, and what would you use? What breaks first when the
> API runs as three instances behind a load balancer?

(The answer we're listening for involves a second Redis channel and the fact that an
in-process list of connections doesn't survive horizontal scaling. We want to know you can see
it, not that you built it.)

## 5. Where detection state lives

The rules are stateful across frames — the previous frame's speed, a 60-second drowsiness
debounce, a 180-second idle run. Once detection runs in a worker, that state has to live
somewhere that survives the frame arriving on a different worker.

We are not prescribing a solution. In-worker state keyed by `trip_id` is fine **if** you
partition so a trip always lands on the same consumer; Redis-held state is fine if you accept
the round trip. What we want in `NOTES.md` is which you chose and what breaks under the other.

This is the most interesting design question in the exercise. Spend a paragraph on it.

## 6. What we check

- `POST /ingest/telemetry` returns before detection has run for those frames.
- `fleet:telemetry` exists as a Stream with a consumer group (we run `XINFO GROUPS`).
- **Crash recovery:** stop the detector, ingest a batch, restart it — the alerts still appear,
  and nothing from the gap is lost.
- **Two detectors, one group:** run a second detector against the same Redis and re-run the
  bulk load. Both should share the work, and the alert count must not change.
- Re-running the bulk load produces no duplicate frames or alerts.

## 7. If you run short on time

Ship the seam even if you can't ship the infrastructure. Define the publisher and consumer as
interfaces, implement them over a Go channel, and describe the Redis version in `NOTES.md` as
prose. That scores far better than an inline detector with a comment saying "would use a queue
in production" — it shows you know where the boundary goes, which is the thing we're reading.
