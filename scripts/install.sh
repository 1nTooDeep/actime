#!/bin/bash
# Installation script for Actime (Linux)

set -e

echo "Installing Actime..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or higher."
    exit 1
fi

# Build the project
echo "Building Actime..."
make build

# Install binaries
echo "Installing binaries to /usr/local/bin..."
sudo cp build/actime /usr/local/bin/
sudo cp build/actimed /usr/local/bin/
sudo chmod +x /usr/local/bin/actime
sudo chmod +x /usr/local/bin/actimed

# Create data directory
echo "Creating data directory..."
mkdir -p ~/.actime

# Copy default config
if [ ! -f ~/.actime/config.yaml ]; then
    echo "Creating default configuration..."
    cp configs/config.yaml ~/.actime/config.yaml
fi

echo "Installation complete!"
echo ""
echo "Usage:"
echo "  actime stats     - View usage statistics"
echo "  actimed start    - Start the daemon"
echo "  actimed stop     - Stop the daemon"
echo "  actimed status   - Check daemon status"
echo ""
echo "Configuration file: ~/.actime/config.yaml"