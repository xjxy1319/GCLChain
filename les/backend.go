// Copyright 2016 The go-gclchaineum Authors
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

// Package les implements the Light Gclchain Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/gclchaineum/go-gclchaineum/accounts"
	"github.com/gclchaineum/go-gclchaineum/common"
	"github.com/gclchaineum/go-gclchaineum/common/hexutil"
	"github.com/gclchaineum/go-gclchaineum/consensus"
	"github.com/gclchaineum/go-gclchaineum/core"
	"github.com/gclchaineum/go-gclchaineum/core/bloombits"
	"github.com/gclchaineum/go-gclchaineum/core/rawdb"
	"github.com/gclchaineum/go-gclchaineum/core/types"
	"github.com/gclchaineum/go-gclchaineum/gcl"
	"github.com/gclchaineum/go-gclchaineum/gcl/downloader"
	"github.com/gclchaineum/go-gclchaineum/gcl/filters"
	"github.com/gclchaineum/go-gclchaineum/gcl/gasprice"
	"github.com/gclchaineum/go-gclchaineum/event"
	"github.com/gclchaineum/go-gclchaineum/internal/gclapi"
	"github.com/gclchaineum/go-gclchaineum/light"
	"github.com/gclchaineum/go-gclchaineum/log"
	"github.com/gclchaineum/go-gclchaineum/node"
	"github.com/gclchaineum/go-gclchaineum/p2p"
	"github.com/gclchaineum/go-gclchaineum/p2p/discv5"
	"github.com/gclchaineum/go-gclchaineum/params"
	rpc "github.com/gclchaineum/go-gclchaineum/rpc"
)

type LightGclchain struct {
	lesCommons

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool

	// Handlers
	peers      *peerSet
	txPool     *light.TxPool
	blockchain *light.LightChain
	serverPool *serverPool
	reqDist    *requestDistributor
	retriever  *retrieveManager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *gclapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *gcl.Config) (*LightGclchain, error) {
	chainDb, err := gcl.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.ConstantinopleOverride)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lgcl := &LightGclchain{
		lesCommons: lesCommons{
			chainDb: chainDb,
			config:  config,
			iConfig: light.DefaultClientIndexerConfig,
		},
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		peers:          peers,
		reqDist:        newRequestDistributor(peers, quitSync),
		accountManager: ctx.AccountManager,
		engine:         gcl.CreateConsensusEngine(ctx, chainConfig, &config.Ethash, nil, false, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   gcl.NewBloomIndexer(chainDb, params.BloomBitsBlocksClient, params.HelperTrieConfirmations),
	}

	lgcl.relay = NewLesTxRelay(peers, lgcl.reqDist)
	lgcl.serverPool = newServerPool(chainDb, quitSync, &lgcl.wg)
	lgcl.retriever = newRetrieveManager(peers, lgcl.reqDist, lgcl.serverPool)

	lgcl.odr = NewLesOdr(chainDb, light.DefaultClientIndexerConfig, lgcl.retriever)
	lgcl.chtIndexer = light.NewChtIndexer(chainDb, lgcl.odr, params.CHTFrequencyClient, params.HelperTrieConfirmations)
	lgcl.bloomTrieIndexer = light.NewBloomTrieIndexer(chainDb, lgcl.odr, params.BloomBitsBlocksClient, params.BloomTrieFrequency)
	lgcl.odr.SetIndexers(lgcl.chtIndexer, lgcl.bloomTrieIndexer, lgcl.bloomIndexer)

	// Note: NewLightChain adds the trusted checkpoint so it needs an ODR with
	// indexers already set but not started yet
	if lgcl.blockchain, err = light.NewLightChain(lgcl.odr, lgcl.chainConfig, lgcl.engine); err != nil {
		return nil, err
	}
	// Note: AddChildIndexer starts the update process for the child
	lgcl.bloomIndexer.AddChildIndexer(lgcl.bloomTrieIndexer)
	lgcl.chtIndexer.Start(lgcl.blockchain)
	lgcl.bloomIndexer.Start(lgcl.blockchain)

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lgcl.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lgcl.txPool = light.NewTxPool(lgcl.chainConfig, lgcl.blockchain, lgcl.relay)
	if lgcl.protocolManager, err = NewProtocolManager(lgcl.chainConfig, light.DefaultClientIndexerConfig, true, config.NetworkId, lgcl.eventMux, lgcl.engine, lgcl.peers, lgcl.blockchain, nil, chainDb, lgcl.odr, lgcl.relay, lgcl.serverPool, quitSync, &lgcl.wg); err != nil {
		return nil, err
	}
	lgcl.ApiBackend = &LesApiBackend{lgcl, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.MinerGasPrice
	}
	lgcl.ApiBackend.gpo = gasprice.NewOracle(lgcl.ApiBackend, gpoParams)
	return lgcl, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Gclchainbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Gclchainbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for Gclchainbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the gclchaineum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightGclchain) APIs() []rpc.API {
	return append(gclapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "gcl",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "gcl",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "gcl",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightGclchain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightGclchain) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightGclchain) TxPool() *light.TxPool              { return s.txPool }
func (s *LightGclchain) Engine() consensus.Engine           { return s.engine }
func (s *LightGclchain) LesVersion() int                    { return int(ClientProtocolVersions[0]) }
func (s *LightGclchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightGclchain) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightGclchain) Protocols() []p2p.Protocol {
	return s.makeProtocols(ClientProtocolVersions)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Gclchain protocol implementation.
func (s *LightGclchain) Start(srvr *p2p.Server) error {
	log.Warn("Light client mode is an experimental feature")
	s.startBloomHandlers(params.BloomBitsBlocksClient)
	s.netRPCService = gclapi.NewPublicNetAPI(srvr, s.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Gclchain protocol.
func (s *LightGclchain) Stop() error {
	s.odr.Stop()
	s.bloomIndexer.Close()
	s.chtIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()
	s.engine.Close()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
