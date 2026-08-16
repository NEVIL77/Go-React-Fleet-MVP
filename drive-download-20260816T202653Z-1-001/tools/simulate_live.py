#!/usr/bin/env python3
"""
Dashcam / telematics device simulator.

Replays the recorded telemetry in `data/telemetry/*.ndjson` against YOUR ingest
endpoint, exactly the way the real in-vehicle devices would: batched HTTP POSTs,
paced by the frames' own timestamps.

Stdlib only - no pip install required.

Examples
--------
# 1. Load every trip as fast as the API accepts it - the usual choice
python3 tools/simulate_live.py --all --speed max \
    --url http://localhost:8080/api/v1/ingest/telemetry --token nxc_demo_token_9f2b41c7

# 2. Just the completed trips, or just the two in-progress ones
python3 tools/simulate_live.py --all-historical --speed max --url <...>
python3 tools/simulate_live.py --live --speed max --no-rebase --url <...>

# 3. Replay one trip at ten-times speed, paced by its own timestamps - handy for
#    watching your detector work, but nothing in the assignment requires it
python3 tools/simulate_live.py --trip TRIP-1009 --speed 10 --url <...>

# 4. See what would be sent, without a server
python3 tools/simulate_live.py --trip TRIP-1009 --speed 60 --dry-run
"""

import argparse
import json
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timedelta, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TELE = ROOT / "data" / "telemetry"
TRIPS = ROOT / "data" / "trips.json"


def load_frames(trip_id):
    path = TELE / f"{trip_id}.ndjson"
    if not path.exists():
        sys.exit(f"no telemetry file for {trip_id} ({path})")
    with path.open() as fh:
        return [json.loads(line) for line in fh if line.strip()]


def parse_ts(s):
    return datetime.fromisoformat(s.replace("Z", "+00:00"))


def rebase(frames, start_at):
    """Shift every timestamp so the first frame lands on `start_at`."""
    t0 = parse_ts(frames[0]["recorded_at"])
    delta = start_at.replace(microsecond=0) - t0
    for f in frames:
        f["recorded_at"] = (parse_ts(f["recorded_at"]) + delta).isoformat().replace("+00:00", "Z")
    return frames


def post(url, token, batch, retries=3):
    body = json.dumps({"frames": batch}).encode()
    req = urllib.request.Request(url, data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    for attempt in range(retries):
        try:
            with urllib.request.urlopen(req, timeout=15) as resp:
                return resp.status, resp.read(200).decode(errors="replace")
        except urllib.error.HTTPError as e:
            return e.code, e.read(300).decode(errors="replace")
        except Exception as e:              # noqa: BLE001 - device retries on any transport error
            if attempt == retries - 1:
                return 0, str(e)
            time.sleep(1.5 * (attempt + 1))
    return 0, "unreachable"


def replay(trip_id, args):
    frames = load_frames(trip_id)
    if args.rebase:
        frames = rebase(frames, datetime.now(timezone.utc))

    print(f"[{trip_id}] {len(frames)} frames  vehicle={frames[0]['vehicle_id']} "
          f"driver={frames[0]['driver_id']}  "
          f"speed={'unpaced' if args.speed == 'max' else args.speed + 'x'}  batch={args.batch}")

    t_wall = time.monotonic()
    t_first = parse_ts(frames[0]["recorded_at"])
    sent = 0

    for i in range(0, len(frames), args.batch):
        batch = frames[i:i + args.batch]

        if args.speed != "max":
            offset = (parse_ts(batch[-1]["recorded_at"]) - t_first).total_seconds() / float(args.speed)
            sleep_for = offset - (time.monotonic() - t_wall)
            if sleep_for > 0:
                time.sleep(sleep_for)

        if args.dry_run:
            f = batch[-1]
            road = f["cameras"]["road_facing"]
            drv = f["cameras"]["driver_facing"]
            crash = f["sensors"]["crash_sensor"]

            flags = []
            if road["forward_collision_warning"]:
                flags.append("FCW")
            if road["status"] != "ONLINE":
                flags.append(f"cam={road['status']}")
            if crash["impact_detected"]:
                flags.append(f"IMPACT {crash['peak_g']}g {crash['impact_direction']}")
            suffix = ("  " + "  ".join(flags)) if flags else ""

            print(f"  seq={f['seq']:5d} {f['recorded_at']} "
                  f"{f['location']['lat']:.5f},{f['location']['lon']:.5f} "
                  f"{f['speed_kph']:5.1f} kph  driver={drv['driver_state']}{suffix}")
        else:
            status, snippet = post(args.url, args.token, batch)
            if status not in (200, 201, 202, 204):
                print(f"  !! seq={batch[0]['seq']}-{batch[-1]['seq']} HTTP {status} {snippet}")
                if args.stop_on_error:
                    sys.exit(1)
        sent += len(batch)
        if args.verbose and sent % (args.batch * 20) == 0:
            print(f"  .. {sent}/{len(frames)} frames")

    print(f"[{trip_id}] done - {sent} frames in {time.monotonic() - t_wall:.1f}s wall time")


def main():
    p = argparse.ArgumentParser(description="Replay dashcam telemetry into your ingest API")
    p.add_argument("--url", default="http://localhost:8080/api/v1/ingest/telemetry")
    p.add_argument("--token", default="nxc_demo_token_9f2b41c7")
    p.add_argument("--trip", action="append", default=[], help="trip id (repeatable)")
    p.add_argument("--live", action="store_true", help="replay the trips flagged live_replay")
    p.add_argument("--all-historical", action="store_true", help="replay every completed trip")
    p.add_argument("--all", action="store_true", help="every trip with telemetry (the usual choice)")
    p.add_argument("--speed", default="1", help="'1' real time, '10' ten-times, 'max' no pacing")
    p.add_argument("--batch", type=int, default=5, help="frames per POST")
    p.add_argument("--rebase", action="store_true", default=None,
                   help="shift timestamps so the trip starts now (default: on for --live)")
    p.add_argument("--no-rebase", dest="rebase", action="store_false")
    p.add_argument("--loop", action="store_true", help="restart when the trip finishes")
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--stop-on-error", action="store_true")
    p.add_argument("--verbose", action="store_true")
    args = p.parse_args()

    if args.speed != "max":
        float(args.speed)

    trips = json.loads(TRIPS.read_text())
    selected = list(args.trip)
    if args.live:
        selected += [t["trip_id"] for t in trips if t.get("live_replay")]
    if args.all_historical:
        selected += [t["trip_id"] for t in trips
                     if t["telemetry_file"] and not t.get("live_replay")]
    if args.all:
        selected += [t["trip_id"] for t in trips if t["telemetry_file"]]
    if not selected:
        sys.exit("nothing selected: pass --all / --trip / --live / --all-historical")

    if args.rebase is None:
        args.rebase = bool(args.live)

    seen = []
    for t in selected:
        if t not in seen:
            seen.append(t)

    while True:
        for trip_id in seen:
            replay(trip_id, args)
        if not args.loop:
            break
        print("-- looping --")


if __name__ == "__main__":
    main()
