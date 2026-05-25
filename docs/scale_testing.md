# VEGM Scale Testing Workflow

Reference anchor: `projectplan.md`, Pass 10 — Scale to floor size.

## Purpose

Use this workflow to generate repeatable fleet manifests for 15 or 30 simulated VEGMs on the current lab network.

Current lab defaults:

- VEGM host IP: `192.168.10.161`
- Controller endpoint: `http://tspi4.local:8444/g2s`
- Subnet mask: `255.255.255.0`
- Gateway: `192.168.10.1`
- Wire port base: `18443`
- Control port base: `19001`

## Generate a 15-VEGM manifest

```powershell
Set-Location C:\Users\SnowM\Documents\GitHub\VEGM
.\scripts\new-fleet-manifest.ps1 -Count 15 -OutFile .\generated\fleet-15.json
```

## Generate a 30-VEGM manifest

```powershell
Set-Location C:\Users\SnowM\Documents\GitHub\VEGM
.\scripts\new-fleet-manifest.ps1 -Count 30 -OutFile .\generated\fleet-30.json
```

## Build binaries

```powershell
go build -o .\bin\vegm.exe .\cmd\vegm
go build -o .\bin\vegm-supervisor.exe .\cmd\vegm-supervisor
```

## Run 15-VEGM supervisor

```powershell
$env:VEGM_CHILD_BINARY = "$PWD\bin\vegm.exe"
.\bin\vegm-supervisor.exe -manifest .\generated\fleet-15.json -serve
```

## Run 30-VEGM supervisor

```powershell
$env:VEGM_CHILD_BINARY = "$PWD\bin\vegm.exe"
.\bin\vegm-supervisor.exe -manifest .\generated\fleet-30.json -serve
```

## Supervisor UI

Open:

```text
http://127.0.0.1:18081/ui/supervisor.html
```

Use the UI to start machines, inspect status, and export evidence.

## What to verify

- No collision errors on manifest load
- All generated wire ports are unique
- All generated control ports are unique
- Each VEGM row has correct host endpoint: `http://tspi4.local:8444/g2s`
- Each VEGM row has correct EGM host endpoint IP: `192.168.10.161`
- Start one VEGM first
- Then start a small group
- Then start all

## Expected port ranges

For 15 VEGMs:

- Wire: `18443-18457`
- Control: `19001-19015`

For 30 VEGMs:

- Wire: `18443-18472`
- Control: `19001-19030`
