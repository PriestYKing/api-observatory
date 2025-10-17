#!/bin/bash

# Realistic API usage scenarios for API Observatory

set -e

INGEST_URL="http://localhost:8081/api/ingest"
ORG_ID="1"

echo "🚀 API Observatory - Realistic Traffic Simulator"
echo "=================================================="
echo ""

# Scenario 1: Morning Spike (9 AM - lots of users logging in)
morning_spike() {
    echo "📊 Scenario 1: Morning Spike (High Traffic)"
    echo "Simulating morning login rush..."
    go run scripts/traffic-simulator.go \
        -url="$INGEST_URL" \
        -duration=2m \
        -rps=100 \
        -concurrency=20 \
        -org="$ORG_ID"
    echo "✓ Morning spike completed"
    echo ""
}

# Scenario 2: Steady Business Hours
steady_traffic() {
    echo "📈 Scenario 2: Steady Business Hours"
    echo "Simulating normal business day traffic..."
    go run scripts/traffic-simulator.go \
        -url="$INGEST_URL" \
        -duration=5m \
        -rps=50 \
        -concurrency=10 \
        -org="$ORG_ID"
    echo "✓ Steady traffic completed"
    echo ""
}

# Scenario 3: API Spike (Campaign Send, Batch Processing)
spike_event() {
    echo "🔥 Scenario 3: Spike Event (Email Campaign)"
    echo "Simulating bulk email send campaign..."
    go run scripts/traffic-simulator.go \
        -url="$INGEST_URL" \
        -duration=1m \
        -rps=200 \
        -concurrency=30 \
        -org="$ORG_ID"
    echo "✓ Spike event completed"
    echo ""
}

# Scenario 4: Low Traffic (Night Time)
night_traffic() {
    echo "🌙 Scenario 4: Night Time (Low Traffic)"
    echo "Simulating overnight maintenance and monitoring..."
    go run scripts/traffic-simulator.go \
        -url="$INGEST_URL" \
        -duration=1m \
        -rps=10 \
        -concurrency=2 \
        -org="$ORG_ID"
    echo "✓ Night traffic completed"
    echo ""
}

# Scenario 5: Gradual Ramp Up
ramp_up() {
    echo "📶 Scenario 5: Gradual Ramp Up"
    echo "Simulating traffic gradually increasing..."

    for rps in 10 25 50 75 100; do
        echo "  → RPS: $rps"
        go run scripts/traffic-simulator.go \
            -url="$INGEST_URL" \
            -duration=30s \
            -rps=$rps \
            -concurrency=10 \
            -org="$ORG_ID"
    done
    echo "✓ Ramp up completed"
    echo ""
}

# Main menu
echo "Select a scenario to run:"
echo "  1) Morning Spike (2 min, 100 RPS)"
echo "  2) Steady Traffic (5 min, 50 RPS)"
echo "  3) Spike Event (1 min, 200 RPS)"
echo "  4) Night Traffic (1 min, 10 RPS)"
echo "  5) Gradual Ramp Up (2.5 min, 10-100 RPS)"
echo "  6) Full Day Simulation (10 min, all scenarios)"
echo "  7) Continuous Load (run indefinitely)"
echo ""
read -p "Enter choice [1-7]: " choice

case $choice in
    1)
        morning_spike
        ;;
    2)
        steady_traffic
        ;;
    3)
        spike_event
        ;;
    4)
        night_traffic
        ;;
    5)
        ramp_up
        ;;
    6)
        echo "🌍 Running Full Day Simulation..."
        echo ""
        morning_spike
        steady_traffic
        spike_event
        steady_traffic
        night_traffic
        echo "✅ Full day simulation completed!"
        ;;
    7)
        echo "♾️  Running Continuous Load..."
        echo "Press Ctrl+C to stop"
        while true; do
            go run scripts/traffic-simulator.go \
                -url="$INGEST_URL" \
                -duration=1m \
                -rps=50 \
                -concurrency=10 \
                -org="$ORG_ID"
            sleep 5
        done
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "🎉 Simulation complete! Check your dashboard at http://localhost:3000"
