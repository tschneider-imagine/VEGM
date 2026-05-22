# VEGM Project Plan

## 0. Project premise

VEGM is a **Virtual EGM floor** used to test and qualify the **G2S Mute Controller**.

The system under test is the external controller. VEGM must behave like a set of participating cabinets so the controller can:

- establish G2S communications,
- maintain session/liveness,
- issue mute / restore / hold / lock style commands,
- receive ACKs or failures,
- and allow the test team to verify cabinet state and history.

The local muting system concept keeps current floor systems in place while adding a local G2S muting path for mute, restore, status, and event history. The controller responsibility is trigger handling plus G2S mute/restore actions, status visibility, and event logging.

## 1. Ground rules for future coding passes

1. Do not drift away from G2S terminology.
2. Treat the VEGM as an EGM peer, not as the host/controller.
3. Keep the physical network model visible in config and UI.
4. Do not build UI-only behavior that is not backed by runtime state.
5. Every protocol feature needs a matching evidence path: raw XML, parsed message, state change, and timestamp.
6. Every coding pass must state which reference inputs it needs before implementation.

## 2. Physical topology

Target lab topology:

```text
PC1: VEGMs + Supervisor + Control Plane
  -> Ethernet
  -> Router / Switch
  -> PC2: G2S Mute Controller
```

Implications:

- VEGMs run on PC1.
- The controller runs on another PC.
- VEGM endpoints must bind to an address reachable from PC2.
- `127.0.0.1` is valid only for local testing on PC1.
- Lab/fleet config must distinguish actual bind address from G2S identity fields.

## 3. G2S stack and binding layers

The implementation must keep these layers separate:

```text
L2 Ethernet / NIC / VLAN
L3 IP / routing
L4 TCP port
L5 HTTP or HTTPS session
L6 TLS + SOAP/XML presentation
L7 G2S message classes and operations
```

Binding path for an inbound G2S service endpoint:

```text
IP:port
  -> HTTP(S)
  -> path, normally /g2s
  -> SOAP envelope
  -> G2S body
  -> operation handler
```

Binding failure buckets:

- wrong IP bind,
- wrong port,
- wrong HTTP path,
- wrong content type,
- wrong SOAP envelope,
- wrong namespace,
- wrong operation name,
- wrong response mapping.

## 4. Corrected G2S session model

The model to implement is:

```text
VEGM -> Host / Controller:
  commsOnLine
  descriptor/session startup as required
  keepAlive or liveness messages as configured

Host / Controller -> VEGM:
  mute / restore
  cabinet lock / unlock
  hold / release
  status/config/control requests
```

The repo currently contains pieces of both inbound and outbound handling. The next phase must make the G2S session engine explicit instead of leaving the flow split across UI actions, scheduler actions, and generic outbound sends.

## 5. Core G2S connection requirements

Minimum connection/startup fields:

### Endpoint / transport

- `host_endpoint_url`
  - The controller/host endpoint the VEGM will call for session startup if that flow is enabled.
- `egm_endpoint_url`
  - The local VEGM endpoint exposed for controller-to-EGM command traffic.
- `scheme`
  - `http` or `https`.
- `bind_ip`
  - Local IP/interface the VEGM listens on.
- `port`
  - TCP port for the endpoint.
- `path`
  - G2S SOAP path, normally `/g2s`.

### Identity

- `hostId`
  - G2S host/controller identity.
- `egmId`
  - G2S EGM identity.
- `sessionId`
  - Session correlation ID. Created by requester and echoed by responder.
- `sessionType`
  - Request/response orientation in G2S XML where applicable.

### Security / trust

- `trust_mode`
- `cert_file`
- `key_file`
- `ca_file`
- `server_name`
- optional revocation policy later: OCSP/SCEP settings.

### Registration / authorization

- registered host list,
- host role / permissions,
- allowed device classes or command scopes.

## 6. Reference sources to anchor the work

### Required reference packages / docs

1. IGSA G2S Message Protocol package
   - Needed for canonical message/class definitions.
2. IGSA Point-to-Point SOAP/HTTPS Transport package
   - Needed for exact transport and SOAP/HTTPS rules.
3. G2S Quick Start implementation guide
   - Needed for WSDL/XSD workflow and starter command sequence.
4. OpenG2S
   - Needed as a public implementation sanity check.
5. RadBlue CVT/RGS docs
   - Needed for practical session, commConfig, registered-host, and testing behavior.
6. Team-provided muting controller docs and slides
   - Needed to keep the Virtual EGM aligned to controller qualification, not generic simulation.

### Existing uploaded/team docs to use

- `G2S_Session_Authentication_Guide.md`
- `G2S_Certificate_Bringup_Checklist.md`
- `G2S_Lab_Runbook_Template.md`
- `G2S_XML_Format_Reference_Sheet.md`
- `G2S_Muting_System_9S.pptx`
- `Ai-o1-G2S_mute_architecture_spec.md`
- `g2s_public_bundle_full_v5.zip`

## 7. Current repo status

Current repo has useful pieces:

- `cmd/vegm`
- `cmd/vegm-supervisor`
- fleet manifest/config generation
- supervisor UI
- embedded scenario UI
- JSONL logging
- SQLite index
- message pack model
- starter operations for mute/restore/hold/lock
- live state fields for audio/hold/lock/session/heartbeat

Current repo is not complete because:

- G2S flow is not yet cleanly modeled as a spec-correct EGM session engine.
- Endpoint model still mixes implementation fields with G2S concepts.
- UI exists but is not yet a dependable test bench.
- SOAP/WSDL/XSD validation is still loose.
- Registered-host / permissions model is incomplete.
- TLS/cert bring-up is configurable but not yet fully workflow-driven.

## 8. Development path

## Pass 0 — Stabilize build and branch baseline

Goal: make sure every future pass starts from a green repo.

Tasks:

- Run `go test ./...`.
- Fix current supervisor/runtime compile issues if any.
- Confirm `go run ./cmd/vegm-supervisor -manifest ./example.fleet.json -serve` starts.
- Confirm supervisor UI loads.
- Confirm at least one VEGM can start/stop from supervisor UI.

Reference needed:

- none beyond current repo.

Exit criteria:

- `go test ./...` passes.
- Supervisor UI launches.
- Start/stop works for one instance.

## Pass 1 — Replace loose network settings with G2S endpoint model

Goal: align config names and generated settings to G2S endpoint reality.

Tasks:

- Add explicit `egm_endpoint` and `host_endpoint` sections.
- Preserve implementation-level bind fields but stop treating them as protocol concepts.
- Add endpoint parsing helpers.
- Render endpoint fields in supervisor UI.
- Keep backward compatibility with existing generated config for now.

Proposed config shape:

```json
{
  "egm_id": "EGM-001",
  "host_id": "HOST-001",
  "egm_endpoint": {
    "scheme": "http",
    "bind_ip": "0.0.0.0",
    "host": "192.168.10.10",
    "port": 18443,
    "path": "/g2s"
  },
  "host_endpoint": {
    "url": "http://192.168.10.50:443/g2s"
  }
}
```

Reference needed:

- controller endpoint configuration fields,
- actual controller host URL,
- expected EGM endpoint URL pattern,
- G2S version and transport spec version.

Exit criteria:

- supervisor shows both EGM endpoint and host endpoint per VEGM.
- generated configs contain both.
- old start/stop still works.

## Pass 2 — Binding and transport hardening

Goal: make the endpoint listener deterministic and G2S-binding correct.

Tasks:

- Validate path exactly, default `/g2s`.
- Enforce POST for G2S messages.
- Validate inbound `Content-Type` for XML/SOAP.
- Return clear HTTP status for bad method/path/type.
- Capture raw request and response payloads.
- Add tests for wrong path, wrong method, wrong content type.

Reference needed:

- XTP / SOAP/HTTPS transport rules,
- expected controller content type,
- expected endpoint path.

Exit criteria:

- bad binding requests fail predictably.
- valid SOAP request reaches operation detection.
- raw evidence captured for each first-message test.

## Pass 3 — SOAP and XML namespace correctness

Goal: stop relying on loose XML detection for core startup.

Tasks:

- Define SOAP namespace constants.
- Define G2S namespace constants per selected G2S version.
- Parse `Envelope`, `Header` if present, `Body`, and G2S body separately.
- Extract `g2sBody` routing fields: `hostId`, `egmId`, `sessionId`, `sessionType`.
- Keep permissive lab mode as an option, but default core tests should use strict mode.
- Add tests for namespace mismatch and missing required fields.

Reference needed:

- WSDL/XSD files,
- sample controller XML,
- exact namespace URIs,
- sample `commsOnLine`, `commsOnLineAck`, `keepAlive`, `keepAliveAck` transcripts.

Exit criteria:

- parser can reject malformed SOAP.
- parser extracts routing/session fields reliably.
- operation detection uses parsed body, not regex-like root guessing.

## Pass 4 — Spec-correct VEGM session engine

Goal: implement the VEGM-side G2S communications startup as a first-class engine.

Tasks:

- Add `session_engine` package or runtime module.
- On instance start, optionally initiate G2S session to `host_endpoint`.
- Send `commsOnLine`.
- Receive and validate `commsOnLineAck`.
- Handle descriptor exchange as required.
- Configure or observe keepalive flow.
- Maintain `session_state`, `host_connection_state`, `last_comms_online_at`, `last_keepalive_at`, `last_ack_at`, `last_error`.
- Add reconnect/backoff policy.

Reference needed:

- expected startup direction for the controller under test,
- `commsOnLine` XML schema,
- `commsOnLineAck` XML schema,
- descriptor startup expectations,
- `setKeepAlive` / `keepAlive` direction and timing for the controller.

Exit criteria:

- one VEGM can complete startup against a mock host.
- state shows online only after correct ACK.
- failed ACK or timeout is visible in supervisor.

## Pass 5 — Inbound command receiver for controller actions

Goal: keep and harden Host-to-VEGM command handling.

Tasks:

- Keep `/g2s` listener for controller commands.
- Route mute/restore/hold/lock commands via G2S operation mapping.
- Update normalized state:
  - `audio_state`,
  - `cabinet_lock_state`,
  - `hold_state`,
  - `machine_state`.
- Return operation-specific ACK/fault response.
- Capture state transition evidence.

Reference needed:

- controller command XML for mute/restore,
- controller command XML for cabinet lock/unlock,
- controller command XML for hold/release,
- expected ACKs/faults,
- command namespace/class.

Exit criteria:

- mock command changes only the targeted VEGM.
- supervisor shows state change within one poll cycle.
- raw XML and state history are saved.

## Pass 6 — Registered hosts and permissions

Goal: model the EGM registered-host configuration used by G2S communication control.

Tasks:

- Add registered host table per VEGM.
- Fields:
  - `hostId`,
  - host URL,
  - role: owner/configurator/guest,
  - allowed device classes,
  - certificate identity if TLS is on.
- Reject or fault commands from unregistered hosts in strict mode.
- Expose registered host config in supervisor settings editor.

Reference needed:

- commConfig registered-host behavior,
- expected host roles for mute controller,
- hostId used by controller,
- permissions needed for cabinet/audio/lock classes.

Exit criteria:

- strict mode blocks unknown host.
- owner role allows control commands.
- UI shows host registration status.

## Pass 7 — TLS and certificate workflow

Goal: make TLS setup testable without guessing.

Tasks:

- Keep `plaintext_lab` for early testing.
- Add explicit TLS modes:
  - server TLS only,
  - mutual TLS,
  - no revocation,
  - accept-all lab.
- Validate cert/key pair on startup.
- Expose cert status in supervisor.
- Add OpenSSL-friendly runbook outputs.
- Add UI fields for host URL SAN, cert paths, CA path, and trust result.

Reference needed:

- certificate checklist,
- exact host URL,
- DNS vs IP decision,
- CA chain,
- server cert,
- EGM client cert if mutual TLS,
- revocation policy later.

Exit criteria:

- TLS handshake can be proven before G2S.
- supervisor shows cert configuration per VEGM.
- TLS failure is separated from G2S XML failure.

## Pass 8 — Generic UI becomes workable test bench

Goal: generic UI must support actual lab testing, not just display config.

Tasks:

Supervisor UI:

- start/stop/restart each VEGM,
- start/stop all,
- show EGM endpoint and host endpoint,
- show registered host status,
- show session/heartbeat state,
- show mute/lock/hold/machine state,
- show last inbound command,
- show last outbound G2S message,
- show last error,
- link to raw payloads/evidence.

Per-VEGM UI:

- live status panel,
- raw last request/response,
- session timeline,
- command/state timeline,
- endpoint/cert view.

Reference needed:

- lab operator workflow,
- required status fields for pass/fail,
- which state names the team wants on screen,
- sample runbook output format.

Exit criteria:

- tester can determine from UI whether the controller connected and sent the correct command.
- tester does not need CLI for normal validation.

## Pass 9 — Evidence and runbook export

Goal: every test run can be reviewed and handed off.

Tasks:

- Add run ID.
- Export bundle per VEGM and fleet.
- Include:
  - config snapshot,
  - endpoint/cert snapshot,
  - raw XML,
  - parsed message summary,
  - state transitions,
  - pass/fail notes.
- Add runbook checklist JSON/HTML export.

Reference needed:

- lab runbook template,
- evidence requirements,
- expected pass/fail gate.

Exit criteria:

- one click export from supervisor.
- evidence contains first-message worksheet values.
- failed run includes reason bucket.

## Pass 10 — Scale to floor size

Goal: reliably run 15-30 VEGMs.

Tasks:

- Convert `go run` launch to built binary launch.
- Add per-process logs.
- Add health check and restart policy.
- Add port collision detection.
- Add endpoint collision detection.
- Add resource usage display.
- Add fleet groups/banks.

Reference needed:

- expected number of test EGMs,
- PC resources,
- controller endpoint limits,
- required bank/group layout.

Exit criteria:

- 15 VEGMs start and stay healthy.
- 30 VEGMs start if resources allow.
- UI remains usable.

## 9. Immediate next coding pass

Next pass should be:

**Pass 1 + part of Pass 2**

Specific tasks:

1. Add `hostId` as first-class config/fleet field.
2. Add `egm_endpoint` and `host_endpoint` structures.
3. Preserve current generated config compatibility.
4. Update supervisor editor to show:
   - hostId,
   - egmId,
   - EGM endpoint,
   - host endpoint,
   - protocol,
   - path.
5. Add validation:
   - path must not be empty,
   - scheme must be `http` or `https`,
   - port must be valid,
   - hostId and egmId must be present.
6. Add tests for endpoint resolution.

Do not add more UI features until these are stable.

## 10. Reference checklist before coding next pass

Before Pass 1/2 implementation, collect or confirm:

- G2S version targeted by the controller.
- Controller host URL for session startup.
- Whether controller expects HTTP or HTTPS first.
- `hostId` value.
- `egmId` naming pattern.
- G2S endpoint path.
- Whether `commsOnLine` is expected first from EGM.
- Whether descriptor exchange is mandatory before commands.
- Keepalive direction and interval.
- Mute command XML sample.
- Restore command XML sample.
- Lock command XML sample.
- Hold command XML sample.
- TLS requirement for first lab run.
- Cert SAN naming convention if TLS is required.

## 11. Done definition

The project is ready for controller qualification when:

1. Fleet starts from one manifest.
2. Each VEGM has correct G2S endpoint and identity.
3. Each VEGM can establish G2S session with the host/controller.
4. Keepalive remains stable.
5. Controller can send mute/restore/hold/lock to targeted VEGM.
6. Only targeted VEGM state changes.
7. Supervisor shows state and evidence clearly.
8. Run export contains raw XML, parsed fields, state transitions, and pass/fail summary.
