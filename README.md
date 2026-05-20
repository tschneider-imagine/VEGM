# VEGM

A lightweight Virtual EGM runtime for MVP-1 and beyond.

## What this repo is right now

This repository is the **active VEGM source tree** seeded from the latest local runtime work and then adapted into a coherent GitHub handoff base.

It already includes:
- a runnable `cmd/vegm` entrypoint
- message-pack loading, validation, summaries, and initial overlay support
- a wire listener with HTTP or HTTPS depending on trust mode
- registration checks and baseline session handling
- outbound session open / heartbeat support
- an automatic outbound scheduler
- JSONL event logging plus payload capture
- JSON bundle export from the repo seed logger
- repo-native docs, schemas, example packs, and smoke tests

It is still a **repo seed**, not full parity with every richer local artifact produced earlier in the chat.

## Current capabilities

### Security / transport
Supported trust modes:
- `plaintext_lab`
- `tls_server_only`
- `strict_mtls`
- `mtls_no_revocation`
- `accept_all_lab`

### Session / control behavior
Current repo seed supports:
- host registration checks
- `commsOnLine` / `commsOnLineAck`
- `keepAlive` / `keepAliveAck`
- `getDescriptor` / `descriptorList`
- `setKeepAlive` / `setKeepAliveAck` through the extended pack
- `getCabinetStatus` / `cabinetStatus` through the extended pack
- `commsClosing` / `commsClosingAck`
- outbound session open, outbound heartbeat, and generic outbound send
- automatic outbound scheduler with attempt counters and degradation state

### Logging / observability
The repo seed logger currently provides:
- JSONL event logging
- payload capture for raw/rendered XML when enabled
- event filtering by category, level, text, message type, host ID, session ID, and time window
- JSON bundle export with:
  - filtered events
  - optional payload file list
  - state snapshot
  - config snapshot
  - pack summary

## What is not done yet

Not yet completed in this repo seed:
- full parity with every local runtime artifact built outside GitHub
- SQLite-backed indexing
- deeper XML extraction beyond the current local-name oriented approach
- fuller runtime/server test coverage
- vendor-certified behavior packs
- polished live wire restart for trust-mode changes

## Quick start

Run the starter plaintext config:

```bash
go run ./cmd/vegm -config ./example.vegm.json
```

Run the dynamic local-dev config with automatic port assignment:

```bash
go run ./cmd/vegm -config ./example.dynamic.vegm.json
```

Run the outbound-focused config:

```bash
go run ./cmd/vegm -config ./example.outbound.vegm.json
```

Use the strict mTLS config once cert files exist under `./certs/`:

```bash
go run ./cmd/vegm -config ./example.strict-mtls.vegm.json
```

## Useful control endpoints

- `GET /healthz`
- `GET /control/state`
- `GET /control/logs?message_type=keepAlive&limit=50`
- `POST /control/export`
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

## Example control calls

Log query:

```bash
curl "http://127.0.0.1:19001/control/logs?message_type=keepAlive&limit=20"
```

Bundle export:

```bash
curl -X POST "http://127.0.0.1:19001/control/export"
```

Outbound session open:

```bash
curl -X POST http://127.0.0.1:19001/control/outbound/session/open \
  -H "Content-Type: application/json" \
  -d '{"host_id":"HOST-001","session_id":"S-100","target_url":"http://127.0.0.1:18080/host"}'
```

Outbound heartbeat:

```bash
curl -X POST http://127.0.0.1:19001/control/outbound/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"host_id":"HOST-001","session_id":"S-100"}'
```

Scheduler start:

```bash
curl -X POST http://127.0.0.1:19001/control/outbound/scheduler/start
```

Generic outbound send:

```bash
curl -X POST http://127.0.0.1:19001/control/outbound/send \
  -H "Content-Type: application/json" \
  -d '{"message_type":"getDescriptor","session_id":"S-200"}'
```

## Example assets included in the repo

Configs:
- `example.vegm.json`
- `example.dynamic.vegm.json`
- `example.outbound.vegm.json`
- `example.strict-mtls.vegm.json`

Packs:
- `example.pack.json`
- `example.overlay.json`
- `example.extended.pack.json`
- `example.vendorquirk.pack.json`

Docs:
- `docs/project_build_log.md`
- `docs/VEGM_MVP1_protocol_matrix.md`
- `docs/SQLite_blocker_and_unblock_plan.md`

Schemas:
- `schemas/message-pack.schema.json`
- `schemas/message-overlay.schema.json`

## Repository highlights

- `cmd/vegm` — runnable VEGM server entrypoint
- `runtime/` — wire plane, control plane, scheduler, logger, outbound behavior
- `pack/` — message-pack loading, validation, overlays, summaries
- `schemas/` — message-pack and overlay schemas
- `docs/` — build log, MVP-1 matrix, SQLite unblock plan

## SQLite note

SQLite is still blocked here by **dependency acquisition**, not by VEGM design.

The repo already includes the unblock plan in:
- `docs/SQLite_blocker_and_unblock_plan.md`

Recommended next external step:
- vendor the chosen SQLite driver into this repo on a connected machine
- then wire SQLite in as a search/index layer on top of the current JSONL + payload capture foundation
