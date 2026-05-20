# VEGM MVP-1 Protocol Matrix

## Scope

This repo-native matrix defines the **MVP-1 protocol and behavior surface** for the current VEGM seed in this repository.

It is intentionally focused on:
- host registration
- session bring-up shell
- heartbeat / keepalive
- mTLS-first operation with explicit bypass modes
- observable live and historical logging
- configurable network identity at the runtime-config level

This is not the mute-controller implementation spec.

## Locked assumptions

### Traffic direction
VEGM must support both:
- inbound host-to-VEGM control/session requests
- outbound VEGM-to-host session bring-up and heartbeat requests

### Security modes
The runtime recognizes these trust modes:
- `strict_mtls`
- `mtls_no_revocation`
- `tls_server_only`
- `accept_all_lab`
- `plaintext_lab`

### First feature target
The first useful protocol target remains:
- host registration checks
- `commsOnLine` / `commsOnLineAck`
- `keepAlive` / `keepAliveAck`
- baseline descriptor and session-closing support where available in the pack

## Runtime surfaces in this repo seed

### Wire plane
Current seed supports a configurable listener path from the active message pack.

Expected behavior:
- inbound POST on the active listener path
- XML parsing based on local-name extraction
- operation lookup by pack-defined message type
- response generation from the first active response template for the operation
- optional artificial response delay from the pack timer block

### Control plane
Current seed includes endpoints for:
- `/healthz`
- `/control/state`
- `/control/logs`
- `/control/export`
- `/control/overlay`
- `/control/hosts/add`
- `/control/hosts/remove`
- `/control/security/mode`
- `/control/security/reload`
- `/control/pack/summary`
- `/control/pack/operations`
- `/control/outbound/session/open`
- `/control/outbound/heartbeat`
- `/control/outbound/send`
- `/control/outbound/scheduler/start`
- `/control/outbound/scheduler/stop`

## Registration matrix

### Registration modes
The current seed pack/runtime recognizes:
- `open`
- `strict_match`
- `deny_all`
- `simulate_vendor_quirk`

### Minimum expected behaviors
1. Accept known host in `strict_match`
2. Reject unknown host in `strict_match`
3. Accept any host in `open`
4. Surface the decision in logs
5. Allow live add/remove of allowed hosts through the control plane

### Acceptance criteria
- known host reaches normal operation handling
- unknown host receives rejection
- runtime state records the last host ID
- decision is visible in JSONL logs

## Session matrix

### Minimum inbound operations
The repo-native starter pack ships with:
- `commsOnLine`
- `keepAlive`
- `getDescriptor`
- `commsClosing`

### Expected state effects
- `commsOnLine` moves session state to `online`
- `keepAlive` moves heartbeat state to `healthy`
- `commsClosing` moves session state to `closed`

### Acceptance criteria
- inbound `commsOnLine` returns `commsOnLineAck`
- inbound `keepAlive` returns `keepAliveAck`
- inbound `getDescriptor` returns `descriptorList`
- inbound `commsClosing` returns `commsClosingAck`

## Outbound matrix

### Manual outbound controls
Current seed supports:
- session open
- heartbeat send
- generic outbound send

### Automatic outbound controls
Current seed includes a scheduler with:
- start/stop endpoints
- session-open on idle/offline
- recurring heartbeat attempts
- failure counting
- degradation signaling after repeated failures

### Acceptance criteria
- outbound `commsOnLine` can mark session online on valid ACK
- outbound `keepAlive` can mark heartbeat healthy on valid ACK
- scheduler increments attempt counters
- scheduler logs failures and degradation transitions

## Logging matrix

### Current repo seed
The repo seed includes:
- JSONL event logging
- payload capture for raw/rendered XML paths that are enabled in config
- event query support
- a stub export hook that preserves the interface for a fuller export layer

### Must-have log fields
- instance ID
- category
- message type where known
- host ID where known
- session ID where known
- trust mode changes
- scheduler failures

## Network identity matrix

### Current repo seed expectations
The runtime config must expose:
- listen host
- listen port
- control bind address
- outbound default target URL

This seed does not yet reconfigure the host operating system network stack.
It models and exposes network identity at the application layer.

## Pack and overlay matrix

### Pack layer
This repo contains:
- pack loading
- pack validation
- pack summary generation
- initial overlay support

### Overlay support in the current seed
Overlay handling is intentionally limited and compile-oriented.
The starter overlay demonstrates:
- changing artificial response delay
- changing registration mode

## Non-goals in this seed

Not yet completed in the repo seed:
- full parity with every earlier local runtime artifact
- SQLite-backed indexing
- deep XPath or full schema-driven XML extraction
- full vendor-certified quirk packs
- full automatic wire restart on trust-mode switch

## Next build targets after this seed
1. strengthen tests around the seeded runtime
2. improve export implementation behind the existing interface
3. increase pack/overlay parity with the richer local artifacts
4. add SQLite once a vendored driver is available
