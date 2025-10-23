#!/bin/bash
set -e
ACCOUNTS_DIR="accounts"
ADDRESS_DIR="addresses"
ALICE=$(solana-keygen pubkey $ACCOUNTS_DIR/alice.json)
BOB=$(solana-keygen pubkey $ACCOUNTS_DIR/bob.json)
FEE_PAYER_KEYPAIR="$ACCOUNTS_DIR/fee_payer.json"

echo "[airdrop] Funding fee payer..."
FEE_PAYER=$(solana-keygen pubkey "$FEE_PAYER_KEYPAIR")
solana airdrop 20 "$FEE_PAYER"

echo "[airdrop] Funding Alice and Bob..."
solana airdrop 1 "$ALICE"
solana airdrop 1 "$BOB"

echo "[token] Creating mint..."
MINT=$(spl-token create-token --fee-payer "$FEE_PAYER_KEYPAIR" | grep -oP '(?<=Creating token )\w+')

echo "$MINT" > "$ADDRESS_DIR/mint.txt"

echo "[done] MINT=$MINT"

echo "[token] Creating token accounts..."
ATA_ALICE=$(spl-token create-account --fee-payer "$FEE_PAYER_KEYPAIR" "$MINT" --owner "$ALICE"  | grep -oP '(?<=Creating account )\w+')
ATA_BOB=$(spl-token create-account "$MINT" --fee-payer "$FEE_PAYER_KEYPAIR" --owner "$BOB"  | grep -oP '(?<=Creating account )\w+')

sleep 1  # Wait for accounts to be created

echo "[token] Minting tokens..."
spl-token mint "$MINT" 100 "$ATA_ALICE"
spl-token mint "$MINT" 100 "$ATA_BOB"

echo "[done] MINT=$MINT"
echo "[done] ATA_ALICE=$ATA_ALICE"
echo "[done] ATA_BOB=$ATA_BOB"

echo "[info] Alice's token account: $ATA_ALICE"
spl-token balance --address "$ATA_ALICE"

echo "[info] Bob's token account: $ATA_BOB"
spl-token balance --address "$ATA_BOB"