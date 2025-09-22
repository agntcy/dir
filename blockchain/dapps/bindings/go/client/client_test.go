// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/agntcy/dir/blockchain/dapps/bindings/go/agentstore"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	testChainURL          = "http://localhost:8545"
	testAgentStoreAddress = "0x5fbdb2315678afecb367f032d93f642f64180aa3" // Replace with deployed contract address
	testPrivKey           = "8166f546bab6da521a8369cab06c5d2b9e46670292d85c875ee9ec20e84ffb61"
)

func setupTestClient(t *testing.T) *Client {
	t.Helper()

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(testPrivKey, "0x"))
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, big.NewInt(123321))
	if err != nil {
		t.Fatalf("failed to create transactor: %v", err)
	}

	client, err := New(
		WithChainURL(testChainURL),
		WithAgentStore(testAgentStoreAddress),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	client.options.txOpts = txOpts

	return client
}

func TestAgentStore_Get(t *testing.T) {
	client := setupTestClient(t)
	callOpts := &bind.CallOpts{Context: context.Background()}

	_, err := client.AgentStore().Get(callOpts, "test-agent-id")
	if err != nil {
		t.Logf("expected error or empty result: %v", err)
	}
}

func TestAgentStore_Total(t *testing.T) {
	client := setupTestClient(t)
	callOpts := &bind.CallOpts{Context: context.Background()}

	total, err := client.AgentStore().Total(callOpts)
	if err != nil {
		t.Fatalf("failed to call Total: %v", err)
	}

	t.Logf("Total agents: %v", total)
}

func TestAgentStore_Add(t *testing.T) {
	client := setupTestClient(t)

	agent := agentstore.Agent{
		Id:        "test-agent-id",
		Signature: "test-signature",
		Owner:     client.TxOpts().From,
	}

	_, err := client.AgentStore().Add(client.TxOpts(), agent)
	if err != nil {
		t.Logf("expected error if contract is not deployed or tx fails: %v", err)
	}
}

func TestClient_TxOpts(t *testing.T) {
	client := setupTestClient(t)

	txOpts := client.TxOpts()
	if txOpts == nil {
		t.Fatal("txOpts should not be nil")
	}
}

func TestClient_RawClient(t *testing.T) {
	client := setupTestClient(t)

	if client.RawClient() == nil {
		t.Fatal("RawClient should not be nil")
	}
}
