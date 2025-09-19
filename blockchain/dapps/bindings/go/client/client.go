// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/agntcy/dir/blockchain/dapps/bindings/go/agentstore"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	options *Options
	client  *ethclient.Client
	store   *agentstore.AgentStore
}

func New(opts ...Option) (*Client, error) {
	// construct options
	options := &Options{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	// Connect to node via JSON-RPC
	ethClient, err := ethclient.Dial(options.chainURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ethclient: %w", err)
	}

	// Connect to agent store
	store, err := agentstore.NewAgentStore(common.HexToAddress(options.agentStoreAddress), ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create agentstore: %w", err)
	}

	return &Client{
		options: options,
		client:  ethClient,
		store:   store,
	}, nil
}

func (c *Client) AgentStore() *agentstore.AgentStore {
	return c.store
}

func (c *Client) TxOpts() *bind.TransactOpts {
	return c.options.txOpts
}

func (c *Client) RawClient() *ethclient.Client {
	return c.client
}

type Options struct {
	chainURL          string
	agentStoreAddress string
	txOpts            *bind.TransactOpts
}

type Option func(options *Options) error

func WithTxWriter(chainID string, hexPrivKey string) Option {
	return func(options *Options) error {
		// Hex private key to ECDSA private key
		hexPrivKey = strings.TrimPrefix(hexPrivKey, "0x")

		privKey, err := crypto.HexToECDSA(hexPrivKey)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}

		// Parse chainID
		chainIDNum, err := strconv.ParseInt(chainID, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse chainID: %w", err)
		}

		// Create options to use when submitting payable transactions
		txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, big.NewInt(chainIDNum))
		if err != nil {
			return fmt.Errorf("failed to create transactor: %w", err)
		}

		// set options
		options.txOpts = txOpts

		return nil
	}
}

func WithChainURL(chainURL string) Option {
	return func(options *Options) error {
		options.chainURL = chainURL

		return nil
	}
}

func WithAgentStore(agentStoreAddress string) Option {
	return func(options *Options) error {
		agentStoreAddress = strings.TrimPrefix(agentStoreAddress, "0x")
		options.agentStoreAddress = agentStoreAddress

		return nil
	}
}
