class WSConnection {
    constructor(userName, walletManager) {
        this.userName = userName;
        this.ws = null;
        this.messageId = 0;
        this.pendingRequests = new Map();
        this.messageHandlers = new Map();
        this.connected = false;
        this.walletManager = walletManager;
    }

    connect(onConnect, onError) {
        const wsUrl = `ws://${window.location.host}/connect`;
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
            window.log('WebSocket connected');

            // Send CrossContractInitialize with wallet addresses
            const initMsg = {
                type: 'CrossContractInitialize',
                message: {
                    ethClientAddress: this.walletManager.ethAddress,
                    solClientAddress: this.walletManager.solAddress,
                    egoisticClient: false
                }
            };

            this.ws.send(JSON.stringify(initMsg));
        };

        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleMessage(data);
        };

        this.ws.onerror = (error) => {
            window.log(`WebSocket error: ${error}`, 'error');
            if (onError) onError(error);
        };

        this.ws.onclose = () => {
            window.log('WebSocket disconnected');
            this.connected = false;
            client.updateConnectionStatus('Disconnected', false);
        };
    }

    handleMessage(data) {
        // Check if it's a response to a request
        if (data.id !== undefined && this.pendingRequests.has(data.id)) {
            const resolver = this.pendingRequests.get(data.id);
            this.pendingRequests.delete(data.id);
            resolver(data.message);
            return;
        }

        // Check for Initialized message
        if (data.type === 'Initialized') {
            this.connected = true;
            client.updateConnectionStatus('Connected', true);
            window.log(`Initialized with L2 address: ${data.message.l2Address}`, 'success');
            return;
        }

        // Handle signing requests
        if (data.type === 'SignETHData' || data.message?.type === 'SignETHData') {
            this.handleEthSignRequest(data.message?.message || data.message);
            return;
        }

        if (data.type === 'SignSolData' || data.message?.type === 'SignSolData') {
            this.handleSolSignRequest(data.message?.message || data.message);
            return;
        }

        if (data.type === 'SendETHTx' || data.message?.type === 'SendETHTx') {
            this.handleEthTxRequest(data.message?.message || data.message);
            return;
        }

        if (data.type === 'SendSolTx' || data.message?.type === 'SendSolTx') {
            this.handleSolTxRequest(data.message?.message || data.message);
            return;
        }

        // Handle incoming messages by type
        const msgType = data.type || (data.message && data.message.type);
        if (msgType && this.messageHandlers.has(msgType)) {
            this.messageHandlers.get(msgType)(data.message?.message || data.message);
        } else {
            window.log(`Received: ${msgType || 'unknown'}`);
        }
    }

    async handleEthSignRequest(data) {
        window.log('🦊 MetaMask signature requested', 'info');

        try {
            const hexData = Array.isArray(data.data)
                ? '0x' + data.data.map(b => b.toString(16).padStart(2, '0')).join('')
                : data.data;

            const signature = await this.walletManager.signEthereumMessage(hexData);

            // Convert hex signature to byte array
            const sigBytes = [];
            for (let i = 2; i < signature.length; i += 2) {
                sigBytes.push(parseInt(signature.substr(i, 2), 16));
            }

            const response = {
                type: 'SignResponse',
                message: {
                    signature: sigBytes
                }
            };

            this.ws.send(JSON.stringify(response));
            window.log('✅ Ethereum signature sent', 'success');
        } catch (error) {
            window.log(`❌ Ethereum signing failed: ${error.message}`, 'error');
        }
    }

    async handleSolSignRequest(data) {
        window.log('👻 Phantom signature requested', 'info');

        try {
            const messageStr = Array.isArray(data.data)
                ? new TextDecoder().decode(new Uint8Array(data.data))
                : data.data;

            const signature = await this.walletManager.signSolanaMessage(messageStr);

            const response = {
                type: 'SignResponse',
                message: {
                    signature: Array.from(signature)
                }
            };

            this.ws.send(JSON.stringify(response));
            window.log('✅ Solana signature sent', 'success');
        } catch (error) {
            window.log(`❌ Solana signing failed: ${error.message}`, 'error');
        }
    }

    async handleEthTxRequest(data) {
        window.log('🦊 MetaMask transaction requested', 'info');

        try {
            const txHash = await this.walletManager.signEthereumTransaction(data.transaction);

            const response = {
                type: 'SendETHTxResponse',
                message: {
                    transaction: data.transaction
                }
            };

            this.ws.send(JSON.stringify(response));
            window.log(`✅ Ethereum tx sent: ${txHash.substring(0, 16)}...`, 'success');
        } catch (error) {
            window.log(`❌ Ethereum tx failed: ${error.message}`, 'error');
        }
    }

    async handleSolTxRequest(data) {
        window.log('👻 Phantom transaction requested', 'info');

        try {
            const txBuffer = Uint8Array.from(atob(data.tx), c => c.charCodeAt(0));
            const signature = await this.walletManager.signSolanaTransaction(txBuffer);

            const response = {
                type: 'SendSolTxResponse',
                message: {
                    tx: data.tx
                }
            };

            this.ws.send(JSON.stringify(response));
            window.log(`✅ Solana tx sent: ${signature.substring(0, 16)}...`, 'success');
        } catch (error) {
            window.log(`❌ Solana tx failed: ${error.message}`, 'error');
        }
    }

    request(messageType, messageData) {
        return new Promise((resolve, reject) => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
                reject(new Error('WebSocket not connected'));
                return;
            }

            const id = this.messageId++;
            const request = {
                id: id,
                message: {
                    type: messageType,
                    message: messageData
                }
            };

            this.pendingRequests.set(id, resolve);
            this.ws.send(JSON.stringify(request));

            // Timeout after 30 seconds
            setTimeout(() => {
                if (this.pendingRequests.has(id)) {
                    this.pendingRequests.delete(id);
                    reject(new Error('Request timeout'));
                }
            }, 30000);
        });
    }

    on(messageType, handler) {
        this.messageHandlers.set(messageType, handler);
    }
}
