#!/bin/bash
# Run Gego with Local MongoDB

echo "🚀 Starting Gego with Local MongoDB..."
echo "📍 MongoDB: localhost:27017"
echo ""

export GEGO_ENV=local
export MONGODB_DATABASE=gego

# Check if MongoDB is running
if ! command -v mongod &> /dev/null; then
    echo "⚠️  MongoDB doesn't appear to be installed"
    echo "   Install: https://docs.mongodb.com/manual/installation/"
fi

# Check if MongoDB service is running (Linux)
if command -v systemctl &> /dev/null; then
    if ! systemctl is-active --quiet mongod; then
        echo "⚠️  MongoDB service is not running"
        echo "   Start it with: sudo systemctl start mongod"
    fi
fi

# Check if MongoDB service is running (macOS with Homebrew)
if command -v brew &> /dev/null; then
    if ! brew services list | grep -q "mongodb.*started"; then
        echo "⚠️  MongoDB service is not running"
        echo "   Start it with: brew services start mongodb-community"
    fi
fi

echo "🏃 Executing: gego $@"
echo ""

# Use local gego binary if it exists, otherwise use system gego
if [ -f "./gego" ]; then
    exec ./gego "$@"
elif command -v gego &> /dev/null; then
    exec gego "$@"
else
    echo "❌ Error: gego binary not found"
    echo "   Build it with: go build -o gego cmd/gego/main.go"
    echo "   Or install it: go install github.com/fissionx/gego/cmd/gego@latest"
    exit 1
fi

