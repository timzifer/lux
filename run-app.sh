#!/bin/bash
# Build and run as macOS .app bundle (required for accessibility on Tahoe+)
set -e

APP_DIR="/tmp/lux.app/Contents"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOG="$SCRIPT_DIR/output.txt"

mkdir -p "$APP_DIR/MacOS"

# Create Info.plist if missing
[ -f "$APP_DIR/Info.plist" ] || cat > "$APP_DIR/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>lux</string>
    <key>CFBundleIdentifier</key>
    <string>com.lux.debug</string>
    <key>CFBundleName</key>
    <string>Lux</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
</dict>
</plist>
EOF

# Build
CGO_ENABLED=0 go build -tags cocoa -o "$APP_DIR/MacOS/lux" ./examples/kitchen-sink/

# Kill previous
pkill -f "lux.app/Contents/MacOS/lux" 2>/dev/null || true
sleep 0.3

# Run directly from bundle path — macOS recognizes it as bundled app
# because the binary lives inside a .app/Contents/MacOS/ structure.
> "$LOG"
"$APP_DIR/MacOS/lux" > "$LOG" 2>&1 &
echo "PID=$!  Log: $LOG"
