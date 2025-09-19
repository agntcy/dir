// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0
// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package agentstore

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// Agent is an auto generated low-level Go binding around an user-defined struct.
type Agent struct {
	Id        string
	Signature string
	Owner     common.Address
}

// AgentStoreMetaData contains all meta data concerning the AgentStore contract.
var AgentStoreMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"id\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"signature\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"indexed\":false,\"internalType\":\"structAgent\",\"name\":\"agent\",\"type\":\"tuple\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"Added\",\"type\":\"event\"},{\"inputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"id\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"signature\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"internalType\":\"structAgent\",\"name\":\"req\",\"type\":\"tuple\"}],\"name\":\"add\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"agent_id\",\"type\":\"string\"}],\"name\":\"get\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"id\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"signature\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"internalType\":\"structAgent\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"total\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// AgentStoreABI is the input ABI used to generate the binding from.
// Deprecated: Use AgentStoreMetaData.ABI instead.
var AgentStoreABI = AgentStoreMetaData.ABI

// AgentStore is an auto generated Go binding around an Ethereum contract.
type AgentStore struct {
	AgentStoreCaller     // Read-only binding to the contract
	AgentStoreTransactor // Write-only binding to the contract
	AgentStoreFilterer   // Log filterer for contract events
}

// AgentStoreCaller is an auto generated read-only Go binding around an Ethereum contract.
type AgentStoreCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AgentStoreTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AgentStoreTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AgentStoreFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AgentStoreFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AgentStoreSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AgentStoreSession struct {
	Contract     *AgentStore       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AgentStoreCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AgentStoreCallerSession struct {
	Contract *AgentStoreCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// AgentStoreTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AgentStoreTransactorSession struct {
	Contract     *AgentStoreTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// AgentStoreRaw is an auto generated low-level Go binding around an Ethereum contract.
type AgentStoreRaw struct {
	Contract *AgentStore // Generic contract binding to access the raw methods on
}

// AgentStoreCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AgentStoreCallerRaw struct {
	Contract *AgentStoreCaller // Generic read-only contract binding to access the raw methods on
}

// AgentStoreTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AgentStoreTransactorRaw struct {
	Contract *AgentStoreTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAgentStore creates a new instance of AgentStore, bound to a specific deployed contract.
func NewAgentStore(address common.Address, backend bind.ContractBackend) (*AgentStore, error) {
	contract, err := bindAgentStore(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AgentStore{AgentStoreCaller: AgentStoreCaller{contract: contract}, AgentStoreTransactor: AgentStoreTransactor{contract: contract}, AgentStoreFilterer: AgentStoreFilterer{contract: contract}}, nil
}

// NewAgentStoreCaller creates a new read-only instance of AgentStore, bound to a specific deployed contract.
func NewAgentStoreCaller(address common.Address, caller bind.ContractCaller) (*AgentStoreCaller, error) {
	contract, err := bindAgentStore(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AgentStoreCaller{contract: contract}, nil
}

// NewAgentStoreTransactor creates a new write-only instance of AgentStore, bound to a specific deployed contract.
func NewAgentStoreTransactor(address common.Address, transactor bind.ContractTransactor) (*AgentStoreTransactor, error) {
	contract, err := bindAgentStore(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AgentStoreTransactor{contract: contract}, nil
}

// NewAgentStoreFilterer creates a new log filterer instance of AgentStore, bound to a specific deployed contract.
func NewAgentStoreFilterer(address common.Address, filterer bind.ContractFilterer) (*AgentStoreFilterer, error) {
	contract, err := bindAgentStore(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AgentStoreFilterer{contract: contract}, nil
}

// bindAgentStore binds a generic wrapper to an already deployed contract.
func bindAgentStore(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AgentStoreMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AgentStore *AgentStoreRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AgentStore.Contract.AgentStoreCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AgentStore *AgentStoreRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentStore.Contract.AgentStoreTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AgentStore *AgentStoreRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AgentStore.Contract.AgentStoreTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AgentStore *AgentStoreCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AgentStore.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AgentStore *AgentStoreTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AgentStore.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AgentStore *AgentStoreTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AgentStore.Contract.contract.Transact(opts, method, params...)
}

// Get is a free data retrieval call binding the contract method 0x693ec85e.
//
// Solidity: function get(string agent_id) view returns((string,string,address))
func (_AgentStore *AgentStoreCaller) Get(opts *bind.CallOpts, agent_id string) (Agent, error) {
	var out []interface{}
	err := _AgentStore.contract.Call(opts, &out, "get", agent_id)

	if err != nil {
		return *new(Agent), err
	}

	out0 := *abi.ConvertType(out[0], new(Agent)).(*Agent)

	return out0, err

}

// Get is a free data retrieval call binding the contract method 0x693ec85e.
//
// Solidity: function get(string agent_id) view returns((string,string,address))
func (_AgentStore *AgentStoreSession) Get(agent_id string) (Agent, error) {
	return _AgentStore.Contract.Get(&_AgentStore.CallOpts, agent_id)
}

// Get is a free data retrieval call binding the contract method 0x693ec85e.
//
// Solidity: function get(string agent_id) view returns((string,string,address))
func (_AgentStore *AgentStoreCallerSession) Get(agent_id string) (Agent, error) {
	return _AgentStore.Contract.Get(&_AgentStore.CallOpts, agent_id)
}

// Total is a free data retrieval call binding the contract method 0x2ddbd13a.
//
// Solidity: function total() view returns(uint256)
func (_AgentStore *AgentStoreCaller) Total(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AgentStore.contract.Call(opts, &out, "total")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Total is a free data retrieval call binding the contract method 0x2ddbd13a.
//
// Solidity: function total() view returns(uint256)
func (_AgentStore *AgentStoreSession) Total() (*big.Int, error) {
	return _AgentStore.Contract.Total(&_AgentStore.CallOpts)
}

// Total is a free data retrieval call binding the contract method 0x2ddbd13a.
//
// Solidity: function total() view returns(uint256)
func (_AgentStore *AgentStoreCallerSession) Total() (*big.Int, error) {
	return _AgentStore.Contract.Total(&_AgentStore.CallOpts)
}

// Add is a paid mutator transaction binding the contract method 0xc6ea2eed.
//
// Solidity: function add((string,string,address) req) returns()
func (_AgentStore *AgentStoreTransactor) Add(opts *bind.TransactOpts, req Agent) (*types.Transaction, error) {
	return _AgentStore.contract.Transact(opts, "add", req)
}

// Add is a paid mutator transaction binding the contract method 0xc6ea2eed.
//
// Solidity: function add((string,string,address) req) returns()
func (_AgentStore *AgentStoreSession) Add(req Agent) (*types.Transaction, error) {
	return _AgentStore.Contract.Add(&_AgentStore.TransactOpts, req)
}

// Add is a paid mutator transaction binding the contract method 0xc6ea2eed.
//
// Solidity: function add((string,string,address) req) returns()
func (_AgentStore *AgentStoreTransactorSession) Add(req Agent) (*types.Transaction, error) {
	return _AgentStore.Contract.Add(&_AgentStore.TransactOpts, req)
}

// AgentStoreAddedIterator is returned from FilterAdded and is used to iterate over the raw logs and unpacked data for Added events raised by the AgentStore contract.
type AgentStoreAddedIterator struct {
	Event *AgentStoreAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AgentStoreAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AgentStoreAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AgentStoreAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AgentStoreAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AgentStoreAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AgentStoreAdded represents a Added event raised by the AgentStore contract.
type AgentStoreAdded struct {
	Agent     Agent
	Timestamp *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAdded is a free log retrieval operation binding the contract event 0x428d7e9655012a7b6cd787cb9b20534c8246705a9fe14d6dacdd3eb51d9f3088.
//
// Solidity: event Added((string,string,address) agent, uint256 timestamp)
func (_AgentStore *AgentStoreFilterer) FilterAdded(opts *bind.FilterOpts) (*AgentStoreAddedIterator, error) {

	logs, sub, err := _AgentStore.contract.FilterLogs(opts, "Added")
	if err != nil {
		return nil, err
	}
	return &AgentStoreAddedIterator{contract: _AgentStore.contract, event: "Added", logs: logs, sub: sub}, nil
}

// WatchAdded is a free log subscription operation binding the contract event 0x428d7e9655012a7b6cd787cb9b20534c8246705a9fe14d6dacdd3eb51d9f3088.
//
// Solidity: event Added((string,string,address) agent, uint256 timestamp)
func (_AgentStore *AgentStoreFilterer) WatchAdded(opts *bind.WatchOpts, sink chan<- *AgentStoreAdded) (event.Subscription, error) {

	logs, sub, err := _AgentStore.contract.WatchLogs(opts, "Added")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AgentStoreAdded)
				if err := _AgentStore.contract.UnpackLog(event, "Added", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAdded is a log parse operation binding the contract event 0x428d7e9655012a7b6cd787cb9b20534c8246705a9fe14d6dacdd3eb51d9f3088.
//
// Solidity: event Added((string,string,address) agent, uint256 timestamp)
func (_AgentStore *AgentStoreFilterer) ParseAdded(log types.Log) (*AgentStoreAdded, error) {
	event := new(AgentStoreAdded)
	if err := _AgentStore.contract.UnpackLog(event, "Added", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
