#!/bin/bash
# Run Gego with MongoDB Atlas (Cloud)

echo "🚀 Starting Gego with MongoDB Atlas (Cloud)..."
echo "☁️  MongoDB: fissionxgeo.mcwvkmk.mongodb.net"
echo ""

export GEGO_ENV=dev
export MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/"
export MONGODB_DATABASE=gego

echo "📝 Note: Ensure your IP (106.222.202.9) is whitelisted in MongoDB Atlas"
echo "   Network Access → Add IP Address"
echo ""

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

