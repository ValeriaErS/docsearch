Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "  NAGRUZOCHNOE TESTIROVANIE DOCSEARCH" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -ErrorAction Stop
    Write-Host "Server is running" -ForegroundColor Green
} catch {
    Write-Host "Server is not running. Run: ./docsearch.exe serve" -ForegroundColor Red
    exit 1
}

Write-Host ""
go run scripts/load.go