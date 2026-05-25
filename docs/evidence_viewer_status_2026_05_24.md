# Evidence Viewer Status — 2026-05-24

Reference anchor: `projectplan.md` Pass 8 and Pass 9.

## Plan alignment

This pass supports:

- Pass 8 — Generic UI becomes workable test bench
- Pass 9 — Evidence and runbook export
- Ground rule: every protocol feature needs raw XML, parsed message, state change, and timestamp evidence

## Landed

Runtime already exposes a per-VEGM latest evidence endpoint:

```text
/control/evidence/latest
```

A standalone supervisor-served UI page was added:

```text
/ui/evidence-latest.html
```

The page allows a tester to enter a VEGM control URL such as:

```text
http://192.168.10.161:19001
```

and view:

- instance ID
- XML mode
- XML namespace
- EGM location
- expected ACK
- actual ACK
- last error
- latest outbound request metadata
- latest outbound request XML
- latest response metadata
- latest response XML
- raw JSON from `/control/evidence/latest`

## Validation steps

1. Start supervisor.
2. Start a VEGM, for example `vegm-001`.
3. Trigger Initiate or Force Heartbeat.
4. Open:

```text
http://127.0.0.1:18081/ui/evidence-latest.html?control_url=http://192.168.10.161:19001
```

5. Confirm latest request/response XML and parsed fields are visible.

## Known limitation

A direct per-row link from `supervisor.html` to the evidence page was not added in this pass because the supervisor HTML file is compressed and previous broad updates have been fragile. The standalone page is sufficient for testing and can be linked from the supervisor in a later small UI cleanup pass.
