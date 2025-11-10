#!/bin/bash

BASE_URL="http://localhost:8080/v1"
CONCURRENCY=${1:-10}
DURATION=${2:-60}
TARGET_RPS=${3:-100}

echo "Starting load test with:"
echo "  Concurrency: $CONCURRENCY"
echo "  Duration: ${DURATION}s"
echo "  Target RPS: $TARGET_RPS"
echo ""

echo "Testing health endpoint..."
curl -s "$BASE_URL/health" | jq '.status' || echo "Health check failed"

echo ""
echo "Testing health endpoint multiple times..."
for i in {1..5}; do
    echo "Request $i:"
    curl -s "$BASE_URL/health" | jq '.status'
done

echo ""
echo "Load testing health endpoint with curl..."
echo "Running $CONCURRENCY concurrent requests for ${DURATION} seconds..."

# Start background processes for concurrent requests
pids=()
for i in $(seq 1 $CONCURRENCY); do
    (
        while true; do
            curl -s "$BASE_URL/health" > /dev/null 2>&1
            sleep 0.1
        done
    ) &
    pids+=($!)
done

echo "Load test started. Waiting for completion..."
sleep $DURATION

# Kill background processes
for pid in "${pids[@]}"; do
    kill $pid 2>/dev/null
done

echo "Load test completed!"

echo ""
echo "Final health check..."
curl -s "$BASE_URL/health" | jq '.'
