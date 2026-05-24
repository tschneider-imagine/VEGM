param(
    [string]$InstanceId = "vegm-001",
    [string]$GeneratedDir = ".\generated",
    [string]$LogRoot = ".\logs\fleet-a",
    [switch]$ShowPayloadMatches
)

$ErrorActionPreference = "Stop"

$configPath = Join-Path $GeneratedDir "$InstanceId.json"
$payloadDir = Join-Path (Join-Path $LogRoot $InstanceId) "payloads"

Write-Host "G2S XML mode validation"
Write-Host "Instance: $InstanceId"
Write-Host "Config:   $configPath"
Write-Host "Payloads: $payloadDir"
Write-Host ""

if (!(Test-Path $configPath)) {
    throw "Generated config not found: $configPath"
}

$config = Get-Content $configPath -Raw | ConvertFrom-Json
$mode = $config.g2s_xml.mode
$namespace = $config.g2s_xml.namespace
$egmLocation = $config.g2s_xml.egm_location

$missing = @()
if ([string]::IsNullOrWhiteSpace($mode)) { $mode = "<missing>"; $missing += "mode" }
if ([string]::IsNullOrWhiteSpace($namespace)) { $namespace = "<missing>"; $missing += "namespace" }
if ([string]::IsNullOrWhiteSpace($egmLocation)) { $egmLocation = "<missing>"; $missing += "egm_location" }

Write-Host "Configured g2s_xml:"
Write-Host "  mode:        $mode"
Write-Host "  namespace:   $namespace"
Write-Host "  egm_location:$egmLocation"
Write-Host ""

if ($missing.Count -gt 0) {
    Write-Warning "Missing g2s_xml field(s): $($missing -join ', '). Open supervisor editor, save the VEGM settings, or regenerate the manifest/configs with the updated generator."
}

if (!(Test-Path $payloadDir)) {
    Write-Warning "Payload directory not found yet. Start the VEGM and Initiate or Force Heartbeat, then rerun this script."
    exit 0
}

$xmlFiles = Get-ChildItem $payloadDir -Filter *.xml -ErrorAction SilentlyContinue | Sort-Object LastWriteTime -Descending
if (!$xmlFiles -or $xmlFiles.Count -eq 0) {
    Write-Warning "No XML payloads found yet. Initiate or Force Heartbeat, then rerun this script."
    exit 0
}

$xsdPatterns = @("g2sMessage", "g2sBody", "communications")
$legacyPatterns = @("soapenv:Envelope", "soap:Envelope", "Envelope")

$xsdMatches = @()
$legacyMatches = @()
foreach ($file in $xmlFiles) {
    $text = Get-Content $file.FullName -Raw
    foreach ($pattern in $xsdPatterns) {
        if ($text -match [regex]::Escape($pattern)) {
            $xsdMatches += [pscustomobject]@{ File = $file.FullName; Pattern = $pattern; LastWriteTime = $file.LastWriteTime }
            break
        }
    }
    foreach ($pattern in $legacyPatterns) {
        if ($text -match [regex]::Escape($pattern)) {
            $legacyMatches += [pscustomobject]@{ File = $file.FullName; Pattern = $pattern; LastWriteTime = $file.LastWriteTime }
            break
        }
    }
}

Write-Host "Payload evidence:"
Write-Host "  XML payload files: $($xmlFiles.Count)"
Write-Host "  XSD-shaped matches: $($xsdMatches.Count)"
Write-Host "  Legacy/SOAP-shaped matches: $($legacyMatches.Count)"
Write-Host ""

if ($ShowPayloadMatches) {
    if ($xsdMatches.Count -gt 0) {
        Write-Host "Recent XSD-shaped payloads:"
        $xsdMatches | Select-Object -First 10 LastWriteTime, Pattern, File | Format-Table -AutoSize
    }
    if ($legacyMatches.Count -gt 0) {
        Write-Host "Recent legacy/SOAP-shaped payloads:"
        $legacyMatches | Select-Object -First 10 LastWriteTime, Pattern, File | Format-Table -AutoSize
    }
}

Write-Host "Recommendation:"
if ($missing.Count -gt 0) {
    Write-Warning "Config metadata is incomplete. Save from supervisor editor or regenerate configs before judging payload mode."
} elseif ($mode -eq "xsd_g2s_message") {
    if ($xsdMatches.Count -gt 0) {
        Write-Host "  PASS: mode is xsd_g2s_message and XSD-shaped payload evidence was found."
    } else {
        Write-Warning "Mode is xsd_g2s_message but no XSD-shaped payload evidence was found yet. Initiate or Force Heartbeat and rerun."
    }
} elseif ($mode -eq "lab_legacy_xml") {
    if ($legacyMatches.Count -gt 0) {
        Write-Host "  PASS: mode is lab_legacy_xml and legacy/SOAP-shaped payload evidence was found."
    } else {
        Write-Warning "Mode is lab_legacy_xml but no legacy/SOAP payload evidence was found yet. Initiate or Force Heartbeat and rerun."
    }
} else {
    Write-Warning "Unknown or missing g2s_xml.mode: $mode"
}
