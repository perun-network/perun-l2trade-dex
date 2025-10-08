# Perun DEX WebSocket

A WebSocket-based backend demonstrating Perun payment channels for ETH/SOL asset swapping between Alice and Bob.

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
│   ├── websocket/       # WebSocket hub and message handlers
│   ├── types/           # Type definitions and message structures
│   ├── perun/           # Perun client integration (placeholder)
│   └── swap/            # Swap logic and exchange rates
├── web/                 # Frontend interface
│   └── index.html       # Demo web interface
├── go.mod               # Go module dependencies
└── README.md           # This file
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

3. Run the server:
```bash
go run cmd/server/main.go
```

4. Open your browser and navigate to:
```
http://localhost:8080
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

#### Client Registration
```json
{
  "type": "register",
  "data": {
    "user_id": "alice",
    "address": "0x742d35Cc6634C0532925a3b8D2Ba8d1FF4dA7e42",
    "is_alice": true
  }
}
```

#### Channel Creation
```json
{
  "type": "create_channel",
  "data": {
    "partner_user_id": "bob",
    "initial_balances": {
      "ETH": "5000000000000000000",
      "SOL": "0"
    },
    "challenge_duration": 3600
  }
}
```

#### Swap Proposal
```json
{
  "type": "swap_propose",
  "data": {
    "channel_id": "uuid",
    "from_asset": "ETH",
    "to_asset": "SOL",
    "from_amount": "1000000000000000000",
    "to_amount": "50000000000",
    "exchange_rate": "50.0"
  }
}
```

#### Swap Acceptance
```json
{
  "type": "swap_accept",
  "data": {
    "channel_id": "uuid",
    "swap_id": "uuid"
  }
}
```

## Exchange Rates

Current mock exchange rates:
- 1 ETH = 50 SOL
- 1 SOL = 0.02 ETH

## Technical Implementation

### WebSocket Hub
- Manages client connections and message routing
- Handles channel state management
- Broadcasts messages to channel participants

### Message Handlers
- `handleRegister`: Client registration and authentication
- `handleCreateChannel`: Payment channel creation
- `handleSwapPropose`: Swap proposal processing
- `handleSwapAccept/Reject`: Swap response handling
- `handleSettle`: Channel settlement

### Asset Management
- ETH amounts in wei (1 ETH = 10^18 wei)
- SOL amounts in lamports (1 SOL = 10^9 lamports)
- Big integer arithmetic for precise calculations

## Development Notes

This is a demonstration implementation focusing on the WebSocket communication layer and UI interaction. In a production environment, you would need to integrate with:

- Real Perun Go client implementation
- Actual blockchain connections (Ethereum, Solana)
- Proper cryptographic signatures
- State channel consensus mechanisms
- Real exchange rate feeds

## Testing

The web interface provides a complete testing environment:
- Real-time message logging
- Interactive controls for both Alice and Bob
- Visual feedback for connection status and balances
- Error handling and display

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
