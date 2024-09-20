#!/bin/bash

# Find the root directory of the project (Git repository)
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"

if [ -z "$PROJECT_ROOT" ]; then
    echo "Error: Unable to determine the project root directory. Are you inside a Git repository?"
    exit 1
fi

# Path to the prometheus.yml file relative to the project root
CONFIG_FILE="$PROJECT_ROOT/config/prometheus.yml"

# Check if the Prometheus config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Prometheus configuration file not found at $CONFIG_FILE"
    exit 1
fi

# Start Prometheus using the configuration file
/opt/homebrew/bin/prometheus --config.file="$CONFIG_FILE"
