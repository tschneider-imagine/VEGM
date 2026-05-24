# XSD G2S Status Handoff — 2026-05-24

Reference anchor: `projectplan.md`, section 12, XSD/G2S envelope work.

## Current pass completed

This pass continued the evidence/UI bridge recommended in `projectplan.md` after the previous green `go test ./...` run.

### Runtime state exposure

Parsed G2S response evidence is now exposed through the existing child VEGM `/control/state` JSON path by extending `RuntimeState.MarshalJSON` in `runtime/session_timestamps.go`.

New JSON fields:

- `last_parsed_root_kind`
- `last_parsed_class`
- `last_parsed_operation`
- `last_raw_root`
- `last_expected_ack`
- `last_actual_ack`

A focused runtime test was added:

- `runtime/session_parse_evidence_state_test.go`

The test verifies that parsed response evidence recorded through `recordParsedResponseEvidence(...)` appears in runtime state JSON.

### Supervisor UI exposure

The supervisor page now displays parsed evidence from each child VEGM's `/control/state` fetch.

Displayed fields in the Live State cell:

- XML mode
- parsed root kind
- parsed class
- parsed operation
- raw root
- expected ACK
- actual ACK

Changed file:

- `webui/static/supervisor.html`

Note: supervisor currently falls back to displaying `lab_legacy_xml` for XML mode if the runtime state does not expose an explicit XML mode field. A future pass should expose the actual configured `g2s_xml.mode` from runtime state or supervisor instance metadata.

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

1. Expose actual `g2s_xml.mode`, namespace, and EGM location in child `/control/state` or supervisor instance metadata.
2. Add supervisor editor fields for:
   - XML mode
   - G2S namespace
   - EGM location
3. Allow switching per VEGM between:
   - `lab_legacy_xml`
   - `xsd_g2s_message`
4. Regenerate configs cleanly with explicit `g2s_xml` blocks.
5. Update `projectplan.md` section 12 after validation.

## Known safe-patch guidance

Large full-file rewrites have occasionally been blocked by the GitHub write safety layer. Prefer small focused changes, especially for:

- `runtime/server.go`
- `runtime/session_timestamps.go`
- `runtime/outbound.go`
- `webui/static/supervisor.html`

When possible, add focused helper files and tests rather than broad rewrites.
