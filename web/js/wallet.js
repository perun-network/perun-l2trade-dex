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

    // ============== ETHEREUM: SIGN MESSAGE ==============

    // data can be either:
    // - string: human-readable text (will be hex-encoded and signed with personal_sign)
    // - 0x-prefixed hex string for binary payload
    async signEthereumMessage(data) {
        if (!this.ethereum || !this.ethAddress) {
            throw new Error('MetaMask not connected');
        }

        const isHex = typeof data === 'string' && data.startsWith('0x');
        const toHexUtf8 = (s) => {
            // hex-encode UTF-8 string for personal_sign
            const enc = new TextEncoder();
            const bytes = enc.encode(s);
            return '0x' + Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
        };

        // Use personal_sign for safety and UX; MetaMask recommends this for off-chain auth
        // If data is hex bytes already, keep it; otherwise encode the text to hex.
        const msgHex = isHex ? data : toHexUtf8(String(data));

        // personal_sign params: [data, address]
        const signature = await this.ethereum.request({
            method: 'personal_sign',
            params: [msgHex, this.ethAddress],
        });
        return signature; // 0x... signature hex
    }

    // ============== ETHEREUM: SIGN/SEND TRANSACTION ==============

    // tx must be an object compatible with eth_sendTransaction:
    // { to, value, data, gas, gasPrice, maxFeePerGas, maxPriorityFeePerGas, nonce, chainId }
    // values should be 0x hex strings where required; value/data can be "0x0" or omitted.
    async signEthereumTransaction(tx) {
        if (!this.ethereum || !this.ethAddress) {
            throw new Error('MetaMask not connected');
        }
        // Ensure from is set
        const txReq = { from: this.ethAddress, ...tx };

        // Send request to MetaMask; it will prompt the user and return a tx hash
        const txHash = await this.ethereum.request({
            method: 'eth_sendTransaction',
            params: [txReq],
        });
        console.log("Ethereum transaction sent, hash:", txHash);
        return txHash; // 0x... hash
    }

    // ============== SOLANA: SIGN MESSAGE ==============

    // data can be:
    // - Uint8Array or Array<number>
    // - string (utf8), which will be encoded before signing
    async signSolanaMessage(data) {
        if (!this.solana || !this.solAddress) {
            throw new Error('Phantom not connected');
        }

        let bytes;
        if (data instanceof Uint8Array) {
            bytes = data;
        } else if (Array.isArray(data)) {
            bytes = new Uint8Array(data);
        } else if (typeof data === 'string') {
            const enc = new TextEncoder();
            bytes = enc.encode(data);
        } else {
            throw new Error('Unsupported data type for Solana signMessage');
        }

        // Phantom returns { signature: Uint8Array, publicKey }
        const { signature } = await this.solana.signMessage(bytes, 'utf8');
        // Return a plain array for JSON-friendly transport
        return Array.from(signature);
    }

    // ============== SOLANA: SIGN TRANSACTION ==============
    async signSolanaTransaction(base64Tx) {
        if (!this.solana || !this.solAddress) {
            throw new Error('Phantom not connected');
        }

        // Decode base64 to Uint8Array
        const rawTx = Uint8Array.from(atob(base64Tx), c => c.charCodeAt(0));
        // Use VersionedTransaction for versioned txs; fallback to Transaction if needed
        const { VersionedTransaction, Transaction } = window.solanaWeb3 || {};
        if (!VersionedTransaction) throw new Error('web3.js not loaded');
        let tx;
        try {
            tx = VersionedTransaction.deserialize(rawTx);
        } catch (e) {
            // fallback to legacy
            tx = Transaction.from(rawTx);
        }

        const signedTx = await this.solana.signTransaction(tx);
        // Serialize to base64 string for transport
        let signedBytes;
        if (signedTx.serialize) {
            // VersionedTransaction (v1.75+)
            signedBytes = signedTx.serialize();
        } else {
            // fallback for anything legacy
            signedBytes = signedTx;
        }
        const signedTxBase64 = btoa(String.fromCharCode(...signedBytes));
        return signedTxBase64;
    }
}

window.WalletManager = WalletManager;
