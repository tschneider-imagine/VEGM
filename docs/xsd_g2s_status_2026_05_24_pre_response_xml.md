# XSD G2S Pre-Response XML Metadata Note — 2026-05-24

Reference anchor: `projectplan.md` section 12, `docs/xsd_g2s_status_2026_05_24.md`, and `docs/xsd_g2s_status_2026_05_24_ui_note.md`.

## Latest pass status

This pass addresses the previous caveat that XML metadata only appeared after a parsed response was recorded.

## Landed

Runtime now keeps configured XML mode metadata separately from parsed response evidence.

Changed / added files:

- `runtime/xml_mode_registry.go`
- `runtime/config.go`
- `runtime/session_timestamps.go`
- `runtime/session_parse_evidence_state_test.go`

Behavior:

- `ValidateConfig` registers configured XML mode metadata for the instance.
- `/control/state` uses parsed evidence values when available.
- `/control/state` falls back to configured XML metadata before any parsed response is received.

Runtime `/control/state` now exposes before first parsed ACK:

- `g2s_xml_mode`
- `g2s_xml_namespace`
- `g2s_xml_egm_location`

Test coverage:

- `TestRuntimeStateJSONIncludesConfiguredXMLMetadataBeforeParsedResponse`

## Supervisor status

The supervisor already displays:

- XML mode
- XML namespace
- EGM location
- parsed root kind
- parsed class
- parsed operation
- raw root
- expected ACK
- actual ACK

These fields are sourced from each child VEGM's `/control/state` fetch.

## Next validation command

```powershell
git pull
go test ./...
```

## Next recommended pass after green

Add supervisor editor fields and save/restart support for:

- XML mode
- G2S namespace
- EGM location

Then allow switching per VEGM between:

- `lab_legacy_xml`
- `xsd_g2s_message`

Keep `lab_legacy_xml` as the default until intentionally changed.
