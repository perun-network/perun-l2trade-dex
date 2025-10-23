#!/bin/bash
set -e

SCRIPTS_DIR="scripts"
ACCOUNTS_DIR="accounts"
ADDRESS_DIR="addresses"

if [ -d $ACCOUNTS_DIR ]; then
  rm -rf $ACCOUNTS_DIR/*
fi
mkdir -p $ACCOUNTS_DIR

if [ -d $ADDRESS_DIR ]; then
  rm -rf $ADDRESS_DIR/*
fi
mkdir -p $ADDRESS_DIR

# Ensure CLI is localnet
solana config set -ul

echo "[keygen] Generating fee payer..."
solana-keygen new --no-bip39-passphrase --force --outfile $ACCOUNTS_DIR/fee_payer.json

echo "[setup] Generating Alice deterministic keypair"
expect <<EOF
spawn solana-keygen recover "prompt://?key=0/0" --force --outfile accounts/alice.json
expect "seed phrase:"
send "better shield palace essay armed tonight pull smart walk cram ill pond\r"
expect "If this seed phrase has an associated passphrase"
send "\r"
expect "Continue? (y/n):"
send "y\r"
expect eof
EOF

echo "[setup] Generating Bob deterministic keypair"
expect <<EOF
spawn solana-keygen recover "prompt://?key=0/0" --force --outfile accounts/bob.json
expect "seed phrase:"
send "unfold skin essence coin south tower north stereo bleak primary dizzy measure\r"
expect "If this seed phrase has an associated passphrase"
send "\r"
expect "Continue? (y/n):"
send "y\r"
expect eof
EOF
