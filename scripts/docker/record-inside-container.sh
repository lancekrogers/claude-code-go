#!/bin/bash
# Runs inside Docker container to record a demo
# Usage: record-inside-container.sh <demo-name>

set -e

DEMO_NAME="$1"
if [ -z "$DEMO_NAME" ]; then
    echo "Usage: $0 <demo-name>"
    exit 1
fi

BINARY="/demo/bin/demo-${DEMO_NAME}"
EXPECT_SCRIPT="/expect/${DEMO_NAME}.exp"
CAST_FILE="/output/${DEMO_NAME}.cast"

# Handle streaming demo (default demo binary name)
if [ "$DEMO_NAME" = "streaming" ]; then
    BINARY="/demo/bin/demo"
fi

# Verify files exist
if [ ! -f "$BINARY" ]; then
    echo "Error: Binary not found: $BINARY"
    exit 1
fi

if [ ! -f "$EXPECT_SCRIPT" ]; then
    echo "Error: Expect script not found: $EXPECT_SCRIPT"
    exit 1
fi

# Make binary executable
chmod +x "$BINARY"

echo "Recording demo: $DEMO_NAME"
echo "Binary: $BINARY"
echo "Expect script: $EXPECT_SCRIPT"
echo "Output: $CAST_FILE"

# Run expect script with asciinema recording
# The expect script spawns the binary and interacts with it
asciinema rec "$CAST_FILE" \
    --command="expect $EXPECT_SCRIPT" \
    --title="Claude Code Go SDK - ${DEMO_NAME} demo" \
    --idle-time-limit=2 \
    --overwrite \
    --output-format=asciicast-v2

echo "Recording complete: $CAST_FILE"
