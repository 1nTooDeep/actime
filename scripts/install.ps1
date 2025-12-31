# Installation script for Actime (Windows)

Write-Host "Installing Actime..." -ForegroundColor Green

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed. Please install Go 1.21 or higher." -ForegroundColor Red
    exit 1
}

# Build the project
Write-Host "Building Actime..." -ForegroundColor Yellow
make build

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

# Create installation directory
$installDir = "$env:LOCALAPPDATA\Actime"
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

# Copy binaries
Write-Host "Installing binaries to $installDir..." -ForegroundColor Yellow
Copy-Item "build\actime.exe" -Destination "$installDir\actime.exe"
Copy-Item "build\actimed.exe" -Destination "$installDir\actimed.exe"

# Add to PATH (optional)
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$installDir*") {
    Write-Host "Adding Actime to user PATH..." -ForegroundColor Yellow
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$installDir", "User")
}

# Create data directory
$dataDir = "$env:USERPROFILE\.actime"
if (-not (Test-Path $dataDir)) {
    New-Item -ItemType Directory -Path $dataDir | Out-Null
}

# Copy default config
if (-not (Test-Path "$dataDir\config.yaml")) {
    Write-Host "Creating default configuration..." -ForegroundColor Yellow
    Copy-Item "configs\config.yaml" -Destination "$dataDir\config.yaml"
}

Write-Host "Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Usage:" -ForegroundColor Cyan
Write-Host "  actime stats     - View usage statistics"
Write-Host "  actimed start    - Start the daemon"
Write-Host "  actimed stop     - Stop the daemon"
Write-Host "  actimed status   - Check daemon status"
Write-Host ""
Write-Host "Configuration file: $dataDir\config.yaml" -ForegroundColor Cyan
Write-Host ""
Write-Host "Note: You may need to restart your terminal for PATH changes to take effect." -ForegroundColor Yellow