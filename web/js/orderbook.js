class OrderBookManager {
    constructor(ws, channelManager) {
        this.ws = ws;
        this.channelManager = channelManager;
        this.orders = [];
        this.bids = [];
        this.asks = [];

        // Register handlers
        this.ws.on('GetOrderBookResponse', (msg) => this.handleSnapshot(msg.snapshot));
        this.ws.on('OrderBookDelta', (msg) => this.handleDelta(msg));

        // Poll order book every 5 second when channel is active
        setInterval(() => {
            if (this.channelManager.channelId) {
                this.refreshOrderBook();
            }
        }, 5000);
    }

    async createOrder() {
        await initAssets();
        if (!ETH_ASSET) {
            window.log("ETH_ASSET not initialized yet.", "error");
            return;
        }
        if (!this.channelManager.channelId) {
            alert('No active channel. Please open a channel first.');
            return;
        }

        const side = document.getElementById('order-side').value;
        const baseAsset = document.getElementById('base-asset').value === "ETH" ? ETH_ASSET : SOL_ASSET;
        const quoteAsset = document.getElementById('quote-asset').value === "SOL" ? SOL_ASSET : ETH_ASSET;
        const price = document.getElementById('order-price').value;
        const amount = document.getElementById('order-amount').value;

        if (!price || !amount) {
            alert('Please enter price and amount');
            return;
        }

        const orderId = `order-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

        window.log(`Creating ${side} order...`, 'info');

        try {
            this.ws.request('CreateOrder', {
                order: {
                    id: orderId,
                    channelID: Array.from(this.channelManager.channelId),
                    makerIdx: this.channelManager.channelIdx,
                    side: side,
                    base: baseAsset,
                    quote: quoteAsset,
                    price: price,
                    amount: amount,
                    status: 'open',
                    createdAt: Math.floor(Date.now() / 1000)
                }
            });

            this.refreshOrderBook();

            window.log(`✅ Order created: ${orderId.substring(0, 16)}...`, 'success');
        } catch (err) {
            window.log(`Failed to create order: ${err.message}`, 'error');
        }
    }

    async cancelOrder(orderId) {
        window.log(`Canceling order ${orderId.substring(0, 16)}...`, 'info');

        try {
            const response = await this.ws.request('CancelOrder', {
                channelID: Array.from(this.channelManager.channelId),
                id: orderId
            });

            if (response.type === 'CancelOrderAck' && response.message.success) {
                window.log(`✅ Order canceled`, 'success');
                this.refreshOrderBook();
            } else {
                window.log(`❌ Cancel failed: ${response.message.reason}`, 'error');
            }
        } catch (err) {
            window.log(`Failed to cancel order: ${err.message}`, 'error');
        }
    }

    async acceptOrder(orderId) {
        const confirmed = confirm(
            'Accept this order?\n\n' +
            'This will:\n' +
            '1. Accept the order in the order book\n' +
            '2. Automatically settle trade via channel update\n\n' +
            'Continue?'
        );
        if (!confirmed) return;

        window.log(`Accepting order ${orderId.substring(0, 16)}...`, 'info');

        try {
            // 1. Accept the order on the order book
            this.ws.request('AcceptOrder', {
                channelID: Array.from(this.channelManager.channelId),
                id: orderId
            });
            window.log(`✅ Order accepted. Settling in channel...`, 'success');

            this.ws.request('CancelOrder', {
                channelID: Array.from(this.channelManager.channelId),
                id: orderId
            });

            // 2. Locate order
            const order = this.bids.concat(this.asks).find(o => o.id === orderId);
            if (!order) {
                window.log('Order not found locally. Refresh and retry.', 'error');
                return;
            }

            // 3. Get channel state
            const chState = this.channelManager.getState();
            if (!chState.channelId || chState.channelIdx === null) {
                window.log('No active channel found.', 'error');
                return;
            }

            await initAssets();
            if (!ETH_ASSET || !SOL_ASSET) {
                window.log("Assets not initialized.", "error");
                return;
            }

            const assets = [ETH_ASSET, SOL_ASSET]; // ETH index 0, SOL index 1

            const myIdx = this.channelManager.channelIdx; // always taker
            const peerIdx = 1 - myIdx;

            let bals = chState.balances ? [...chState.balances.my] : [];
            let peerBals = chState.balances ? [...chState.balances.peer] : [];

            const makerIdx = order.makerIdx;
            const takerIdx = 1 - makerIdx;

            // Ensure correct BigInt initialization
            bals = bals.map(x => BigInt(x));
            peerBals = peerBals.map(x => BigInt(x));

            // 4. Asset and amount calculations
            const baseAsset = order.base;
            const quoteAsset = order.quote;
            const baseSymbol = baseAsset.asset.assetType === 'Ethereum' ? 'ETH' : 'SOL';
            const quoteSymbol = quoteAsset.asset.assetType === 'Ethereum' ? 'ETH' : 'SOL';

            const baseDecimals = baseSymbol === 'ETH' ? 1e18 : 1e9;
            const quoteDecimals = quoteSymbol === 'ETH' ? 1e18 : 1e9;

            const baseAmount = BigInt(Math.round(parseFloat(order.amount) * baseDecimals));
            const quoteAmount = BigInt(Math.round(parseFloat(order.amount) * parseFloat(order.price) * quoteDecimals));

            const idxFor = sym => (sym === 'ETH' ? 0 : 1);
            const add = (arr, sym, amt) => { arr[idxFor(sym)] = arr[idxFor(sym)] + amt; };
            const sub = (arr, sym, amt) => { arr[idxFor(sym)] = arr[idxFor(sym)] - amt; };

            console.log(`Settling trade: ${order.side} ${order.amount} ${baseSymbol} @ ${order.price} ${quoteSymbol}`);
            console.log(`baseAmt=${baseAmount}, quoteAmt=${quoteAmount}`);
            console.log(`I am taker (acceptOrder caller). MakerIdx=${makerIdx}`);

            // 5. Mirror updates (taker perspective)
            if (order.side === 'bid') {
                // Maker buys base, pays quote
                // Taker sells base, receives quote

                // Taker (me)
                add(bals, quoteSymbol, quoteAmount); // receive quote
                sub(bals, baseSymbol, baseAmount);   // send base

                // Maker (peer)
                sub(peerBals, quoteSymbol, quoteAmount);
                add(peerBals, baseSymbol, baseAmount);
            } else if (order.side === 'ask') {
                // Maker sells base, receives quote
                // Taker buys base, pays quote

                // Taker (me)
                sub(bals, quoteSymbol, quoteAmount); // pay quote
                add(bals, baseSymbol, baseAmount);   // receive base

                // Maker (peer)
                add(peerBals, quoteSymbol, quoteAmount);
                sub(peerBals, baseSymbol, baseAmount);
            } else {
                window.log(`Invalid order side: ${order.side}`, 'error');
                return;
            }

            // Convert back to strings for serialization
            bals = bals.map(x => x.toString());
            peerBals = peerBals.map(x => x.toString());

            // 6. Submit update
            const updateRequest = {
                id: Array.from(this.channelManager.channelId),
                state: {
                    balance: bals,
                    peerBalance: peerBals,
                    assets: assets,
                    backends: [1, 6], // ETH idx=1, SOL idx=6 (adjust as per backend)
                    isFinal: false
                }
            };

            window.log('Submitting UpdateChannel: ' + JSON.stringify(updateRequest), 'info');
            this.ws.request('UpdateChannel', updateRequest);

            window.log('✅ Channel updated. Trade settled!', 'success');
            this.channelManager.refreshChannelInfo();

        } catch (err) {
            window.log(`Failed to accept order or update channel: ${err.message}`, 'error');
        }
    }

    async refreshOrderBook() {
        if (!this.channelManager.channelId) return;

        try {
            this.ws.request('GetOrderBook', {
                channelID: Array.from(this.channelManager.channelId),
                sinceSequence: 0
            });
        } catch (err) {
            // Silently fail on polling errors
            console.error('Order book refresh failed:', err);
        }
    }

    handleSnapshot(snapshot) {
        this.bids = snapshot.bids || [];
        this.asks = snapshot.asks || [];
        this.displayOrders();
    }

    handleDelta(delta) {
        // Apply delta to local order book
        if (delta.added) {
            delta.added.forEach(order => {
                if (order.side === 'bid') {
                    this.bids.push(order);
                } else {
                    this.asks.push(order);
                }
            });
        }

        if (delta.removed) {
            this.bids = this.bids.filter(o => !delta.removed.includes(o.id));
            this.asks = this.asks.filter(o => !delta.removed.includes(o.id));
        }

        if (delta.updated) {
            delta.updated.forEach(upd => {
                const bidIdx = this.bids.findIndex(o => o.id === upd.id);
                if (bidIdx >= 0) this.bids[bidIdx] = upd;

                const askIdx = this.asks.findIndex(o => o.id === upd.id);
                if (askIdx >= 0) this.asks[askIdx] = upd;
            });
        }

        this.displayOrders();
    }

    displayOrders() {
        this.displayOrderList('bids-list', this.bids, 'bid');
        this.displayOrderList('asks-list', this.asks, 'ask');
    }

    displayOrderList(elementId, orders, side) {
        const listEl = document.getElementById(elementId);

        if (orders.length === 0) {
            listEl.innerHTML = `<p class="empty-message">No ${side} orders</p>`;
            return;
        }

        // Sort: bids descending by price, asks ascending
        const sorted = [...orders].sort((a, b) => {
            const priceA = parseFloat(a.price);
            const priceB = parseFloat(b.price);
            return side === 'bid' ? priceB - priceA : priceA - priceB;
        });

        listEl.innerHTML = sorted.map(order => {
            const isMyOrder = order.makerIdx === this.channelManager.channelIdx;
            const shortId = order.id.substring(0, 12);

            // Map asset types to symbols
            const baseSymbol = order.base.asset.assetType === 'Ethereum' ? 'ETH' : 'SOL';
            const quoteSymbol = order.quote.asset.assetType === 'Ethereum' ? 'ETH' : 'SOL';

            // Display price clearly: quote per unit of base
            const priceDisplay = `${parseFloat(order.price).toFixed(6)} ${quoteSymbol} per ${baseSymbol}`;

            return `
            <div class="order-item ${side} ${isMyOrder ? 'my-order' : ''}">
                <div class="order-header">
                    <span class="order-badge ${side}">
                        ${side.toUpperCase()} ${isMyOrder ? '(YOU)' : ''}
                    </span>
                    <span class="order-id">${shortId}...</span>
                </div>
                <div class="order-details">
                    <strong>${baseSymbol} / ${quoteSymbol}</strong><br>
                    Price: ${priceDisplay}<br>
                    Amount: ${order.amount} ${baseSymbol}
                </div>
                <div class="order-actions">
                    ${isMyOrder ? `
                        <button class="danger-btn small-btn" onclick="client.cancelOrder('${order.id}')">
                            Cancel
                        </button>
                    ` : `
                        <button class="success-btn small-btn" onclick="client.acceptOrder('${order.id}')">
                            Accept & Trade
                        </button>
                    `}
                </div>
            </div>
        `;
        }).join('');
    }
}
