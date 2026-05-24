# XSD G2S Validation Script Status Note — 2026-05-24

Reference anchor: `projectplan.md` section 12 and prior XSD handoff notes.

## Latest pass status

A PowerShell validation helper was added to make one-VEGM XML-mode verification easier.

## Landed

New file:

- `scripts/test-g2s-xml-mode.ps1`

The script checks:

- generated child config path
- `g2s_xml.mode`
- `g2s_xml.namespace`
- `g2s_xml.egm_location`
- recent payload XML files
- whether payloads look XSD-shaped:
  - `g2sMessage`
  - `g2sBody`
  - `communications`
- whether payloads look legacy/SOAP-shaped:
  - `soapenv:Envelope`
  - `soap:Envelope`
  - `Envelope`

## Recommended validation flow

1. Keep fleet default as `lab_legacy_xml`.
2. Use supervisor editor to switch only `vegm-001` to:

```text
xsd_g2s_message
```

3. Save + Restart `vegm-001`.
4. Initiate or Force Heartbeat for `vegm-001`.
5. Run:

```powershell
.\scripts\test-g2s-xml-mode.ps1 -InstanceId vegm-001 -ShowPayloadMatches
```

Expected after XSD mode switch:

- config mode shows `xsd_g2s_message`
- XSD-shaped matches are greater than zero

To verify an untouched legacy VEGM:

```powershell
.\scripts\test-g2s-xml-mode.ps1 -InstanceId vegm-002 -ShowPayloadMatches
```

Expected for legacy mode:

- config mode shows `lab_legacy_xml`
- legacy/SOAP-shaped matches are greater than zero

## Next validation command

```powershell
git pull
go test ./...
```

## Next recommended pass after green

1. Run the validation helper against `vegm-001` and `vegm-002`.
2. If generated child configs do not preserve `g2s_xml` from the manifest before supervisor save, patch the Go fleet resolver/generator.
3. If validation works, add a small UI hint beside G2S XML Mode listing valid values:
   - `lab_legacy_xml`
   - `xsd_g2s_message`
