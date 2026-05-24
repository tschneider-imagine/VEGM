# XSD G2S UI Evidence Display Note — 2026-05-24

Reference anchor: `projectplan.md` section 12 and `docs/xsd_g2s_status_2026_05_24.md`.

## Latest pass status

The supervisor evidence display patch landed after the runtime XML metadata pass.

Changed file:

- `webui/static/supervisor.html`

The supervisor Live State evidence block now displays:

- XML mode
- XML namespace
- EGM location
- parsed root kind
- parsed class
- parsed operation
- raw root
- expected ACK
- actual ACK

Runtime `/control/state` already exposes the source fields:

- `g2s_xml_mode`
- `g2s_xml_namespace`
- `g2s_xml_egm_location`
- `last_parsed_root_kind`
- `last_parsed_class`
- `last_parsed_operation`
- `last_raw_root`
- `last_expected_ack`
- `last_actual_ack`

## Validation

Run:

```powershell
git pull
go test ./...
```

Then rebuild/restart supervisor and hard refresh the browser.

## Remaining caveat

The XML metadata fields are still populated through the parsed-response evidence path. They appear after `recordParsedResponseEvidence(...)` runs. A future pass should expose configured `g2s_xml` before the first parsed response.

## Next recommended pass

Expose configured `g2s_xml.mode`, namespace, and EGM location before first parsed response, then add supervisor editor fields for XML mode, namespace, and EGM location.
