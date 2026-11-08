# Local CI entry. No paid SaaS; same commands for developers and self-hosted runs.
# Covers the gates listed in testing-strategy.md section 9, including PostgreSQL
# integration tests and Playwright E2E against the container stack.
# NOTE: keep this file ASCII-only. Windows PowerShell 5 reads BOM-less UTF-8 as
# ANSI, so non-ASCII comments can swallow the following statements.
param(
    # Regenerate Playwright visual baselines instead of comparing them.
    [switch]$UpdateE2E
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

# Some machines point npm/golangci-lint caches at unwritable locations; keep
# every tool cache inside the repo instead (.tmp is gitignored).
$env:npm_config_cache = Join-Path $root ".tmp/npm-cache"
$env:GOLANGCI_LINT_CACHE = Join-Path $root ".tmp/golangci-lint"

function Require-Command($name, $installHint) {
    $cmd = Get-Command $name -ErrorAction SilentlyContinue
    if ($null -eq $cmd) { throw "Missing $name. Install it first: $installHint" }
    return $cmd.Source
}

Write-Host "== backend ==" -ForegroundColor Cyan
Push-Location (Join-Path $root "backend")
try {
    $unformatted = gofmt -l .
    if ($unformatted) { throw "gofmt failed:`n$unformatted" }

    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

    $lintPath = Require-Command "golangci-lint" "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0"
    & $lintPath run ./...
    if ($LASTEXITCODE -ne 0) { throw "golangci-lint failed" }

    $vulnPath = Require-Command "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest"
    & $vulnPath ./...
    if ($LASTEXITCODE -ne 0) { throw "govulncheck failed" }

    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed" }
    Write-Host "SKIP: go test -race (no cgo toolchain on Windows; run CGO_ENABLED=1 go test -race ./... on Linux/container)" -ForegroundColor Yellow

    $serverOut = Join-Path $root ".tmp/server-ci.exe"
    $cliOut = Join-Path $root ".tmp/cli-ci.exe"
    go build -o $serverOut ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "server build failed" }
    go build -o $cliOut ./cmd/cli
    if ($LASTEXITCODE -ne 0) { throw "cli build failed" }
    Remove-Item $serverOut, $cliOut -ErrorAction SilentlyContinue
} finally {
    Pop-Location
}

Write-Host "== integration ==" -ForegroundColor Cyan
Push-Location $root
try {
    docker compose up -d --wait db
    if ($LASTEXITCODE -ne 0) { throw "db did not become ready" }

    $dbPassword = if ($env:POSTGRES_PASSWORD) { $env:POSTGRES_PASSWORD } else { "nm_local_only" }
    $env:TEST_DATABASE_URL = "postgres://novel:$dbPassword@127.0.0.1:5432/novel?sslmode=disable"

    Push-Location (Join-Path $root "backend")
    try {
        go test -tags=integration ./internal/integration/...
        if ($LASTEXITCODE -ne 0) { throw "integration tests failed" }
    } finally {
        Pop-Location
    }
} finally {
    Pop-Location
}

Write-Host "== frontend ==" -ForegroundColor Cyan
Push-Location (Join-Path $root "frontend")
try {
    npm ci
    if ($LASTEXITCODE -ne 0) { throw "npm ci failed" }

    npm run lint
    if ($LASTEXITCODE -ne 0) { throw "lint failed" }

    npm run typecheck
    if ($LASTEXITCODE -ne 0) { throw "typecheck failed" }

    npm test
    if ($LASTEXITCODE -ne 0) { throw "unit tests failed" }

    npm run build
    if ($LASTEXITCODE -ne 0) { throw "build failed" }
} finally {
    Pop-Location
}

Write-Host "== image build ==" -ForegroundColor Cyan
Push-Location $root
try {
    docker compose build
    if ($LASTEXITCODE -ne 0) { throw "docker compose build failed" }
} finally {
    Pop-Location
}

Write-Host "== e2e ==" -ForegroundColor Cyan
Push-Location $root
try {
    docker compose up -d
    if ($LASTEXITCODE -ne 0) { throw "docker compose up failed" }

    $baseUrl = "http://127.0.0.1:17860"
    $deadline = (Get-Date).AddMinutes(3)
    while ($true) {
        try {
            $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/health" -TimeoutSec 3
            if ($response.status -eq "ok") { break }
        } catch {
            if ((Get-Date) -gt $deadline) { throw "service did not become healthy at $baseUrl" }
            Start-Sleep -Seconds 3
        }
    }

    Push-Location (Join-Path $root "frontend")
    try {
        $script = if ($UpdateE2E) { "test:e2e:update" } else { "test:e2e" }
        npm run $script
        if ($LASTEXITCODE -ne 0) { throw "playwright e2e failed" }
    } finally {
        Pop-Location
    }
} finally {
    Pop-Location
}

Write-Host "CI PASSED" -ForegroundColor Green
