# XSD G2S Status Handoff — 2026-05-24

Reference anchor: `projectplan.md`, section 12, XSD/G2S envelope work.

## Current pass completed

This pass continued the evidence/UI bridge recommended in `projectplan.md` after the previous green `go test ./...` run.

### Runtime state exposure

Parsed G2S response evidence is exposed through the existing child VEGM `/control/state` JSON path by extending `RuntimeState.MarshalJSON` in `runtime/session_timestamps.go`.

JSON fields:

- `last_parsed_root_kind`
- `last_parsed_class`
- `last_parsed_operation`
- `last_raw_root`
- `last_expected_ack`
- `last_actual_ack`

A focused runtime test exists:

- `runtime/session_parse_evidence_state_test.go`

The test verifies that parsed response evidence recorded through `recordParsedResponseEvidence(...)` appears in runtime state JSON.

## Latest pass — XML metadata in runtime state

Status: runtime side landed; supervisor namespace/location display still pending.

Landed:

- `runtime/session_parse_evidence.go`
  - evidence recorder now captures:
    - `G2SXMLMode`
    - `G2SXMLNamespace`
    - `G2SXMLEGMLocation`
- `runtime/session_timestamps.go`
  - `/control/state` JSON now exposes:
    - `g2s_xml_mode`
    - `g2s_xml_namespace`
    - `g2s_xml_egm_location`
- `runtime/session_parse_evidence_state_test.go`
  - test now asserts XML mode, namespace, and EGM location are included in runtime state JSON.

Important caveat:

- These XML metadata fields are currently populated through the parsed-response evidence path. They appear after `recordParsedResponseEvidence(...)` runs. A future pass should expose configured `g2s_xml` directly from runtime config/state so the UI can show the configured mode before the first successful/parsed response.

### Supervisor UI exposure

Current supervisor page displays parsed evidence from each child VEGM's `/control/state` fetch.

Displayed fields in the Live State cell:

- XML mode
- parsed root kind
- parsed class
- parsed operation
- raw root
- expected ACK
- actual ACK

Changed file already in earlier pass:

- `webui/static/supervisor.html`

Attempted but not landed in this latest pass:

- add `g2s_xml_namespace` and `g2s_xml_egm_location` to the supervisor Live State evidence block.

Reason:

- The full `webui/static/supervisor.html` rewrite was blocked by the GitHub write safety layer. This needs a smaller/safe patch approach in the next UI pass.

## Important context

Default XML mode remains:

```text
lab_legacy_xml
```

XSD mode remains opt-in:

```text
xsd_g2s_message
```

Normal session engine strict ACK behavior is still preserved:

- `commsOnLine` expects `commsOnLineAck`
- `getDescriptor` expects `descriptorList`
- `setKeepAlive` expects `setKeepAliveAck`
- `keepAlive` expects `keepAliveAck`

Force Heartbeat generic `g2sResponse` compatibility remains a lab/debug concern only. It must not be generalized into normal session-engine validation.

## Next validation command

```powershell
git pull
go test ./...
```

## Next recommended pass after green

1. Patch supervisor evidence block only, preferably by making the smallest possible change to `evidenceText(state)`, so it displays:
   - `g2s_xml_namespace`
   - `g2s_xml_egm_location`
2. Expose configured `g2s_xml.mode`, namespace, and EGM location before first parsed response.
   - Safe options:
     - add a small dedicated `/control/g2s-xml` endpoint, or
     - add first-class runtime state fields in a small controlled patch, or
     - seed parse-evidence XML metadata during server creation/startup.
3. Add supervisor editor fields for:
   - XML mode
   - G2S namespace
   - EGM location
4. Allow switching per VEGM between:
   - `lab_legacy_xml`
   - `xsd_g2s_message`
5. Regenerate configs cleanly with explicit `g2s_xml` blocks.
6. Update `projectplan.md` section 12 after validation.

## Known safe-patch guidance

Large full-file rewrites have occasionally been blocked by the GitHub write safety layer. Prefer small focused changes, especially for:

- `runtime/server.go`
- `runtime/session_timestamps.go`
- `runtime/outbound.go`
- `webui/static/supervisor.html`

When possible, add focused helper files and tests rather than broad rewrites.
