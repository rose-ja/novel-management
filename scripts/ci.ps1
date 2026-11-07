# 本地 CI 入口：不依赖付费 SaaS，开发者与自托管环境都执行同一套命令。
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

Write-Host "== 后端 ==" -ForegroundColor Cyan
Push-Location (Join-Path $root "backend")
try {
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet 失败" }

    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test 失败" }

    $serverOut = Join-Path $env:TEMP "novel-server-ci.exe"
    $cliOut = Join-Path $env:TEMP "novel-cli-ci.exe"
    go build -o $serverOut ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "server 构建失败" }
    go build -o $cliOut ./cmd/cli
    if ($LASTEXITCODE -ne 0) { throw "cli 构建失败" }
    Remove-Item $serverOut, $cliOut -ErrorAction SilentlyContinue
} finally {
    Pop-Location
}

Write-Host "== 前端 ==" -ForegroundColor Cyan
Push-Location (Join-Path $root "frontend")
try {
    npm ci
    if ($LASTEXITCODE -ne 0) { throw "npm ci 失败" }

    npm run lint
    if ($LASTEXITCODE -ne 0) { throw "lint 失败" }

    npm run typecheck
    if ($LASTEXITCODE -ne 0) { throw "typecheck 失败" }

    npm test
    if ($LASTEXITCODE -ne 0) { throw "单测失败" }

    npm run build
    if ($LASTEXITCODE -ne 0) { throw "构建失败" }
} finally {
    Pop-Location
}

Write-Host "CI 通过" -ForegroundColor Green
