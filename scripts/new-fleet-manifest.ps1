param(
    [int]$Count = 15,
    [string]$Template = ".\example.fleet.json",
    [string]$OutFile = ".\generated\fleet-$Count.json",
    [string]$VegmHostIp = "192.168.10.162",
    [string]$HostEndpointUrl = "http://192.168.10.25:8444/g2s",
    [string]$SubnetMask = "255.255.255.0",
    [string]$Gateway = "192.168.10.1",
    [int]$WirePortBase = 18443,
    [int]$ControlPortBase = 19001,
    [string]$G2SXmlMode = "lab_legacy_xml",
    [string]$G2SXmlNamespace = "http://www.gamingstandards.com/g2s/schemas/v1.0.3"
)

$ErrorActionPreference = "Stop"

if ($Count -lt 1) {
    throw "Count must be greater than zero."
}

if (!(Test-Path $Template)) {
    throw "Template manifest not found: $Template"
}

$manifest = Get-Content $Template -Raw | ConvertFrom-Json
$manifest.fleet_name = "lab-floor-$Count"
$manifest.description = "$Count-VEGM generated scale manifest for $VegmHostIp talking to $HostEndpointUrl."

$manifest.defaults.listen_host = $VegmHostIp
$manifest.defaults.wire_port_base = $WirePortBase
$manifest.defaults.control_port_base = $ControlPortBase
$manifest.defaults.egm_endpoint.bind_ip = $VegmHostIp
$manifest.defaults.egm_endpoint.host = $VegmHostIp
$manifest.defaults.host_endpoint.url = $HostEndpointUrl
$manifest.defaults.advertised_host = $VegmHostIp
$manifest.defaults.advertised_ip = $VegmHostIp
$manifest.defaults.subnet_mask = $SubnetMask
$manifest.defaults.gateway = $Gateway
$manifest.defaults | Add-Member -MemberType NoteProperty -Name g2s_xml -Value ([ordered]@{
    mode = $G2SXmlMode
    namespace = $G2SXmlNamespace
    egm_location = ("{0}:{1}" -f $VegmHostIp, $WirePortBase)
}) -Force

if ($manifest.profiles.baseline) {
    $manifest.profiles.baseline.advertised_host = $VegmHostIp
}
if ($manifest.profiles.vendorquirk) {
    $manifest.profiles.vendorquirk.advertised_host = $VegmHostIp
    $manifest.profiles.vendorquirk.server_name = $VegmHostIp
}

$bankSwitch = [Math]::Ceiling($Count * 0.67)
$instances = @()

for ($i = 1; $i -le $Count; $i++) {
    $wirePort = $WirePortBase + ($i - 1)
    $controlPort = $ControlPortBase + ($i - 1)
    $group = if ($i -le $bankSwitch) { "bank_a" } else { "bank_b" }

    $inst = [ordered]@{
        instance_id = ("vegm-{0:D3}" -f $i)
        egm_id = ("EGM-{0:D3}" -f $i)
        host_id = $manifest.defaults.host_id
        group = $group
        wire_port = $wirePort
        control_port = $controlPort
        egm_endpoint = [ordered]@{
            host = $VegmHostIp
            port = $wirePort
            path = $manifest.defaults.egm_endpoint.path
        }
        host_endpoint = [ordered]@{
            url = $HostEndpointUrl
        }
        g2s_xml = [ordered]@{
            mode = $G2SXmlMode
            namespace = $G2SXmlNamespace
            egm_location = ("{0}:{1}" -f $VegmHostIp, $wirePort)
        }
        advertised_host = $VegmHostIp
        advertised_ip = $VegmHostIp
        subnet_mask = $SubnetMask
        gateway = $Gateway
    }

    if ($group -eq "bank_b") {
        $inst.overrides = [ordered]@{
            normalized_state = [ordered]@{
                audio_state = "normal"
                hold_state = "inactive"
                lock_state = "inactive"
                machine_state = "available"
            }
        }
    }

    $instances += [pscustomobject]$inst
}

$manifest.instances = $instances

$outDir = Split-Path -Parent $OutFile
if ($outDir -and !(Test-Path $outDir)) {
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
}

$json = $manifest | ConvertTo-Json -Depth 30
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
$outPath = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($OutFile)
[System.IO.File]::WriteAllText($outPath, $json + [Environment]::NewLine, $utf8NoBom)

Write-Host "Wrote $Count-VEGM manifest: $OutFile"
Write-Host "Wire ports: $WirePortBase - $($WirePortBase + $Count - 1)"
Write-Host "Control ports: $ControlPortBase - $($ControlPortBase + $Count - 1)"
Write-Host "VEGM host IP: $VegmHostIp"
Write-Host "Controller URL: $HostEndpointUrl"
Write-Host "G2S XML mode: $G2SXmlMode"
Write-Host "G2S XML namespace: $G2SXmlNamespace"
