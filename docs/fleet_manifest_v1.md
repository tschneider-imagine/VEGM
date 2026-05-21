# VEGM Fleet Manifest v1

## Purpose

The fleet manifest is the settings model for the **simulated floor**.
It exists so VEGM can represent many EGMs with different identities, profiles, behaviors, and faults while an external G2S controller connects, monitors, and drives commands against that floor.

This is the pivot from:
- one configurable VEGM

to:
- one **floor definition** that can generate and launch many VEGMs consistently.

## Design goals

1. **Fleet-first**
   The primary object is the floor, not a single EGM.

2. **Three override layers**
   Settings can be defined at:
   - fleet default
   - group/profile override
   - per-instance override

3. **Mixed manufacturer simulation**
   Different groups can map the same logical action to different XML operations and state effects.

4. **Observable state**
   Every instance should normalize live state into:
   - audio state
   - hold state
   - lock state
   - machine state
   - session and heartbeat state

5. **Non-blocking implementation path**
   The manifest should be usable first for config generation and later for launching and supervising processes.

## Top-level structure

The manifest contains:
- `schema_version`
- `fleet_name`
- `description`
- `defaults`
- `profiles`
- `groups`
- `instances`

## Defaults

Defaults establish common behavior across the floor.
Typical default domains:
- listen/network defaults
- trust/security defaults
- logging defaults
- storage defaults
- scenario defaults
- registration/session/heartbeat defaults
- normalized floor-state defaults
- fault defaults

## Profiles

A profile represents a protocol flavor or manufacturer-like behavior family.
A profile should define:
- message pack file
- overlays
- logical command mapping
- supported logical commands
- default heartbeat behavior
- default normalized floor-state behavior
- optional notes/version metadata

Example logical commands:
- `audio_mute_on`
- `audio_mute_off`
- `hold_on`
- `hold_off`
- `lock_on`
- `lock_off`

The profile maps those logical commands to actual pack operation names.

## Groups

A group collects instances that should behave similarly.
A group should define:
- selected profile
- optional label
- optional floor zone or bank tag
- optional overrides

Examples:
- `bank_a_igt_like`
- `bank_b_aristocrat_like`
- `bank_c_quirky_lab_mix`

## Instances

An instance is one VEGM process / simulated EGM.
Each instance should carry:
- `instance_id`
- `egm_id`
- `group`
- wire/control addressing
- per-instance log and storage paths
- optional certificate paths
- optional overrides

## Settings buckets

The full test surface should be expressible through the manifest.

### Identity / addressing
- instance id
- egm id
- bind host
- wire port
- control port
- advertised host or IP
- optional DNS / subnet / gateway metadata

### Security
- trust mode
- cert file
- key file
- CA file
- storage backend
- sqlite path

### Registration / session / heartbeat
- registration mode
- allowed hosts
- heartbeat enabled
- heartbeat interval
- heartbeat timeout
- heartbeat jitter
- heartbeat auto-start

### Normalized floor state
- audio state
- hold state
- lock state
- machine state
- session state
- heartbeat state

### Command profile mapping
- logical command to operation mapping
- alternate namespace family
- manufacturer tag
- pack file and overlays

### Fault injection
- delay
- reject
- ignore
- malformed response
- drop connection
- force degraded state

## Resolution rules

When the supervisor generates an effective instance config, resolution should be:
1. fleet defaults
2. profile defaults
3. group overrides
4. instance overrides

That gives a predictable final config for every simulated EGM.

## Immediate use cases

This manifest should support three near-term workflows:

### 1. Generate per-instance VEGM configs
Turn one manifest into many `generated/*.json` runtime configs.

### 2. Create an operator-visible floor definition
The UI can load the manifest and show what the simulated floor is supposed to represent.

### 3. Drive future supervisor launch behavior
The supervisor can later start and monitor all instances directly from this manifest.

## Non-goals for v1

- not yet the full supervisor runtime
- not yet scenario persistence model
- not yet multi-host orchestration
- not yet true per-instance OS-level network provisioning

## Recommended next repo steps

1. add the machine-readable JSON Schema
2. add an example mixed-floor manifest
3. add Go types + validation for the manifest
4. add a `vegm-supervisor` seed that can load and validate the manifest
5. add config generation from the manifest
