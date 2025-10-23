package client

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"

	"github.com/perun-network/perun-dex-websocket/internal/message"
)

// Registry is a registry of clients.
type Registry struct {
	m           map[string]*Client
	l2Addresses map[string]common.Address
	mtx         sync.RWMutex
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{
		m:           make(map[string]*Client),
		l2Addresses: make(map[string]common.Address),
		mtx:         sync.RWMutex{},
	}
}

// Register creates and registers a new client at the given address. Fails if
// another client is already registered at the specified address.
func (r *Registry) Register(eaddr common.Address, saddr string, conn *message.Connection, cfg Config) (*Client, error) {
	sk, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("cannot generate private key: %w", err)
	}
	publicKey := sk.PublicKey

	// Hash the public key bytes with Keccak-256 (Ethereum address is last 20 bytes of the hash)
	address := crypto.PubkeyToAddress(publicKey)
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if _, ok := r.m[address.String()]; ok {
		return nil, fmt.Errorf("client with same l2 address already registered")
	}
	zeroAddress := common.Address{}
	if eaddr != zeroAddress {
		if _, ok := r.l2Addresses[eaddr.String()]; ok {
			log.Println("Client with same ethereum address already registered", eaddr.String(), zeroAddress)
			return nil, fmt.Errorf("client with same ethereum address already registered")
		}
	}
	if _, ok := r.l2Addresses[saddr]; ok && saddr != "" {
		return nil, fmt.Errorf("client with same solana address already registered")
	}

	c, err := NewClient(conn, sk, eaddr, saddr, cfg, r)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	r.m[address.String()] = c
	r.l2Addresses[eaddr.String()] = address
	if saddr != "" {
		r.l2Addresses[saddr] = address
	}
	return c, nil
}

// Remove removes the client at the given address from the registry.
func (r *Registry) Remove(a string) {
	l2, _ := r.getL2Address(a)
	r.mtx.Lock()
	defer r.mtx.Unlock()
	delete(r.l2Addresses, a)
	delete(r.m, a)
	delete(r.m, l2.String())
}

// Get returns the client at the given address.
func (r *Registry) Get(a string) (*Client, bool) {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	l2, ok := r.getL2Address(a)
	if !ok {
		c, ok := r.m[a]
		return c, ok
	}
	c, ok := r.m[l2.String()]
	return c, ok
}

func (r *Registry) getL2Address(a string) (common.Address, bool) {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	c, ok := r.l2Addresses[a]
	return c, ok
}
