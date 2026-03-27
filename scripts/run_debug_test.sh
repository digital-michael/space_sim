#!/bin/bash
# Run performance test with comprehensive debug logging and timeout

TIMEOUT=120  # 2 minutes - should be enough for 10+ tests
LOG_FILE="performance_debug.log"
CONSOLE_FILE="test_console.log"

echo "=========================================="
echo "Performance Test with Debug Logging"
echo "Timeout: ${TIMEOUT}s"
echo "Debug log: $LOG_FILE"
echo "Console log: $CONSOLE_FILE"
echo "Started: $(date)"
echo "=========================================="
echo ""

# Clean up old logs
rm -f "$LOG_FILE" "$CONSOLE_FILE"

if [ ! -f ./bin/space-sim ]; then
    echo "Building application binary..."
    go build -o bin/space-sim ./cmd/space-sim
fi

# Run test with timeout using perl (available on macOS)
echo "Running test with 8 threads on worst profile..."
perl -e 'alarm shift @ARGV; exec @ARGV' "$TIMEOUT" ./bin/space-sim --performance --profile worst --threads 8 > "$CONSOLE_FILE" 2>&1 &
PID=$!

# Wait for completion or timeout
wait $PID
EXIT_CODE=$?

echo ""
echo "=========================================="
echo "Test finished with exit code: $EXIT_CODE"
echo "Ended: $(date)"
echo "=========================================="
echo ""

# Analyze exit code
if [ $EXIT_CODE -eq 0 ]; then
    echo "✓ Test completed successfully"
elif [ $EXIT_CODE -eq 124 ]; then
    echo "❌ Test TIMED OUT after ${TIMEOUT}s"
    echo ""
    echo "Last 20 log entries:"
    tail -20 "$LOG_FILE"
    echo ""
    echo "Searching for last frame processed..."
    grep "\[FRAME" "$LOG_FILE" | tail -5
    echo ""
    echo "Searching for last test started..."
    grep "\[TEST.*Starting:" "$LOG_FILE" | tail -3
elif [ $EXIT_CODE -eq 130 ]; then
    echo "⚠ Test interrupted by user (Ctrl+C)"
else
    echo "⚠ Test exited with error code: $EXIT_CODE"
fi

echo ""
echo "Log file summary:"
echo "  Total log lines: $(wc -l < "$LOG_FILE")"
echo "  Tests started: $(grep -c "\[TEST.*Starting:" "$LOG_FILE")"
echo "  Tests completed: $(grep -c "Test completed:" "$LOG_FILE")"
echo "  Frames processed: $(grep -c "\[FRAME.*Starting frame" "$LOG_FILE")"
echo "  DrawSphereEx calls: $(grep -c "Calling DrawSphereEx" "$LOG_FILE")"
echo ""

# Check for application termination message
echo "Application exit reason:"
grep "Application Terminated:" "$LOG_FILE" | tail -1

# Show last few entries
echo ""
echo "Last 10 log entries:"
tail -10 "$LOG_FILE"

exit $EXIT_CODE
