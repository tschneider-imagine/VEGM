# VEGM

A lightweight Virtual EGM runtime for MVP-1 and beyond.

What it does now:
- Loads a validated VEGM message pack
- Optionally applies validated overlays at startup
- Starts a wire listener on HTTP or HTTPS depending on trust mode
- Supports `plaintext_lab`, `tls_server_only`, `strict_mtls`, `mtls_no_revocation`, and `accept_all_lab`
- Implements baseline registration/session bring-up and keepalive handling
- Adds extended optional protocol coverage for `getDescriptor`, `setKeepAlive`, `getCabinetStatus`, and `commsClosing`
- Exposes a localhost-friendly control plane for state inspection, live log query, run-bundle export, overlay apply, live host add/remove, outbound session/heartbeat initiation, an automatic outbound scheduler, and live trust-mode reload/switch via controlled wire restart
- Writes JSONL event logs and raw/rendered XML payload captures
- Exports zip bundles with filtered events, raw log copy, state snapshot, config snapshot, pack summary, and payload files

What it does not do yet:
- Full G2S class coverage
- SQLite indexing (blocked here by missing embedded SQLite driver/tooling in this environment)
- Deep XPath evaluation beyond local-name extraction
- Scheduler jitter/backoff tuning beyond simple interval/retry settings

Run:
```bash
go run ./cmd/vegm -config ./example.vegm.json
```

Useful control endpoints:
- `GET /healthz`
- `GET /control/state`
- `GET /control/logs?message_type=keepAlive&limit=50`
- `POST /control/export?payloads=true`
- `POST /control/overlay`
- `POST /control/hosts/add`
- `POST /control/hosts/remove`
- `POST /control/security/reload`
- `POST /control/security/mode`
- `POST /control/outbound/session/open`
- `POST /control/outbound/heartbeat`
- `POST /control/outbound/send`
- `GET /control/pack/summary`
- `GET /control/pack/operations`
- `POST /control/outbound/scheduler/start`
- `POST /control/outbound/scheduler/stop`

Example log query:
```bash
curl "http://127.0.0.1:19001/control/logs?message_type=keepAlive&limit=20"
```

Example bundle export:
```bash
curl -X POST "http://127.0.0.1:19001/control/export?payloads=true"
```

Example outbound session open:
```bash
curl -X POST http://127.0.0.1:19001/control/outbound/session/open \
  -H "Content-Type: application/json" \
  -d '{"host_id":"HOST-001","session_id":"S-100","target_url":"http://127.0.0.1:18080/host"}'
```

Example outbound heartbeat:
```bash
curl -X POST http://127.0.0.1:19001/control/outbound/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"host_id":"HOST-001","session_id":"S-100"}'
```

Example scheduler start:
```bash
curl -X POST http://127.0.0.1:19001/control/outbound/scheduler/start
```

Example scheduler stop:
```bash
curl -X POST http://127.0.0.1:19001/control/outbound/scheduler/stop
```

Example generic outbound send:
```bash
curl -X POST http://127.0.0.1:19001/control/outbound/send \
  -H "Content-Type: application/json" \
  -d '{"message_type":"getDescriptor","descriptor_class":"cabinet","session_id":"S-200"}'
```

Included example packs:
- `example.extended.pack.json`
- `example.vendorquirk.pack.json`

Repository highlights:
- `cmd/vegm` — runnable VEGM server
- `runtime/` — wire plane, control plane, scheduler, security reload, outbound behavior
- `pack/` — message-pack loading, validation, overlays, summaries
- `schemas/` — message-pack and overlay schemas
- `docs/` — build log, MVP-1 protocol matrix, SQLite unblock notes
