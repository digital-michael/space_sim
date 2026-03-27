#!/bin/bash
# Simple test runner without timeout - user can Ctrl+C if needed

LOG_FILE="performance_debug.log"
CONSOLE_FILE="test_console.log"
BINARY="./bin/space-sim"

# Verify binary exists
if [ ! -f "$BINARY" ]; then
    echo "ERROR: Binary not found at $BINARY"
    echo "Run: go build -o bin/space-sim ./cmd/space-sim"
    exit 1
fi

echo "=========================================="
# Clean up old logs
rm -f "$LOG_FILE" "$CONSOLE_FILE"

# Run test with 4 threads on better profile, WITH locking (production config)
"$BINARY" --performance --profile better --threads 4 2>&1 | tee "$CONSOLE_FILE"
EXIT_CODE=$?

echo ""
echo "=========================================="
echo "Test finished with exit code: $EXIT_CODE"
echo "Ended: $(date)"
echo "=========================================="
echo ""

# Show summary
if [ -f "$LOG_FILE" ]; then
    echo "Log file summary:"
    echo "  Total log lines: $(wc -l < "$LOG_FILE" | tr -d ' ')"
    echo "  Tests started: $(grep -c "\[TEST.*Starting:" "$LOG_FILE")"
    echo "  Tests completed: $(grep -c "Test completed:" "$LOG_FILE")"
    echo ""
    echo "Application exit reason:"
    grep "Application Terminated:" "$LOG_FILE" | tail -1
    echo ""
    echo "Last 5 log entries:"
    tail -5 "$LOG_FILE"
fi
