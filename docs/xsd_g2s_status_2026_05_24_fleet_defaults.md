# XSD G2S Fleet Defaults Status Note — 2026-05-24

Reference anchor: `projectplan.md` section 12 and prior XSD handoff notes.

## Latest pass status

This pass added explicit G2S XML metadata to fleet generation and the example fleet manifest.

## Landed

Changed files:

- `scripts/new-fleet-manifest.ps1`
- `example.fleet.json`

The PowerShell generator now accepts:

- `-G2SXmlMode`
- `-G2SXmlNamespace`

Defaults remain:

```text
G2SXmlMode      = lab_legacy_xml
G2SXmlNamespace = http://www.gamingstandards.com/g2s/schemas/v1.0.3
```

Generated manifests now include explicit `g2s_xml` blocks at:

- `defaults.g2s_xml`
- each generated instance's `g2s_xml`

Each instance gets an EGM location derived from its wire port:

```text
<VEGM host IP>:<wire port>
```

Example:

```text
192.168.10.161:18443
192.168.10.161:18444
```

The checked-in `example.fleet.json` now documents the same explicit `g2s_xml` shape for defaults and the sample instances.

## Validation

Run:

```powershell
git pull
go test ./...
```

Then regenerate a manifest:

```powershell
.\scripts\new-fleet-manifest.ps1 -Count 15 -OutFile .\generated\fleet-15.json
Select-String .\generated\fleet-15.json -Pattern 'g2s_xml','xsd_g2s_message','lab_legacy_xml'
```

Optional XSD-mode generation test:

```powershell
.\scripts\new-fleet-manifest.ps1 -Count 3 -OutFile .\generated\fleet-xsd-3.json -G2SXmlMode xsd_g2s_message
Select-String .\generated\fleet-xsd-3.json -Pattern 'g2s_xml','xsd_g2s_message'
```

## Next recommended pass after green

1. Verify whether the Go fleet manifest resolver copies manifest `g2s_xml` into generated child config JSON.
2. If not, patch the fleet resolver/generator structs so generated `vegm-*.json` files contain `g2s_xml` without needing supervisor editor save.
3. Add tests in the fleet package for generated config `G2SXML` fields.
4. Keep default mode `lab_legacy_xml` unless explicitly changed.
