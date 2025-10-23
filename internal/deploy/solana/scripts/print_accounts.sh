#!/bin/bash

ACCOUNTS_DIR="accounts"
echo "[info] Alice: $(solana-keygen pubkey $ACCOUNTS_DIR/alice.json)"
echo "[info] Bob:   $(solana-keygen pubkey $ACCOUNTS_DIR/bob.json)"
