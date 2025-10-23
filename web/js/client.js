window.log = function (message, type = 'info') {
    const consoleFunc = type === 'error' ? 'error' : 'log';
    console[consoleFunc](message);
};


class PerunClient {
    constructor() {
        this.walletManager = new WalletManager();
        this.ws = null;
        this.channelManager = null;
        this.orderBookManager = null;
        this.userName = USER_NAME; // Set in each HTML file
    }

    async connectWallets() {
        try {
            this.updateConnectionStatus('Connecting wallets...');

            // Always disconnect previous connections to avoid cached sessions
            await this.walletManager.disconnect();

            // Connect MetaMask explicitly
            const ethAddr = await this.walletManager.connectMetaMask();
            document.getElementById('eth-address').textContent =
                ethAddr.substring(0, 10) + '...' + ethAddr.substring(ethAddr.length - 8);

            // Connect Phantom explicitly
            const solAddr = await this.walletManager.connectPhantom();
            document.getElementById('sol-address').textContent =
                solAddr.substring(0, 10) + '...' + solAddr.substring(solAddr.length - 8);

            this.log(`Wallets connected: ETH ${ethAddr}, SOL ${solAddr}`, 'success');

            // Initialize WebSocket connection
            this.ws = new WSConnection(this.userName, this.walletManager);
            this.channelManager = new ChannelManager(this.ws);
            this.orderBookManager = new OrderBookManager(this.ws, this.channelManager);

            this.ws.connect();

            // Update UI buttons
            document.getElementById('connect-btn').style.display = 'none';
            document.getElementById('disconnect-btn').style.display = 'inline-block';

        } catch (error) {
            this.log(`Wallet connection failed: ${error.message}`, 'error');
            this.updateConnectionStatus('Disconnected', false);
            alert('Failed to connect wallets: ' + error.message);
        }
    }

    async disconnect() {
        if (!this.walletManager) return;

        await this.walletManager.disconnect();

        if (this.ws && this.ws.ws) {
            this.ws.ws.close();
        }

        this.updateConnectionStatus('Disconnected', false);
        document.getElementById('connect-btn').style.display = 'inline-block';
        document.getElementById('disconnect-btn').style.display = 'none';
        document.getElementById('eth-address').textContent = 'Not connected';
        document.getElementById('sol-address').textContent = 'Not connected';

        this.log('Disconnected wallets and WebSocket.', 'info');
    }

    // ... rest of your class methods (openChannel, closeChannel, createOrder, etc.) remain same ...

    updateConnectionStatus(text, connected = null) {
        const statusEl = document.getElementById('connection-status');
        statusEl.textContent = text;
        if (connected !== null) {
            statusEl.className = connected ? 'status-indicator connected' : 'status-indicator';
        }
    }

    log(message, type = 'info') {
        const logContainer = document.getElementById('log-container');
        const entry = document.createElement('div');
        entry.className = `log-entry ${type}`;
        const timestamp = new Date().toLocaleTimeString();
        entry.textContent = `[${timestamp}] ${message}`;
        logContainer.appendChild(entry);
        logContainer.scrollTop = logContainer.scrollHeight;
    }
}

const client = new PerunClient();
window.client = client; // expose globally
