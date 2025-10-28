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

    openChannel() {
        if (!this.channelManager) { alert('Connect wallets first'); return; }
        const peerEth = document.getElementById('peer-eth-addr').value.trim();
        const peerSol = document.getElementById('peer-sol-addr').value.trim();
        const myEth = parseFloat(document.getElementById('my-eth').value.trim() || '0');
        const mySol = parseFloat(document.getElementById('my-sol').value.trim() || '0');
        const peerEthAmt = parseFloat(document.getElementById('peer-eth').value.trim() || '0');
        const peerSolAmt = parseFloat(document.getElementById('peer-sol').value.trim() || '0');
        const challenge = parseInt(document.getElementById('challenge-duration').value.trim() || '0', 10);
        this.channelManager.openChannel({
            peerAddressEth: peerEth, peerAddressSol: peerSol,
            myEthDeposit: myEth, mySolDeposit: mySol, peerEthDeposit: peerEthAmt,
            peerSolDeposit: peerSolAmt, challengeDuration: challenge
        });
    }

    closeChannel() { if (this.channelManager) this.channelManager.closeChannel(); }

    refreshChannel() { if (this.channelManager) this.channelManager.refreshChannelInfo(); }

    createOrder() {
        if (!this.orderBookManager) { alert('Connect wallets first.'); return; }
        const side = document.getElementById('order-side').value, baseAsset = document.getElementById('base-asset').value.trim(), quoteAsset = document.getElementById('quote-asset').value.trim(),
            price = document.getElementById('order-price').value.trim(), amount = document.getElementById('order-amount').value.trim();
        this.orderBookManager.createOrder({ side, baseAsset, quoteAsset, price, amount });
    }

    cancelOrder(orderId) { if (this.orderBookManager) this.orderBookManager.cancelOrder(orderId); }

    acceptOrder(orderId) { if (this.orderBookManager) this.orderBookManager.acceptOrder(orderId); }

    clearLog() { document.getElementById('log-container').innerHTML = ''; }

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
