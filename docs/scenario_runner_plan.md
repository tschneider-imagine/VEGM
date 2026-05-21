# Scenario runner plan

## Goal

Provide a browser-first testing surface for VEGM so sequence testing can be run, observed, and repeated without driving the system from PowerShell or curl.

This layer is intentionally **non-blocking**:
- it sits on top of the existing VEGM control and wire endpoints
- it does not require a runtime rewrite before first use
- it can start as a static UI hosted by the existing web host

## Design principles

1. **Do not break the current runtime**
   - keep the current HTTP endpoints as the execution engine
   - add UI and scenario orchestration on top

2. **One scenario engine, many VEGMs**
   - scenarios select one or more target instances
   - each step can act on an instance, group, or all

3. **Observable simulated floor first**
   - the main UX should show per-instance machine/audio/hold/lock/session state live
   - command sequence execution is only useful if the resulting floor state is visible

4. **Mixed profile aware**
   - a logical scenario step can map to different underlying XML/operations depending on instance profile/group

## First UI sections

### A. Instance board
A live grid/table of VEGMs showing:
- instance id
- egm id
- profile/group
- connection state
- registration state
- session state
- heartbeat state
- audio state
- hold state
- lock state
- machine state
- last command type
- last transition time

### B. Scenario editor / viewer
A step list with:
- step id
- selector: instance / group / all
- action
- parameters
- optional wait condition
- optional delay
- enabled/disabled

### C. Scenario controls
- load scenario JSON
- save scenario JSON
- run
- pause
- stop
- step once
- reset selection

### D. Live timeline
A combined event stream of:
- commands sent or observed
- state changes
- warnings/errors
- timestamps

## Step model v1

Each step should look like:

```json
{
  "id": "step-001",
  "selector": {"type": "instance", "value": "vegm-001"},
  "action": "send_logical_command",
  "params": {
    "logical_command": "audio_mute_on",
    "session_id": "S-100"
  },
  "wait_for": {
    "type": "state_match",
    "field": "audio_state",
    "equals": "muted",
    "timeout_ms": 5000
  },
  "delay_after_ms": 500,
  "enabled": true
}
```

## Actions to support first

- `send_logical_command`
- `wait`
- `start_heartbeat`
- `stop_heartbeat`
- `set_heartbeat_rate`
- `snapshot_state`

## Logical commands to support first

- `audio_mute_on`
- `audio_mute_off`
- `hold_on`
- `hold_off`
- `lock_on`
- `lock_off`

## Data source assumptions

The UI should use the existing VEGM endpoints where available:
- `/control/state`
- `/control/state/history`
- `/control/audio`
- `/control/machine-status`
- `/control/pack/summary`
- `/control/pack/operations`

## Non-blocking integration path

### Phase 1
- static HTML/JS page
- configurable base URLs for one or more VEGMs
- manual scenario load/run in browser
- polling-based live state updates

### Phase 2
- serve the same UI from the existing web host
- add scenario persistence
- add group/profile selectors

### Phase 3
- add a fleet supervisor backend and use the same UI against it

## Why this path

It gives a usable UI quickly without blocking on:
- runtime refactors
- supervisor completion
- a full database-backed orchestration layer
