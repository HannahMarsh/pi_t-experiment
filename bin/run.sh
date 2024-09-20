#!/bin/bash

# Path to the config.yml file
CONFIG_FILE="config/config.yml"

# Array to hold the PIDs of the terminal processes
shutdown_addresses=()

pids=()

# Function to kill all processes started in the terminals and close the terminals
terminate_processes() {
    echo "Terminating all processes and closing terminals..."
    for addr in "${shutdown_addresses[@]}"; do
        echo "Shutting down relay at $addr"
        curl -X POST "$addr/shutdown" > /dev/null 2>&1
    done

    for pid in "${pids[@]}"; do
        kill -9 $pid
    done
    exit 0
}

# Set up a trap to catch the "q" input or SIGINT (Ctrl+C)
trap "terminate_processes" SIGINT

# Check if the correct number of parameters are provided
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <numberRelays> <numberClients>"
    exit 1
fi

# Assign parameters to variables
numberRelays=$1
numberClients=$2

# Print the parameters
echo "Number of relays: $numberRelays"
echo "Number of clients: $numberClients"

# Find the root directory of the project by locating a known file or directory
# For example, let's assume .git exists in the root of the project
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"

if [ -z "$PROJECT_ROOT" ]; then
    echo "Error: Unable to determine the project root directory. Are you inside a Git repository?"
    exit 1
fi

# Change to the project root directory
cd "$PROJECT_ROOT" || { echo "Failed to change directory to $PROJECT_ROOT"; exit 1; }

# Use yq to extract the host and port for the given client ID
HOST=$(yq e ".bulletin_board | .host" $CONFIG_FILE)
PORT=$(yq e ".bulletin_board | .port" $CONFIG_FILE)

if [ -z "$HOST" ] || [ -z "$PORT" ]; then
  echo "Bulletin board not found."
  exit 1
fi

ADDRESS="http://$HOST:$PORT"

echo "Client $id address: $ADDRESS"

shutdown_addresses+=("$ADDRESS")


# Start bulletin board in a new terminal (using osascript for macOS)
osascript -e 'tell app "Terminal"
    do script "cd '"$PROJECT_ROOT"' && go run cmd/bulletin-board/main.go; exit"
end tell' &

SCRIPT_PID=$!
pids+=("$SCRIPT_PID")

# Loop through each client and start in a new terminal
for ((id=1; id<=numberClients; id++))
do
    # Use yq to extract the host and port for the given client ID
    HOST=$(yq e ".clients[] | select(.id == $id) | .host" $CONFIG_FILE)
    PORT=$(yq e ".clients[] | select(.id == $id) | .port" $CONFIG_FILE)

    if [ -z "$HOST" ] || [ -z "$PORT" ]; then
      echo "Client with ID $id not found."
      exit 1
    fi

    ADDRESS="http://$HOST:$PORT"

    echo "Client $id address: $ADDRESS"

    shutdown_addresses+=("$ADDRESS")

    osascript -e 'tell app "Terminal"
        do script "cd '"$PROJECT_ROOT"' && go run cmd/client/main.go -id '"$id"' && exit"
    end tell' &

    SCRIPT_PID=$!
    pids+=("$SCRIPT_PID")


done

# Loop through each relay and start in a new terminal
for ((id=1; id<=numberRelays; id++))
do
    # Use yq to extract the host and port for the given client ID
    HOST=$(yq e ".relays[] | select(.id == $id) | .host" $CONFIG_FILE)
    PORT=$(yq e ".relays[] | select(.id == $id) | .port" $CONFIG_FILE)

    if [ -z "$HOST" ] || [ -z "$PORT" ]; then
      echo "Relay with ID $id not found."
      exit 1
    fi

    ADDRESS="http://$HOST:$PORT"

    echo "Relay $id address: $ADDRESS"

    shutdown_addresses+=("$ADDRESS")
    osascript -e 'tell app "Terminal"
        do script "cd '"$PROJECT_ROOT"' && go run cmd/relay/main.go -id '"$id"' && exit"
    end tell' &

    SCRIPT_PID=$!
    pids+=("$SCRIPT_PID")
done

# Wait for all background processes to complete
# Wait for the user to press "q" or send a SIGINT (Ctrl+C)
while true; do
    sleep 1
done

# Exit the script
exit 0
