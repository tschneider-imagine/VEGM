param(
    [string]$Config = "./example.sqlite.vegm.json",
    [string]$HealthUrl = "http://127.0.0.1:19003/healthz",
    [string]$UiUrl = "http://127.0.0.1:19003/ui/scenario-runner.html",
    [int]$MaxWaitSeconds = 60,
    [switch]$NoServer,
    [switch]$NoBrowser
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

if (-not $NoServer) {
    $psExe = if (Get-Command pwsh -ErrorAction SilentlyContinue) { "pwsh" } else { "powershell" }
    $cmd = "Set-Location '$repoRoot'; go run ./cmd/vegm -config '$Config'"
    Start-Process -FilePath $psExe -WorkingDirectory $repoRoot -ArgumentList @("-NoExit", "-Command", $cmd) | Out-Null
}

$ready = $false
for ($i = 0; $i -lt $MaxWaitSeconds; $i++) {
    try {
        $resp = Invoke-RestMethod $HealthUrl -TimeoutSec 2
        if ($resp.ok -eq $true) {
            $ready = $true
            break
        }
    } catch {
        Start-Sleep -Seconds 1
    }
}

if (-not $ready) {
    throw "VEGM did not become healthy at $HealthUrl within $MaxWaitSeconds seconds."
}

Write-Host "VEGM is healthy at $HealthUrl"
Write-Host "UI: $UiUrl"

if (-not $NoBrowser) {
    Start-Process $UiUrl
}
