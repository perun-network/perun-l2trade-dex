class WalletManager {
    constructor() {
        this.ethereum = null;
        this.solana = null;
        this.ethAddress = null;
        this.solAddress = null;
    }

    async connectMetaMask() {
        if (!window.ethereum) {
            throw new Error('MetaMask not installed. Please install MetaMask extension.');
        }

        try {
            const accounts = await window.ethereum.request({
                method: 'eth_requestAccounts'
            });

            this.ethAddress = accounts[0];
            this.ethereum = window.ethereum;

            window.ethereum.on('accountsChanged', (accounts) => {
                if (accounts.length === 0) {
                    this.ethAddress = null;
                    window.log('MetaMask disconnected', 'warning');
                } else {
                    this.ethAddress = accounts[0];
                    window.log(`MetaMask account changed: ${this.ethAddress}`, 'info');
                }
            });

            window.ethereum.on('chainChanged', (chainId) => {
                window.log(`Chain changed: ${chainId}`, 'info');
                window.location.reload();
            });

            return this.ethAddress;
        } catch (error) {
            console.error('MetaMask connection error:', error);
            throw error;
        }
    }

    async connectPhantom() {
        if (!window.solana || !window.solana.isPhantom) {
            throw new Error('Phantom wallet not installed. Please install Phantom extension.');
        }

        try {
            const resp = await window.solana.connect();
            this.solAddress = resp.publicKey.toString();
            this.solana = window.solana;

            window.solana.on('accountChanged', (publicKey) => {
                if (publicKey) {
                    this.solAddress = publicKey.toString();
                    window.log(`Phantom account changed: ${this.solAddress}`, 'info');
                } else {
                    this.solAddress = null;
                    window.log('Phantom disconnected', 'warning');
                }
            });

            return this.solAddress;
        } catch (error) {
            console.error('Phantom connection error:', error);
            throw error;
        }
    }

    async disconnect() {
        if (this.solana) {
            try {
                await this.solana.disconnect();
            } catch (e) {
                // sometimes phantom may already be disconnected
            }
        }
        // MetaMask doesn't have a disconnect method; refreshing page is usual way
        this.ethAddress = null;
        this.solAddress = null;
        this.ethereum = null;
        this.solana = null;
        window.log('Wallets disconnected', 'info');
    }

    isConnected() {
        return !!(this.ethAddress && this.solAddress);
    }
}
