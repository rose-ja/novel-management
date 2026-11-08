# Local CI entry. No paid SaaS; same commands for developers and self-hosted runs.
# Covers the gates currently available in testing-strategy.md section 9;
# integration/E2E are printed as explicit SKIP while not yet established.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

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
    Write-Host "SKIP: integration tests (not established yet, planned for F1)" -ForegroundColor Yellow

    $serverOut = Join-Path $env:TEMP "novel-server-ci.exe"
    $cliOut = Join-Path $env:TEMP "novel-cli-ci.exe"
    go build -o $serverOut ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "server build failed" }
    go build -o $cliOut ./cmd/cli
    if ($LASTEXITCODE -ne 0) { throw "cli build failed" }
    Remove-Item $serverOut, $cliOut -ErrorAction SilentlyContinue
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
    Write-Host "SKIP: Playwright E2E (not established yet, planned for F6)" -ForegroundColor Yellow
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

Write-Host "CI PASSED" -ForegroundColor Green
