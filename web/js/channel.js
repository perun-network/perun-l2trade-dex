
async function loadEthAssetHolderFromTxt(path = "./frontend_config.txt") {
    const resp = await fetch(path, { cache: "no-cache" }); // avoid stale cache
    if (!resp.ok) throw new Error("Failed to load frontend_config.txt");
    const raw = await resp.text();                          // read as text
    const addr = raw.replace(/^\uFEFF/, "").trim();         // strip BOM + trim
    const cleaned = addr.replace(/["'\s]/g, "");            // remove quotes/spaces

    return cleaned
}

// Globals (or pass through your app init)
let ETH_ASSET = null;
const SOL_ASSET = {
    asset: {
        assetType: "Solana",
        mint: ""
    }
};

async function initAssets() {
    const assetHolder = await loadEthAssetHolderFromTxt();  // e.g., "0x5C23d...728A"
    ETH_ASSET = {
        asset: {
            assetType: "Ethereum",
            assetHolder,              // from file
            chainID: "1337"           // set this from your chain config if dynamic
        }
    };
}

class ChannelManager {
    constructor(ws) {
        this.ws = ws;
        this.channelId = null;
        this.channelIdx = null;
        this.proposalId = null;
        // Internal state
        this.state = {
            status: 'No Channel',            // 'No Channel' | 'Proposing' | 'Active' | 'Closing' | 'Closed' | 'Error'
            channelId: null,                 // Uint8Array or null
            channelIdx: null,                // number or null
            peer: { eth: null, sol: null },  // strings
            balances: { my: [], peer: [] },  // arrays of string base units
        };

        // Register handlers for channel events
        this.ws.on('ChannelProposal', (msg) => this.handleChannelProposal(msg));
        this.ws.on('ChannelCreated', (msg) => this.handleChannelCreated(msg));
        this.ws.on('ChannelClosed', (msg) => this.handleChannelClosed(msg));
        this.ws.on('UpdateChannel', (msg) => this.handleUpdateProposal(msg));
        this.ws.on('FundingError', (msg) => this.handleFundingError(msg));
        this.ws.on('ChannelInfo', (msg) => this.displayChannelInfo(msg)); // detailed info


        // Poll order book every 5 seconds when channel is active
        setInterval(() => {
            this.refreshChannelInfo();
        }, 5000);
    }

    getState() { return JSON.parse(JSON.stringify(this.state)); } // clone to avoid external mutation
    getStatus() { return this.state.status; }
    getChannelId() { return this.state.channelId; }
    getChannelIdx() { return this.state.channelIdx; }
    getPeer() { return { ...this.state.peer }; }
    getBalances() { return { my: [...this.state.balances.my], peer: [...this.state.balances.peer] }; }

    setState(patch) {
        this.state = { ...this.state, ...patch };
    }

    updateChannelStatusPill() {
        const el = document.getElementById('channel-status');
        if (!el) return;
        const active = this.state.status === 'Active';
        el.textContent = active ? 'Active' : this.state.status || 'No Channel';
        if (active) el.classList.add('connected'); else el.classList.remove('connected');
    }

    async openChannel() {
        await initAssets();
        if (!ETH_ASSET) {
            return;
        }
        const assets = [ETH_ASSET, SOL_ASSET]; // plain objects, no wrapper

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

        this.setState({ status: 'Proposing' });
        this.updateChannelStatusPill();
        try {
            this.ws.request('OpenChannel', {
                proposalID: this.proposalId,
                peerAddressEth: peerEth,
                peerAddressSol: peerSol,
                challengeDuration: challenge.toString(),
                state: {
                    balance: [ethWei, solLamports],
                    peerBalance: [peerEthWei, peerSolLamports],
                    assets: assets, // Populated by backend based on chain config
                    backends: [1, 6], // EthereumIndex=1, SolanaIndex=6
                    isFinal: false
                }
            });
            // Immediate UX feedback; actual acceptance/creation arrives via events
            window.log('Channel proposal sent', 'success');
        } catch (err) {
            this.setState({ status: 'Error' });
            this.updateChannelStatusPill();

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
            this.ws.request('CloseChannel', {
                id: Array.from(this.channelId),
                forceClose: false
            });
            this.setState({ status: 'Closing' });
            this.updateChannelStatusPill();
            document.getElementById('channel-info').style.display = 'none';
        } catch (err) {
            window.log(`Failed to close channel: ${err.message}`, 'error');
        }
    }

    async refreshChannelInfo() {
        if (!this.channelId) {
            return;
        }

        try {
            this.ws.request('GetChannelInfo', {
                id: Array.from(this.channelId)
            });
        } catch (err) {
            window.log(`Failed to refresh channel: ${err.message}`, 'error');
        }
    }

    handleChannelProposal(msg) {

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
        this.setState({ status: 'Active', channelId: msg.id, channelIdx: msg.idx });
        this.updateChannelStatusPill();
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

        this.setState({ status: 'Closed', channelId: null, channelIdx: null, balances: { my: [], peer: [] } });
        this.updateChannelStatusPill();
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
        this.setState({ status: 'Error' });
        this.updateChannelStatusPill();
        alert(`Channel funding failed: ${msg.error}`);
    }

    displayChannelInfo(info) {
        const myBalsEl = document.getElementById('my-balances');
        const peerBalsEl = document.getElementById('peer-balances');
        const peerIdEl = document.getElementById('peer-id');

        this.setState({ peer: { eth: info.peerAddressEth, sol: info.peerAddressSol } });
        if (info.peerAddressEth && info.peerAddressSol) {
            const ethShort = info.peerAddressEth.substring(0, 10) + '...';
            const solShort = info.peerAddressSol.substring(0, 10) + '...';
            peerIdEl.textContent = `ETH: ${ethShort}, SOL: ${solShort}`;
        }

        const assetNames = ['ETH', 'SOL']; // map indices to symbols

        if (info.state && info.state.balance) {
            myBalsEl.innerHTML = info.state.balance.map((bal, i) => {
                const formatted = this.formatBalance(bal, i);
                return `<div>${assetNames[i]}: ${formatted}</div>`;
            }).join('');

            peerBalsEl.innerHTML = info.state.peerBalance.map((bal, i) => {
                const formatted = this.formatBalance(bal, i);
                return `<div>${assetNames[i]}: ${formatted}</div>`;
            }).join('');
        }

        // persist balances in state
        this.setState({ balances: { my: info.state.balance, peer: info.state.peerBalance } });
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
