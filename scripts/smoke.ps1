param(
    [string]$BaseUrl = "http://127.0.0.1:17860"
)

$ErrorActionPreference = "Stop"

try {
    $health = Invoke-RestMethod -Uri "$BaseUrl/api/v1/health" -TimeoutSec 10
    if ($health.status -ne "ok") {
        throw "健康状态为 $($health.status)：db=$($health.db) cache=$($health.cache)"
    }

    $meta = Invoke-RestMethod -Uri "$BaseUrl/api/v1/meta" -TimeoutSec 10
    if ($meta.api_version -ne "v1") {
        throw "api_version 不符：$($meta.api_version)"
    }

    Write-Host "SMOKE OK: version=$($health.version) db=$($health.db) cache=$($health.cache)"
} catch {
    Write-Error $_.Exception.Message
    exit 1
}
