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

// Package gcl implements the Gclchain protocol.
package gcl

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gclchaineum/go-gclchaineum/accounts"
	"github.com/gclchaineum/go-gclchaineum/common"
	"github.com/gclchaineum/go-gclchaineum/common/hexutil"
	"github.com/gclchaineum/go-gclchaineum/consensus"
	"github.com/gclchaineum/go-gclchaineum/consensus/clique"
	"github.com/gclchaineum/go-gclchaineum/consensus/gclash"
	"github.com/gclchaineum/go-gclchaineum/core"
	"github.com/gclchaineum/go-gclchaineum/core/bloombits"
	"github.com/gclchaineum/go-gclchaineum/core/rawdb"
	"github.com/gclchaineum/go-gclchaineum/core/types"
	"github.com/gclchaineum/go-gclchaineum/core/vm"
	"github.com/gclchaineum/go-gclchaineum/gcl/downloader"
	"github.com/gclchaineum/go-gclchaineum/gcl/filters"
	"github.com/gclchaineum/go-gclchaineum/gcl/gasprice"
	"github.com/gclchaineum/go-gclchaineum/gcldb"
	"github.com/gclchaineum/go-gclchaineum/event"
	"github.com/gclchaineum/go-gclchaineum/internal/gclapi"
	"github.com/gclchaineum/go-gclchaineum/log"
	"github.com/gclchaineum/go-gclchaineum/miner"
	"github.com/gclchaineum/go-gclchaineum/node"
	"github.com/gclchaineum/go-gclchaineum/p2p"
	"github.com/gclchaineum/go-gclchaineum/params"
	"github.com/gclchaineum/go-gclchaineum/rlp"
	"github.com/gclchaineum/go-gclchaineum/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// Gclchain implements the Gclchain full node service.
type Gclchain struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Gclchain

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb gcldb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *EthAPIBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	gclchainbase common.Address

	networkID     uint64
	netRPCService *gclapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and gclchainbase)
}

func (s *Gclchain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new Gclchain object (including the
// initialisation of the common Gclchain object)
func New(ctx *node.ServiceContext, config *Config) (*Gclchain, error) {
	// Ensure configuration values are compatible and sane
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run gcl.Gclchain in light sync mode, use les.LightGclchain")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if config.MinerGasPrice == nil || config.MinerGasPrice.Cmp(common.Big0) <= 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", config.MinerGasPrice, "updated", DefaultConfig.MinerGasPrice)
		config.MinerGasPrice = new(big.Int).Set(DefaultConfig.MinerGasPrice)
	}
	// Assemble the Gclchain object
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.ConstantinopleOverride)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	gcl := &Gclchain{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, chainConfig, &config.Ethash, config.MinerNotify, config.MinerNoverify, chainDb),
		shutdownChan:   make(chan bool),
		networkID:      config.NetworkId,
		gasPrice:       config.MinerGasPrice,
		gclchainbase:      config.Gclchainbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
	}

	log.Info("Initialising Gclchain protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, Ggcl %s only supports v%d", *bcVersion, params.VersionWithMeta, core.BlockChainVersion)
		} else if bcVersion != nil && *bcVersion < core.BlockChainVersion {
			log.Warn("Upgrade blockchain database version", "from", *bcVersion, "to", core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig = vm.Config{
			EnablePreimageRecording: config.EnablePreimageRecording,
			EWASMInterpreter:        config.EWASMInterpreter,
			EVMInterpreter:          config.EVMInterpreter,
		}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieCleanLimit: config.TrieCleanCache, TrieDirtyLimit: config.TrieDirtyCache, TrieTimeLimit: config.TrieTimeout}
	)
	gcl.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, gcl.chainConfig, gcl.engine, vmConfig, gcl.shouldPreserve)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		gcl.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	gcl.bloomIndexer.Start(gcl.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	gcl.txPool = core.NewTxPool(config.TxPool, gcl.chainConfig, gcl.blockchain)

	if gcl.protocolManager, err = NewProtocolManager(gcl.chainConfig, config.SyncMode, config.NetworkId, gcl.eventMux, gcl.txPool, gcl.engine, gcl.blockchain, chainDb, config.Whitelist); err != nil {
		return nil, err
	}

	gcl.miner = miner.New(gcl, gcl.chainConfig, gcl.EventMux(), gcl.engine, config.MinerRecommit, config.MinerGasFloor, config.MinerGasCeil, gcl.isLocalBlock)
	gcl.miner.SetExtra(makeExtraData(config.MinerExtraData))

	gcl.APIBackend = &EthAPIBackend{gcl, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.MinerGasPrice
	}
	gcl.APIBackend.gpo = gasprice.NewOracle(gcl.APIBackend, gpoParams)

	return gcl, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"ggcl",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (gcldb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*gcldb.LDBDatabase); ok {
		db.Meter("gcl/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Gclchain service
func CreateConsensusEngine(ctx *node.ServiceContext, chainConfig *params.ChainConfig, config *gclash.Config, notify []string, noverify bool, db gcldb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch config.PowMode {
	case gclash.ModeFake:
		log.Warn("Ethash used in fake mode")
		return gclash.NewFaker()
	case gclash.ModeTest:
		log.Warn("Ethash used in test mode")
		return gclash.NewTester(nil, noverify)
	case gclash.ModeShared:
		log.Warn("Ethash used in shared mode")
		return gclash.NewShared()
	default:
		engine := gclash.New(gclash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		}, notify, noverify)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs return the collection of RPC services the gclchaineum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Gclchain) APIs() []rpc.API {
	apis := gclapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "gcl",
			Version:   "1.0",
			Service:   NewPublicGclchainAPI(s),
			Public:    true,
		}, {
			Namespace: "gcl",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "gcl",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "gcl",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Gclchain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Gclchain) Gclchainbase() (eb common.Address, err error) {
	s.lock.RLock()
	gclchainbase := s.gclchainbase
	s.lock.RUnlock()

	if gclchainbase != (common.Address{}) {
		return gclchainbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			gclchainbase := accounts[0].Address

			s.lock.Lock()
			s.gclchainbase = gclchainbase
			s.lock.Unlock()

			log.Info("Gclchainbase automatically configured", "address", gclchainbase)
			return gclchainbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("gclchainbase must be explicitly specified")
}

// isLocalBlock checks whgclchain the specified block is mined
// by local miner accounts.
//
// We regard two types of accounts as local miner account: gclchainbase
// and accounts specified via `txpool.locals` flag.
func (s *Gclchain) isLocalBlock(block *types.Block) bool {
	author, err := s.engine.Author(block.Header())
	if err != nil {
		log.Warn("Failed to retrieve block author", "number", block.NumberU64(), "hash", block.Hash(), "err", err)
		return false
	}
	// Check whgclchain the given address is gclchainbase.
	s.lock.RLock()
	gclchainbase := s.gclchainbase
	s.lock.RUnlock()
	if author == gclchainbase {
		return true
	}
	// Check whgclchain the given address is specified by `txpool.local`
	// CLI flag.
	for _, account := range s.config.TxPool.Locals {
		if account == author {
			return true
		}
	}
	return false
}

// shouldPreserve checks whgclchain we should preserve the given block
// during the chain reorg depending on whgclchain the author of block
// is a local account.
func (s *Gclchain) shouldPreserve(block *types.Block) bool {
	// The reason we need to disable the self-reorg preserving for clique
	// is it can be probable to introduce a deadlock.
	//
	// e.g. If there are 7 available signers
	//
	// r1   A
	// r2     B
	// r3       C
	// r4         D
	// r5   A      [X] F G
	// r6    [X]
	//
	// In the round5, the inturn signer E is offline, so the worst case
	// is A, F and G sign the block of round5 and reject the block of opponents
	// and in the round6, the last available signer B is offline, the whole
	// network is stuck.
	if _, ok := s.engine.(*clique.Clique); ok {
		return false
	}
	return s.isLocalBlock(block)
}

// SetGclchainbase sets the mining reward address.
func (s *Gclchain) SetGclchainbase(gclchainbase common.Address) {
	s.lock.Lock()
	s.gclchainbase = gclchainbase
	s.lock.Unlock()

	s.miner.SetGclchainbase(gclchainbase)
}

// StartMining starts the miner with the given number of CPU threads. If mining
// is already running, this mgclod adjust the number of threads allowed to use
// and updates the minimum price required by the transaction pool.
func (s *Gclchain) StartMining(threads int) error {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		log.Info("Updated mining threads", "threads", threads)
		if threads == 0 {
			threads = -1 // Disable the miner from within
		}
		th.SetThreads(threads)
	}
	// If the miner was not running, initialize it
	if !s.IsMining() {
		// Propagate the initial price point to the transaction pool
		s.lock.RLock()
		price := s.gasPrice
		s.lock.RUnlock()
		s.txPool.SetGasPrice(price)

		// Configure the local mining address
		eb, err := s.Gclchainbase()
		if err != nil {
			log.Error("Cannot start mining without gclchainbase", "err", err)
			return fmt.Errorf("gclchainbase missing: %v", err)
		}
		if clique, ok := s.engine.(*clique.Clique); ok {
			wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
			if wallet == nil || err != nil {
				log.Error("Gclchainbase account unavailable locally", "err", err)
				return fmt.Errorf("signer missing: %v", err)
			}
			clique.Authorize(eb, wallet.SignHash)
		}
		// If mining is started, we can disable the transaction rejection mechanism
		// introduced to speed sync times.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)

		go s.miner.Start(eb)
	}
	return nil
}

// StopMining terminates the miner, both at the consensus engine level as well as
// at the block creation level.
func (s *Gclchain) StopMining() {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	// Stop the block creating itself
	s.miner.Stop()
}

func (s *Gclchain) IsMining() bool      { return s.miner.Mining() }
func (s *Gclchain) Miner() *miner.Miner { return s.miner }

func (s *Gclchain) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Gclchain) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Gclchain) TxPool() *core.TxPool               { return s.txPool }
func (s *Gclchain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Gclchain) Engine() consensus.Engine           { return s.engine }
func (s *Gclchain) ChainDb() gcldb.Database            { return s.chainDb }
func (s *Gclchain) IsListening() bool                  { return true } // Always listening
func (s *Gclchain) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Gclchain) NetVersion() uint64                 { return s.networkID }
func (s *Gclchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Gclchain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Gclchain protocol implementation.
func (s *Gclchain) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers(params.BloomBitsBlocks)

	// Start the RPC service
	s.netRPCService = gclapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Gclchain protocol.
func (s *Gclchain) Stop() error {
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.engine.Close()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)
	return nil
}
