class ChannelManager {
    constructor(ws) {
        this.ws = ws;
        this.channelId = null;
        this.channelIdx = null;
        this.proposalId = null;

        // Register handlers for channel events
        this.ws.on('ChannelProposal', (msg) => this.handleChannelProposal(msg));
        this.ws.on('ChannelCreated', (msg) => this.handleChannelCreated(msg));
        this.ws.on('ChannelClosed', (msg) => this.handleChannelClosed(msg));
        this.ws.on('UpdateChannel', (msg) => this.handleUpdateProposal(msg));
        this.ws.on('FundingError', (msg) => this.handleFundingError(msg));
    }

    async openChannel() {
        const peerEth = document.getElementById('peer-eth-addr').value;
        const peerSol = document.getElementById('peer-sol-addr').value;
        const myEth = parseFloat(document.getElementById('my-eth').value);
        const mySol = parseFloat(document.getElementById('my-sol').value);
        const peerEthVal = parseFloat(document.getElementById('peer-eth').value);
        const peerSolVal = parseFloat(document.getElementById('peer-sol').value);
        const challenge = parseInt(document.getElementById('challenge-duration').value);

        if (!peerEth || !peerSol) {
            alert('Please enter peer addresses');
            return;
        }

        // Generate unique proposal ID
        this.proposalId = Array.from({ length: 32 }, () =>
            Math.floor(Math.random() * 256)
        );

        // Convert to base units (wei for ETH, lamports for SOL)
        const ethWei = (myEth * 1e18).toString();
        const solLamports = (mySol * 1e9).toString();
        const peerEthWei = (peerEthVal * 1e18).toString();
        const peerSolLamports = (peerSolVal * 1e9).toString();

        window.log('Proposing channel...', 'info');

        try {
            const response = await this.ws.request('OpenChannel', {
                proposalID: this.proposalId,
                peerAddressEth: peerEth,
                peerAddressSol: peerSol,
                challengeDuration: challenge.toString(),
                state: {
                    balances: [ethWei, solLamports],
                    peerBalances: [peerEthWei, peerSolLamports],
                    assets: [], // Populated by backend based on chain config
                    backends: [0, 1], // EthereumIndex=0, SolanaIndex=1
                    isFinal: false
                }
            });

            if (response.type === 'Success') {
                window.log('Channel proposal sent successfully', 'success');
            } else if (response.type === 'Error') {
                window.log(`Channel proposal failed: ${response.error}`, 'error');
            }
        } catch (err) {
            window.log(`Failed to propose channel: ${err.message}`, 'error');
        }
    }

    async closeChannel() {
        if (!this.channelId) {
            alert('No active channel');
            return;
        }

        const confirmed = confirm('Are you sure you want to close this channel? This will settle on-chain.');
        if (!confirmed) return;

        window.log('Closing channel...', 'info');

        try {
            const response = await this.ws.request('CloseChannel', {
                id: Array.from(this.channelId),
                forceClose: false
            });

            if (response.type === 'Success') {
                window.log('Channel close initiated', 'success');
            } else if (response.type === 'Error') {
                window.log(`Channel close failed: ${response.error}`, 'error');
            }
        } catch (err) {
            window.log(`Failed to close channel: ${err.message}`, 'error');
        }
    }

    async refreshChannelInfo() {
        if (!this.channelId) {
            window.log('No active channel', 'warning');
            return;
        }

        try {
            const response = await this.ws.request('GetChannelInfo', {
                id: Array.from(this.channelId)
            });

            if (response.type === 'ChannelInfo') {
                this.displayChannelInfo(response.message);
            } else if (response.type === 'Error') {
                window.log(`Failed to get channel info: ${response.error}`, 'error');
            }
        } catch (err) {
            window.log(`Failed to refresh channel: ${err.message}`, 'error');
        }
    }

    handleChannelProposal(msg) {
        window.log('📨 Channel proposal received', 'info');

        // Auto-accept for demo
        this.ws.request('ProposalResponse', {
            accepted: true,
            rejectReason: ''
        }).then(() => {
            window.log('✅ Channel proposal accepted', 'success');
        }).catch(err => {
            window.log(`Failed to accept proposal: ${err.message}`, 'error');
        });
    }

    handleChannelCreated(msg) {
        this.channelId = msg.id;
        this.channelIdx = msg.idx;

        const channelIdHex = Array.from(msg.id)
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');

        document.getElementById('channel-id').textContent =
            channelIdHex.substring(0, 16) + '...';
        document.getElementById('my-index').textContent = msg.idx;
        document.getElementById('channel-info').style.display = 'block';

        window.log(`🎉 Channel created! ID: ${channelIdHex.substring(0, 16)}...`, 'success');
        window.updateChannelStatus('Active', true);

        // Refresh channel info to get balances
        setTimeout(() => this.refreshChannelInfo(), 1000);
    }

    handleChannelClosed(msg) {
        const channelIdHex = Array.from(msg.id)
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');

        window.log(`Channel closed: ${channelIdHex.substring(0, 16)}...`, 'info');

        this.channelId = null;
        this.channelIdx = null;

        document.getElementById('channel-info').style.display = 'none';
        window.updateChannelStatus('No Channel', false);
    }

    handleUpdateProposal(msg) {
        window.log('📨 Channel update proposal received', 'info');

        // Auto-accept for demo
        this.ws.request('ProposalResponse', {
            accepted: true,
            rejectReason: ''
        }).then(() => {
            window.log('✅ Update accepted', 'success');
            setTimeout(() => this.refreshChannelInfo(), 500);
        }).catch(err => {
            window.log(`Failed to accept update: ${err.message}`, 'error');
        });
    }

    handleFundingError(msg) {
        window.log(`❌ Funding error: ${msg.error}`, 'error');
        alert(`Channel funding failed: ${msg.error}`);
    }

    displayChannelInfo(info) {
        const myBalsEl = document.getElementById('my-balances');
        const peerBalsEl = document.getElementById('peer-balances');
        const peerIdEl = document.getElementById('peer-id');

        if (info.peerAddressEth && info.peerAddressSol) {
            const ethShort = info.peerAddressEth.substring(0, 10) + '...';
            const solShort = info.peerAddressSol.substring(0, 10) + '...';
            peerIdEl.textContent = `ETH: ${ethShort}, SOL: ${solShort}`;
        }

        if (info.state && info.state.balances) {
            myBalsEl.innerHTML = info.state.balances.map((bal, i) => {
                const formatted = this.formatBalance(bal, i);
                return `<div>Asset ${i}: ${formatted}</div>`;
            }).join('');

            peerBalsEl.innerHTML = info.state.peerBalances.map((bal, i) => {
                const formatted = this.formatBalance(bal, i);
                return `<div>Asset ${i}: ${formatted}</div>`;
            }).join('');
        }
    }

    formatBalance(balance, assetIndex) {
        // Assume asset 0 is ETH (18 decimals), asset 1 is SOL (9 decimals)
        const decimals = assetIndex === 0 ? 18 : 9;
        const value = BigInt(balance) / BigInt(10 ** decimals);
        const remainder = BigInt(balance) % BigInt(10 ** decimals);
        const decimalPart = remainder.toString().padStart(decimals, '0').substring(0, 4);
        return `${value}.${decimalPart}`;
    }
}
