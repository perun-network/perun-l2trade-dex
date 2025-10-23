#!/bin/bash
set -e

PROGRAM_PATH="contract/perun_solana_program.so"
ADDRESS_DIR="addresses"

echo "[deploy] Deploying program..."
DEPLOY_OUTPUT=$(solana program deploy "$PROGRAM_PATH")

# Extract program ID from deploy output
PROGRAM_ID=$(echo "$DEPLOY_OUTPUT" | grep -oP '(?<=Program Id: )\w+')

# Write to file
echo "$PROGRAM_ID" > "$ADDRESS_DIR/perun_address.txt"
echo "[done] PROGRAM_ID=$PROGRAM_ID"
