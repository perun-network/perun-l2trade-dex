<<<<<<< HEAD
# Perun DEX WebSocket
=======
# Perun DEX WebSocket Node
>>>>>>> 5c2da37 (feat(internal): add client and order book logic)

A WebSocket-based Perun client node for cross-ledger state channels on Ethereum and Solana, featuring on-chain chain/asset configuration and an off-chain order book integrated into the same WebSocket flow.

## Features

- **WebSocket Server**: Real-time communication for payment channel operations
- **Dual Asset Support**: ETH and SOL token swapping
- **Payment Channels**: Create, manage, and settle Perun payment channels
- **Swap Mechanism**: Propose, accept/reject, and execute asset swaps
- **Web Interface**: Interactive frontend for testing Alice and Bob interactions

## Project Structure

```
├── cmd/server/           # Main server application
├── internal/
│   ├── websocket/        # WebSocket hub and message handlers
│   ├── message/          # Type definitions and message structures
│   ├── client/           # Perun client wrapper, channel store, request handler, and order book routing hook.
│   └── wallet/           # Wallet definition
│   └── orderbook/        # In-memory per-channel order book engine and optional streaming WS endpoint.
├── web/                  # Frontend interface
│   └── index.html        # Demo web interface
├── go.mod                # Go module dependencies
└── README.md             # This file
```

## Demo Flow

1. **Connection**: Alice and Bob connect to the WebSocket server
2. **Registration**: Both clients register with their wallet addresses
3. **Channel Creation**: Alice creates a payment channel with initial ETH/SOL balances
4. **Swap Proposal**: Alice proposes to send ETH in exchange for SOL from Bob
5. **Swap Execution**: Bob accepts the swap, and balances are updated
6. **Settlement**: Either party can settle the channel to finalize balances

## Quick Start

### Prerequisites

- Go 1.21 or later
- Modern web browser

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd perun-dex-websocket
```

2. Install dependencies:
```bash
go mod tidy
```

3. Setup Solana Local Validators on one separate terminal and deploy the on-chain Perun Program:
```bash
make dev
```

4. Setup Ethereum Node on a separate terminal with prefunded accounts:
```sh
KEY_DEPLOYER=0x79ea8f62d97bc0591a4224c1725fca6b00de5b2cea286fe2e0bb35c5e76be46e
KEY_ALICE=0x1af2e950272dd403de7a5760d41c6e44d92b6d02797e51810795ff03cc2cda4f
KEY_BOB=0xf63d7d8e930bccd74e93cf5662fde2c28fd8be95edb70c73f1bdd863d07f412e
BALANCE=100000000000000000000

ganache -h 127.0.0.1 --port 8545 --chain.chainId 1337 --wallet.accounts $KEY_DEPLOYER,$BALANCE $KEY_ALICE,$BALANCE $KEY_BOB,$BALANCE -b 5 
```

5. Run the server on a separate terminal:
```bash
go run cmd/server/main.go \
  -addr 127.0.0.1:8080 \
  -ethChains chains_ethereum.yaml \
  -solChains chains_solana.yaml \
  -defaultTimeout 1m \
  -handleTimeout 5m \
  -fundTimeout 10m \
  -settleTimeout 10m \
  -finalityDepth 1 \
  -predefinedGasLimit=false \
  -cert "" \
  -certKey ""
```

Runtime flags:
```bash
-addr: HTTP/WebSocket listen address, default 127.0.0.1:8080.​

-ethChains: Ethereum chains YAML, default chains.yaml.​

-solChains: Solana chains YAML, default chains_solana.yaml.​

-cert, -certKey: TLS certificate and key for HTTPS; omit for HTTP.​

-defaultTimeout, -handleTimeout, -fundTimeout, -settleTimeout: operation timeouts.​

-finalityDepth: required chain confirmations for transactions.​

-predefinedGasLimit: enable predefined gas limits for adjudicator/depositors.​

-horizonURL: compatibility flag retained; not used for Solana in this setup.​
```


### Usage Instructions

1. **Connect Clients**:
   - Click "Connect Alice" to connect Alice
   - Click "Connect Bob" to connect Bob
   - Both should show "Connected" status

2. **Create Payment Channel**:
   - Set Alice's initial balances (default: 5 ETH, 0 SOL)
   - Set Bob's initial balances (default: 0 ETH, 1000 SOL)
   - Click "Create Payment Channel" in Alice's panel

3. **Execute Swap**:
   - In Alice's panel, set the ETH amount to send and SOL amount to request
   - Click "Propose ETH → SOL Swap"
   - In Bob's panel, click "Accept Swap" or "Reject Swap"

4. **Settle Channel**:
   - Click "Settle Channel" in Alice's panel to finalize and close the channel

## API Reference

### WebSocket Messages

Primary control plane:

`ws://<host>/connect` expects an initialization message and then supports typed JSON requests for chain queries, balances, channel operations, and the order book control API.​

Optional streaming feed:

`ws://<host>/ws/orderbook?channel=<channel_id>` streams an initial OrderBookSnapshot and subsequent OrderBookDelta updates in sequence order.​

### Order book API
Messages over `/connect`:

```
CreateOrder -> CreateOrderAck: Insert a new order into the per-channel book.​

CancelOrder -> CancelOrderAck: Remove an active order by ID.​

AcceptOrder -> AcceptOrderAck: Signal acceptance intent; settlement follows via a Perun channel update.​

GetOrderBook -> GetOrderBookResponse: Return a snapshot of the current per-channel book.​
```
Streaming deltas over `/ws/orderbook`:

Initial `OrderBookSnapshot` frame followed by `OrderBookDelta` frames that include added/updated/removed orders and totalOpen with a monotonic sequence.

## Typical Flow
- Connect to `/connect` and initialize in Cross-Contract mode with ETH and SOL client addresses.​

- Query `GetChains/GetAssets` to discover known networks and assets from your YAML config.​

- Open a Perun channel and exchange updates using `OpenChannel/UpdateChannel/CloseChannel.`​

- CreateOrder to propose a trade; the counterparty uses `AcceptOrder` then executes a channel update to settle balances off-chain.​

- Subscribe to `/ws/orderbook` for live deltas while maintaining a local snapshot.​



## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.
