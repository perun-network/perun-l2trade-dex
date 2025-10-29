<h1 align="center"><br>
    <a href="https://perun.network/"><img src=".assets/perun-logo.svg" alt="Perun" width="196"></a>
<br></h1>

# PerunL2Trade - Layer 2 Cross-chain Decentralized Exchange

**PerunL2Trade**: A cross-chain decentralized exchange prototype leveraging Perun state channels for trustless, off-chain trading between Ethereum and Solana. Combines real-time WebSocket communication with browser-based wallet integration (MetaMask + Phantom) to enable fast, low-cost asset swaps through payment channel updates and an integrated off-chain order book.

## Features

- **Cross-chain State Channels**: Create, manage, and settle Perun payment channels
- **WebSocket Architecture**: Real-time communication for payment channel operations.
- **Asset Swapping**: Atomic cross-chain swaps with configurable balances for ETH and SPL tokens, executed through signed state transitions
- **Dual-Wallet Integration**: Seamless support for MetaMask (Ethereum) and Phantom (Solana) directly in the browser—no custodial intermediaries required
- **Off-Chain Order Book**: Create, match, and execute orders entirely off-chain within state channels, minimizing on-chain fees and latency.
- **Browser-Based UI**: Interactive web interface for testing multi-party flows (Alice/Bob scenarios), monitoring channel states, and managing orders without installation

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
│   └── index.html        # Home interface
│   └── alice.html        # Alice interface
│   └── bob.html          # Bob interface
│   └── css/              # Style definition
│   └── js/               # Script definition
├── go.mod                # Go module dependencies
└── README.md             # This file
```

## Demo Flow

1. **Connection**: Alice and Bob connect to the WebSocket server and register with their wallet addresses.
1. **Channel Creation**: Alice creates a payment channel with initial ETH/SOL balances
2. **Swap Proposal**: Alice proposes to send ETH in exchange for SOL from Bob
3. **Swap Execution**: Bob accepts the swap, and balances are updated
4. **Settlement**: Either party can settle the channel to finalize balances

## Quick Start

### Prerequisites

- **Go** 1.21+
- **Node.js** 18+ (for frontend tooling)
- **Ganache** (Ethereum local node)
- **Solana CLI** tools + local validator
- **MetaMask** browser extension
- **Phantom** browser extension
- **Modern web browser**
  
### Installation

1. Clone the repository:
```bash
git clone https://github.com/perun-network/perun-dex-websocket.git
cd perun-dex-websocket
```

2. Install dependencies:
```bash
go mod tidy
```


### Backend Setup
1. Setup Solana Local Validators on one separate terminal and deploy the on-chain Perun Program:
```bash
make dev
```

2. Setup Ethereum Node on a separate terminal with prefunded accounts:
```sh
KEY_DEPLOYER=0x79ea8f62d97bc0591a4224c1725fca6b00de5b2cea286fe2e0bb35c5e76be46e
KEY_ALICE=0x1af2e950272dd403de7a5760d41c6e44d92b6d02797e51810795ff03cc2cda4f
KEY_BOB=0xf63d7d8e930bccd74e93cf5662fde2c28fd8be95edb70c73f1bdd863d07f412e
BALANCE=100000000000000000000

ganache -h 127.0.0.1 --port 8545 --chain.chainId 1337 --wallet.accounts $KEY_DEPLOYER,$BALANCE $KEY_ALICE,$BALANCE $KEY_BOB,$BALANCE -b 5 -g "0xC92A69C00"
```

3. Run the server on a separate terminal:
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

-ethChains: Ethereum chains YAML, default ethereum_chains.yaml.​

-solChains: Solana chains YAML, default chains_solana.yaml.​

-cert, -certKey: TLS certificate and key for HTTPS; omit for HTTP.​

-defaultTimeout, -handleTimeout, -fundTimeout, -settleTimeout: operation timeouts.​

-finalityDepth: required chain confirmations for transactions.​

-predefinedGasLimit: enable predefined gas limits for adjudicator/depositors.​

-horizonURL: compatibility flag retained; not used for Solana in this setup.​
``` 
### WebApp Demo

Use two separated browsers each with Metamask and Phantom extensions in developer mode (to access local node): Our recommnedation are Brave and Chrome.

1. Setup Alice and Bob account for Ethereum and Solana Local node:

ETHEREUM:
```
NODE_URL: http://127.0.0.1:8545, with ChainID: 1337
KEY_ALICE=0x1af2e950272dd403de7a5760d41c6e44d92b6d02797e51810795ff03cc2cda4f
KEY_BOB=0xf63d7d8e930bccd74e93cf5662fde2c28fd8be95edb70c73f1bdd863d07f412e
```

SOLANA:
```
RPC: LOCAL_NET
KEY_ALICE= [222,173,120,13,82,179,86,138,108,25,133,223,242,34,134,202,22,245,150,192,231,167,129,95,140,240,211,238,135,125,115,224,171,39,222,86,185,232,54,33,215,92,18,217,198,82,81,216,192,255,207,30,215,201,11,14,95,192,101,167,3,202,196,252]
KEY_BOB= [61,14,6,45,125,154,131,29,18,142,93,70,56,114,199,39,211,184,230,181,183,211,224,96,41,160,147,45,200,137,104,33,149,168,119,94,130,232,190,23,107,74,113,163,144,160,227,20,177,172,52,136,66,240,148,211,155,186,30,111,121,78,110,24]
```


2. Open Alice interface with: http://localhost:8080/alice.html

3. Open Bob interface with http://localhost:8080/bob.html

4. Connect both clients to the websocket with `Connect Wallets` button.

5. **Propose a payment channel (Alice)**
   - Set Alice's initial balances (default: 5 ETH, 0 SOL)
   - Set Bob's initial balances (default: 0 ETH, 2 SOL)
   - Click "Create Payment Channel" in Alice's panel
   - The payment require signatures from both wallets.

6. **Create an Order (Alice)**
   - Choose either `Bid` or `Ask` (default Bid)
   - Choose base token (default SOL)
   - Choose quote token (default ETH)
   - Price in quote token (default 0.04)
   - Amount of the base token to swap (default 1)
   - This means: Alice wants to swap 1 SOL for 0.04 ETH.
   - Click `Create Order` button to confirm.

7. **Accept and Execute an Order (Bob)**
   - An order will appear on bob side.
   - If he finds the swap fair, click on `Accept`
   - Else he can create a counter `Ask` order.
   - The default is: Bob wants to sell 1 SOL for 0.05 ETH.
   - After the Order is executed, the balance of the channel will be updated. 

8. **Settle Channel (Either)**:
   - Click `Close Channel` finalize and close the channel
   - The final balance will be withdrawn to the corresponding wallets
   - View the Solana's on-chain transactions regarding the Perun Program: [Solana-Exporer](https://explorer.solana.com/address/GQtQCW4dREybk2FR1gabaSb89CFxGrNS74JX5fZ97Qmh/domains?cluster=custom)


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
