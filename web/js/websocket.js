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

            console.log("Sending WebSocket message:", JSON.stringify(initMsg));
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

        if (data.type === 'Response') {
            this.handleResponse(data.message);
            return;
        }

        if (data.type == 'Request') {
            // Handle request messages
            this.handleRequest(data.message)
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

    async handleEthTxRequest(id, data) {
        window.log('🦊 MetaMask transaction requested', 'info');

        try {
            const rawSignedTx = await this.walletManager.signEthereumTransaction(data.transaction);
            console.log("Signed Ethereum transaction:", rawSignedTx);
            const response = {
                type: 'Response',
                message: {
                    id: id,
                    message: {
                        type: 'SendETHTxResponse',
                        message: {
                            transaction: rawSignedTx
                        }
                    }
                }
            };

            this.ws.send(JSON.stringify(response));
            window.log(`✅ Ethereum tx sent: ${rawSignedTx.substring(0, 16)}...`, 'success');
        } catch (error) {
            window.log(`❌ Ethereum tx failed: ${error.message}`, 'error');
        }
    }

    async handleSolTxRequest(id, message) {
        window.log('👻 Phantom transaction requested', 'info');

        try {
            // The tx field may be under data.message.transaction or similar
            const base64Tx = message.transaction;
            const signedTxBase64 = await this.walletManager.signSolanaTransaction(base64Tx);
            console.log("Signed Solana transaction:", signedTxBase64);
            const response = {
                type: 'Response',
                message: {
                    id: id,
                    message: {
                        type: 'SendSolTxResponse',
                        message: {
                            transaction: signedTxBase64
                        }
                    }
                }
            };

            this.ws.send(JSON.stringify(response));
            window.log(`✅ Solana tx sent: ${signedTxBase64.substring(0, 16)}...`, 'success');
        } catch (error) {
            window.log(`❌ Solana tx failed: ${error.message}`, 'error');
        }
    }

    async handleResponse(data) {
        console.log("Handling response:", data);
        if (data.message.type === 'ChannelInfo') {
            console.log("Channel info received:", data.message);
            this.messageHandlers.get(data.message.type)(data.message?.message);
        }


        if (data.message.type === 'GetOrderBookResponse') {
            console.log("Order book response received:", data.message);
            this.messageHandlers.get(data.message.type)(data.message?.message);
        }
    }

    async handleRequest(data) {
        console.log("Handling request:", data);
        if (data.message.type === 'ChannelProposal') {
            // Handle channel proposal
            try {
                const response = {
                    type: 'Response',
                    message: {
                        id: data.id,
                        message: {
                            type: 'ProposalResponse',
                            message: {
                                accepted: true,
                                rejectReason: ''
                            }
                        }
                    }
                };
                this.ws.send(JSON.stringify(response));
            } catch (error) {
                console.error("Error handling channel proposal:", error);
            }
        }

        if (data.message.type === 'UpdateChannel') {
            // Handle channel update
            try {
                const response = {
                    type: 'Response',
                    message: {
                        id: data.id,
                        message: {
                            type: 'ProposalResponse',
                            message: {
                                accepted: true,
                                rejectReason: ''
                            }
                        }
                    }
                };
                this.ws.send(JSON.stringify(response));
            } catch (error) {
                console.error("Error handling channel update:", error);
            }
        }

        if (data.message.type === 'SendSolTx') {
            this.handleSolTxRequest(data.id, data.message?.message);
        }

        if (data.message.type === 'SendETHTx') {
            this.handleEthTxRequest(data.id, data.message?.message);
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
                type: 'Request',
                message: {
                    id: id,
                    message: {
                        type: messageType,
                        message: messageData
                    }
                }
            };

            this.pendingRequests.set(id, resolve);

            console.log("Sending WebSocket message:", JSON.stringify(request));
            this.ws.send(JSON.stringify(request));
        });
    }

    on(messageType, handler) {
        this.messageHandlers.set(messageType, handler);
    }


}