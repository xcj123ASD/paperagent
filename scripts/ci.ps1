$ErrorActionPreference = "Stop"

Write-Host "=== gofmt ===" -ForegroundColor Cyan
$unformatted = gofmt -l .
if ($unformatted) {
    Write-Host "unformatted files:" -ForegroundColor Red
    $unformatted | ForEach-Object { Write-Host "  $_" }
    Write-Host "run: gofmt -w ." -ForegroundColor Yellow
    exit 1
}
Write-Host "OK" -ForegroundColor Green

Write-Host "=== go vet ===" -ForegroundColor Cyan
go vet ./...
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host "OK" -ForegroundColor Green

Write-Host "=== go build ===" -ForegroundColor Cyan
go build ./...
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host "OK" -ForegroundColor Green

Write-Host "=== go test ===" -ForegroundColor Cyan
go test ./...
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host "OK" -ForegroundColor Green

Write-Host "=== govulncheck ===" -ForegroundColor Cyan
$govuln = Join-Path (go env GOPATH) "bin\govulncheck.exe"
if (-not (Test-Path $govuln)) {
    Write-Host "govulncheck not found. Run: go install golang.org/x/vuln/cmd/govulncheck@latest" -ForegroundColor Red
    exit 1
}
& $govuln ./...
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host "OK" -ForegroundColor Green

Write-Host ""
Write-Host "ALL PASSED" -ForegroundColor Green