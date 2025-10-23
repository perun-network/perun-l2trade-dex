class OrderBookManager {
    constructor(ws, channelManager) {
        this.ws = ws;
        this.channelManager = channelManager;
        this.orders = [];
        this.bids = [];
        this.asks = [];

        // Register handlers
        this.ws.on('OrderBookSnapshot', (msg) => this.handleSnapshot(msg));
        this.ws.on('OrderBookDelta', (msg) => this.handleDelta(msg));

        // Poll order book every 5 seconds when channel is active
        setInterval(() => {
            if (this.channelManager.channelId) {
                this.refreshOrderBook();
            }
        }, 5000);
    }

    async createOrder() {
        if (!this.channelManager.channelId) {
            alert('No active channel. Please open a channel first.');
            return;
        }

        const side = document.getElementById('order-side').value;
        const baseAsset = document.getElementById('base-asset').value;
        const quoteAsset = document.getElementById('quote-asset').value;
        const price = document.getElementById('order-price').value;
        const amount = document.getElementById('order-amount').value;

        if (!price || !amount) {
            alert('Please enter price and amount');
            return;
        }

        const orderId = `order-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

        window.log(`Creating ${side} order...`, 'info');

        try {
            const response = await this.ws.request('CreateOrder', {
                order: {
                    id: orderId,
                    channelID: Array.from(this.channelManager.channelId),
                    makerIdx: this.channelManager.channelIdx,
                    side: side,
                    baseAsset: { type: baseAsset },
                    quoteAsset: { type: quoteAsset },
                    price: price,
                    amount: amount,
                    status: 'open',
                    createdAt: Math.floor(Date.now() / 1000)
                }
            });

            if (response.type === 'CreateOrderAck' && response.message.accepted) {
                window.log(`✅ Order created: ${orderId.substring(0, 16)}...`, 'success');
                this.refreshOrderBook();
            } else {
                window.log(`❌ Order rejected: ${response.message.reason}`, 'error');
            }
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
            '2. You should then manually execute a channel update with the trade balances\n\n' +
            'Continue?'
        );

        if (!confirmed) return;

        window.log(`Accepting order ${orderId.substring(0, 16)}...`, 'info');

        try {
            const response = await this.ws.request('AcceptOrder', {
                channelID: Array.from(this.channelManager.channelId),
                id: orderId
            });

            if (response.type === 'AcceptOrderAck' && response.message.accepted) {
                window.log(`✅ Order accepted - now execute channel update to settle trade`, 'success');

                // TODO: Here you would trigger UpdateChannel with new balances
                // For now, just show a message
                alert(
                    'Order accepted!\n\n' +
                    'In a production app, this would trigger a channel update with:\n' +
                    '- Deduct base asset from seller\n' +
                    '- Add base asset to buyer\n' +
                    '- Deduct quote asset from buyer\n' +
                    '- Add quote asset to seller\n\n' +
                    'For this demo, you can manually refresh balances.'
                );

                this.refreshOrderBook();
            } else {
                window.log(`❌ Accept failed: ${response.message.reason}`, 'error');
            }
        } catch (err) {
            window.log(`Failed to accept order: ${err.message}`, 'error');
        }
    }

    async refreshOrderBook() {
        if (!this.channelManager.channelId) return;

        try {
            const response = await this.ws.request('GetOrderBook', {
                channelID: Array.from(this.channelManager.channelId),
                sinceSequence: 0
            });

            if (response.type === 'GetOrderBookResponse' && response.message.snapshot) {
                this.handleSnapshot(response.message.snapshot);
            }
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
            const baseAsset = order.baseAsset?.type || 'ASSET';
            const quoteAsset = order.quoteAsset?.type || 'ASSET';

            return `
                <div class="order-item ${side} ${isMyOrder ? 'my-order' : ''}">
                    <div class="order-header">
                        <span class="order-badge ${side}">
                            ${side.toUpperCase()} ${isMyOrder ? '(YOU)' : ''}
                        </span>
                        <span class="order-id">${shortId}...</span>
                    </div>
                    <div class="order-details">
                        <strong>${baseAsset} / ${quoteAsset}</strong><br>
                        Price: ${order.price} ${quoteAsset} per ${baseAsset}<br>
                        Amount: ${order.amount} (base units)
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
