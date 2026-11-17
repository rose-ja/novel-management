# Performance baseline for SPEC-PERF-001.
# Seeds 1000 characters through the CLI, samples list/search latency,
# measures the first screen with Playwright and writes artifacts/perf/<date>/report.json.
# Keep this file ASCII-only: Windows PowerShell 5 reads BOM-less UTF-8 as ANSI.
param(
    [string]$BaseUrl = "http://127.0.0.1:17860",
    [string]$WorkName = "",
    [int]$Characters = 1000,
    [int]$Samples = 20
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$outDir = Join-Path $root "artifacts/perf/$stamp"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

if (-not $WorkName) { $WorkName = "perf-baseline-$stamp" }

Write-Host "== seed ==" -ForegroundColor Cyan
$seedStdoutPath = Join-Path $outDir "seed.stdout.log"
$seedStderrPath = Join-Path $outDir "seed.stderr.log"
$previousPreference = $ErrorActionPreference
$ErrorActionPreference = "Continue"
docker compose run --rm cli seed --work $WorkName --characters $Characters 1> $seedStdoutPath 2> $seedStderrPath
$seedExit = $LASTEXITCODE
$ErrorActionPreference = $previousPreference
$seedOutput = Get-Content -Path $seedStdoutPath -Raw
if ($seedExit -ne 0) {
    Write-Host (Get-Content -Path $seedStderrPath -Raw)
    throw "seed command failed with exit code $seedExit"
}
Write-Host $seedOutput.Trim()
if ($seedOutput -notmatch "work_id=([0-9a-zA-Z]+)") { throw "seed failed: work_id not found" }
$workID = $Matches[1]

function Measure-Request([string]$Uri) {
    $watch = [System.Diagnostics.Stopwatch]::StartNew()
    $null = Invoke-RestMethod -Uri $Uri -TimeoutSec 30
    $watch.Stop()
    return $watch.Elapsed.TotalMilliseconds
}

function Get-P95($Values) {
    $sorted = @($Values | Sort-Object)
    $index = [Math]::Ceiling($sorted.Count * 0.95) - 1
    return [double]$sorted[$index]
}

Write-Host "== samples ==" -ForegroundColor Cyan
$listSamples = @()
$searchSamples = @()
# 性能人物 URL-encoded to stay ASCII-only in this script.
$searchQuery = "%E6%80%A7%E8%83%BD%E4%BA%BA%E7%89%A9"
for ($i = 0; $i -le $Samples; $i++) {
    $listSamples += Measure-Request "$BaseUrl/api/v1/characters?work_id=$workID&page=1&page_size=50"
    $searchSamples += Measure-Request "$BaseUrl/api/v1/search?work_id=$workID&q=$searchQuery&page=1&page_size=20"
}
# Drop the warm-up request.
$listSamples = @($listSamples | Select-Object -Skip 1)
$searchSamples = @($searchSamples | Select-Object -Skip 1)

$listP95 = Get-P95 $listSamples
$searchP95 = Get-P95 $searchSamples
Write-Host ("list P95: {0:N0}ms  search P95: {1:N0}ms" -f $listP95, $searchP95)

Write-Host "== first screen ==" -ForegroundColor Cyan
$firstScreenFile = Join-Path $outDir "first-screen.json"
$env:PERF_WORK_ID = $workID
$env:PERF_RESULT_FILE = $firstScreenFile
Push-Location (Join-Path $root "frontend")
try {
    npx playwright test e2e/perf.spec.ts --reporter=line
    if ($LASTEXITCODE -ne 0) { throw "playwright first-screen measurement failed" }
} finally {
    Pop-Location
}
$firstScreen = (Get-Content $firstScreenFile -Raw | ConvertFrom-Json).first_screen_ms

$report = [ordered]@{
    generated_at = (Get-Date).ToUniversalTime().ToString("o")
    work_id      = $workID
    base_url     = $BaseUrl
    samples      = $Samples
    thresholds   = [ordered]@{
        first_screen_ms = 2000
        list_p95_ms     = 500
        search_p95_ms   = 150
    }
    results      = [ordered]@{
        first_screen_ms = [Math]::Round([double]$firstScreen, 1)
        list_p95_ms     = [Math]::Round($listP95, 1)
        search_p95_ms   = [Math]::Round($searchP95, 1)
    }
}
$reportPath = Join-Path $outDir "report.json"
$report | ConvertTo-Json -Depth 5 | Set-Content -Path $reportPath -Encoding UTF8
Write-Host "report: $reportPath"

$failures = @()
if ([double]$firstScreen -gt 2000) { $failures += "first screen $firstScreen ms > 2000 ms" }
if ($listP95 -gt 500) { $failures += "list P95 $listP95 ms > 500 ms" }
if ($searchP95 -gt 150) { $failures += "search P95 $searchP95 ms > 150 ms" }
if ($failures.Count -gt 0) { throw ("PERF FAILED: " + ($failures -join "; ")) }

Write-Host "PERF OK" -ForegroundColor Green
