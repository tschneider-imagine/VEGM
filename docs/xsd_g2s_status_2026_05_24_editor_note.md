# XSD G2S Editor Status Note — 2026-05-24

Reference anchor: `projectplan.md` section 12 and the prior XSD handoff notes.

## Latest pass status

Supervisor editor support for G2S XML metadata has been added.

## Landed

Changed files:

- `cmd/vegm-supervisor/settings.go`
- `webui/static/supervisor.html`

Settings API now carries:

- `g2s_xml.mode`
- `g2s_xml.namespace`
- `g2s_xml.egm_location`

Supervisor editor now has fields for:

- G2S XML Mode
- G2S Namespace
- G2S EGM Location

Save / Save + Restart now persists these values into generated child config JSON via `cfg.G2SXML`.

The values are also mirrored into `cfg.Notes` for easy inspection:

- `g2s_xml_mode`
- `g2s_xml_namespace`
- `g2s_xml_egm_location`

## Default behavior

Default remains:

```text
lab_legacy_xml
```

Supported modes remain:

```text
lab_legacy_xml
xsd_g2s_message
```

## Next validation command

```powershell
git pull
go test ./...
```

Then rebuild/restart supervisor and hard refresh the browser.

## Next recommended pass after green

1. Test changing a single VEGM to `xsd_g2s_message` from the supervisor editor.
2. Save + Restart that VEGM.
3. Verify its generated config contains the `g2s_xml` block.
4. Verify `/control/state` shows the selected XML mode/namespace/location before any parsed ACK.
5. Add fleet generation defaults so newly generated manifests/configs include explicit `g2s_xml` blocks.
