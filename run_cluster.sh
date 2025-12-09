#!/bin/bash

# --- Configuration ---
BINARY_NAME="impl-raft-go"
RAFT_BINARY="/Users/khalidrafi/Projects/GO/impl-raft-go/bin/${BINARY_NAME}"
ADDRESSES="1=127.0.0.1:8081,2=127.0.0.1:8082,3=127.0.0.1:8083"

# Function to spawn a command in a new Terminal window
spawn_terminal() {
    NODE_ID=$1
    # FIX: Escape the double quotes around the ADDRESSES value using a backslash.
    # The shell will now pass the inner quotes literally to the osascript block.
    COMMAND="${RAFT_BINARY} --id=${NODE_ID} --addrs=\\\"${ADDRESSES}\\\""
    
    # Use osascript to tell the Terminal application to execute the command
    osascript -e "
        tell application \"Terminal\"
            do script \"echo 'Starting Raft Node ${NODE_ID}...'; ${COMMAND}\"
            set custom title of front window to \"Raft Node ${NODE_ID}\"
            activate
        end tell
    "
}

echo "Starting Raft Cluster on 3 separate Terminal windows..."

spawn_terminal 1
spawn_terminal 2
spawn_terminal 3

echo "All nodes launched."