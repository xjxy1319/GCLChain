// Copyright 2014 The go-gclchaineum Authors
// This file is part of the go-gclchaineum library.
//
// The go-gclchaineum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-gclchaineum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-gclchaineum library. If not, see <http://www.gnu.org/licenses/>.

// Package miner implements Gclchain block creation and mining.
package miner

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gclchaineum/go-gclchaineum/common"
	"github.com/gclchaineum/go-gclchaineum/consensus"
	"github.com/gclchaineum/go-gclchaineum/core"
	"github.com/gclchaineum/go-gclchaineum/core/state"
	"github.com/gclchaineum/go-gclchaineum/core/types"
	"github.com/gclchaineum/go-gclchaineum/gcl/downloader"
	"github.com/gclchaineum/go-gclchaineum/event"
	"github.com/gclchaineum/go-gclchaineum/log"
	"github.com/gclchaineum/go-gclchaineum/params"
)

// Backend wraps all mgclods required for mining.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
}

// Miner creates blocks and searches for proof-of-work values.
type Miner struct {
	mux      *event.TypeMux
	worker   *worker
	coinbase common.Address
	gcl      Backend
	engine   consensus.Engine
	exitCh   chan struct{}

	canStart    int32 // can start indicates whgclchain we can start the mining operation
	shouldStart int32 // should start indicates whgclchain we should start after sync
}

func New(gcl Backend, config *params.ChainConfig, mux *event.TypeMux, engine consensus.Engine, recommit time.Duration, gasFloor, gasCeil uint64, isLocalBlock func(block *types.Block) bool) *Miner {
	miner := &Miner{
		gcl:      gcl,
		mux:      mux,
		engine:   engine,
		exitCh:   make(chan struct{}),
		worker:   newWorker(config, engine, gcl, mux, recommit, gasFloor, gasCeil, isLocalBlock),
		canStart: 1,
	}
	go miner.update()

	return miner
}

// update keeps track of the downloader events. Please be aware that this is a one shot type of update loop.
// It's entered once and as soon as `Done` or `Failed` has been broadcasted the events are unregistered and
// the loop is exited. This to prevent a major security vuln where external parties can DOS you with blocks
// and halt your mining operation for as long as the DOS continues.
func (self *Miner) update() {
	events := self.mux.Subscribe(downloader.StartEvent{}, downloader.DoneEvent{}, downloader.FailedEvent{})
	defer events.Unsubscribe()

	for {
		select {
		case ev := <-events.Chan():
			if ev == nil {
				return
			}
			switch ev.Data.(type) {
			case downloader.StartEvent:
				atomic.StoreInt32(&self.canStart, 0)
				if self.Mining() {
					self.Stop()
					atomic.StoreInt32(&self.shouldStart, 1)
					log.Info("Mining aborted due to sync")
				}
			case downloader.DoneEvent, downloader.FailedEvent:
				shouldStart := atomic.LoadInt32(&self.shouldStart) == 1

				atomic.StoreInt32(&self.canStart, 1)
				atomic.StoreInt32(&self.shouldStart, 0)
				if shouldStart {
					self.Start(self.coinbase)
				}
				// stop immediately and ignore all further pending events
				return
			}
		case <-self.exitCh:
			return
		}
	}
}

func (self *Miner) Start(coinbase common.Address) {
	atomic.StoreInt32(&self.shouldStart, 1)
	self.SetGclchainbase(coinbase)

	if atomic.LoadInt32(&self.canStart) == 0 {
		log.Info("Network syncing, will start miner afterwards")
		return
	}
	self.worker.start()
}

func (self *Miner) Stop() {
	self.worker.stop()
	atomic.StoreInt32(&self.shouldStart, 0)
}

func (self *Miner) Close() {
	self.worker.close()
	close(self.exitCh)
}

func (self *Miner) Mining() bool {
	return self.worker.isRunning()
}

func (self *Miner) HashRate() uint64 {
	if pow, ok := self.engine.(consensus.PoW); ok {
		return uint64(pow.Hashrate())
	}
	return 0
}

func (self *Miner) SetExtra(extra []byte) error {
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("Extra exceeds max length. %d > %v", len(extra), params.MaximumExtraDataSize)
	}
	self.worker.setExtra(extra)
	return nil
}

// SetRecommitInterval sets the interval for sealing work resubmitting.
func (self *Miner) SetRecommitInterval(interval time.Duration) {
	self.worker.setRecommitInterval(interval)
}

// Pending returns the currently pending block and associated state.
func (self *Miner) Pending() (*types.Block, *state.StateDB) {
	return self.worker.pending()
}

// PendingBlock returns the currently pending block.
//
// Note, to access both the pending block and the pending state
// simultaneously, please use Pending(), as the pending state can
// change between multiple mgclod calls
func (self *Miner) PendingBlock() *types.Block {
	return self.worker.pendingBlock()
}

func (self *Miner) SetGclchainbase(addr common.Address) {
	self.coinbase = addr
	self.worker.setGclchainbase(addr)
}
