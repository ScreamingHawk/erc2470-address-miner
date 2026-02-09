package miner

import (
	"encoding/hex"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/screa/erc2470-address-miner/internal/config"
	"github.com/screa/erc2470-address-miner/internal/crypto"
	"github.com/screa/erc2470-address-miner/internal/logger"
	"github.com/screa/erc2470-address-miner/pkg/types"
	"github.com/screa/erc2470-address-miner/pkg/worker"
)

// Miner provides high-performance address mining coordination
type Miner struct {
	config          *config.Config
	logger          *logger.Logger
	attempts        int64
	bestResult      *types.Result
	bestResultBytes [20]byte // for fast isBetter comparison
	mu              sync.RWMutex
	done            chan bool
	wg              sync.WaitGroup
	once            sync.Once
	workerConfig    *types.WorkerConfig
}

// NewMiner creates a new miner instance
func NewMiner(cfg *config.Config, log *logger.Logger) *Miner {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	// Pre-compute initcode and its hash for performance
	initcode, err := cfg.GetBytecode()
	if err != nil {
		panic("bytecode not available: " + err.Error())
	}

	initcodeHash := crypto.Keccak256(initcode)

	// Pre-compute factory address bytes
	factoryBytes, err := crypto.MustAddressBytes(crypto.FactoryAddress)
	if err != nil {
		panic("invalid factory address: " + err.Error())
	}

	// Pre-decode prefix/suffix/target for fast byte-level matching
	var targetBytes, prefixBytes, suffixBytes []byte
	if cfg.Target != "" {
		targetBytes, err = crypto.HexToAddressBytes(cfg.Target)
		if err != nil || len(targetBytes) != 20 {
			panic("invalid target address length (must be 20 bytes / 40 hex chars)")
		}
	}
	if cfg.Prefix != "" {
		prefixBytes, err = crypto.HexToAddressBytes(cfg.Prefix)
		if err != nil {
			panic("invalid prefix: " + err.Error())
		}
	}
	if cfg.Suffix != "" {
		suffixBytes, err = crypto.HexToAddressBytes(cfg.Suffix)
		if err != nil {
			panic("invalid suffix: " + err.Error())
		}
	}

	prefix21 := crypto.Create2PrefixBytes()
	workerConfig := &types.WorkerConfig{
		Initcode:      initcode,
		InitcodeHash:  initcodeHash,
		FactoryBytes:  factoryBytes,
		Target:        cfg.Target,
		Prefix:        cfg.Prefix,
		Suffix:        cfg.Suffix,
		Verbose:       cfg.Verbose,
		TargetBytes:   targetBytes,
		PrefixBytes:   prefixBytes,
		SuffixBytes:   suffixBytes,
		Create2Prefix: prefix21[:],
		Create2Suffix: initcodeHash,
	}

	return &Miner{
		config:       cfg,
		logger:       log,
		done:         make(chan bool),
		workerConfig: workerConfig,
	}
}

// Mine starts the mining process
func (m *Miner) Mine() *types.Result {
	start := time.Now()

	// Start workers
	for i := 0; i < m.config.Workers; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}

	// Start periodic logging if verbose mode is enabled
	var logTicker *time.Ticker
	var logDone chan bool
	if m.config.Verbose {
		interval := time.Duration(m.config.LogInterval) * time.Second
		logTicker = time.NewTicker(interval)
		logDone = make(chan bool)
		go m.periodicLogger(logTicker, logDone, start)

		// Log initial start message
		m.logger.Printf("Mining started with %d workers, logging every %d seconds...",
			m.config.Workers, m.config.LogInterval)
	}

	// Wait for completion
	m.wg.Wait()

	// Stop periodic logging
	if logTicker != nil {
		logTicker.Stop()
		close(logDone)
	}

	if m.bestResult != nil {
		m.bestResult.Duration = time.Since(start)
	}

	return m.bestResult
}

// worker runs the mining logic for a single worker
func (m *Miner) worker(workerID int) {
	defer m.wg.Done()

	batchSize := 1000 // Process in batches for better performance
	w := worker.NewWorker(m.workerConfig, &m.attempts)

	for {
		select {
		case <-m.done:
			return
		default:
			// Process a batch of attempts; check done only once per batch
			for i := 0; i < batchSize; i++ {
				result := w.GenerateAddress()
				if result == nil {
					continue
				}

				// For zero prefix, track the best (lowest) address found for all addresses
				if m.config.IsZeroPrefix() {
					m.mu.Lock()
					if m.bestResult == nil || m.isBetterBytes(result.AddressBytes, m.bestResultBytes) {
						saltStr := result.Salt
						if saltStr == "" {
							saltStr = hex.EncodeToString(result.SaltBytes[:])
						}
						addrStr := result.Address
						if addrStr == "" {
							addrStr = crypto.AddressBytesToChecksumString(result.AddressBytes[:])
						}
						m.bestResult = &types.Result{
							Salt:     saltStr,
							Address:  addrStr,
							Attempts: result.Attempts,
						}
						m.bestResultBytes = result.AddressBytes
					}
					m.mu.Unlock()
				}

				// Check if this matches our criteria
				if result.IsMatch {
					m.mu.Lock()
					if m.bestResult == nil || m.isBetterBytes(result.AddressBytes, m.bestResultBytes) {
						saltStr := result.Salt
						if saltStr == "" {
							saltStr = hex.EncodeToString(result.SaltBytes[:])
						}
						addrStr := result.Address
						if addrStr == "" {
							addrStr = crypto.AddressBytesToChecksumString(result.AddressBytes[:])
						}
						m.bestResult = &types.Result{
							Salt:     saltStr,
							Address:  addrStr,
							Attempts: result.Attempts,
						}
						m.bestResultBytes = result.AddressBytes
						m.once.Do(func() { close(m.done) })
					}
					m.mu.Unlock()
					return
				}
			}
		}
	}
}

// isBetterBytes compares two 20-byte addresses; returns true if new is lexicographically smaller (lower address).
// Zero oldAddr is treated as "no previous best" so any new address is better.
func (m *Miner) isBetterBytes(newAddr, oldAddr [20]byte) bool {
	// No previous best (all zeros): accept any
	var zero [20]byte
	if oldAddr == zero {
		return true
	}
	for i := 0; i < 20; i++ {
		if newAddr[i] != oldAddr[i] {
			return newAddr[i] < oldAddr[i]
		}
	}
	return false
}

// Stop stops the mining process
func (m *Miner) Stop() {
	m.once.Do(func() { close(m.done) })
}

// GetBestResult returns the current best result
func (m *Miner) GetBestResult() *types.Result {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bestResult
}

// periodicLogger logs mining progress at regular intervals
func (m *Miner) periodicLogger(ticker *time.Ticker, done chan bool, start time.Time) {
	for {
		select {
		case <-ticker.C:
			attempts := atomic.LoadInt64(&m.attempts)
			elapsed := time.Since(start)

			// Calculate rate safely
			rate := 0.0
			if elapsed.Seconds() > 0 {
				rate = float64(attempts) / elapsed.Seconds()
			}

			m.mu.RLock()
			bestResult := m.bestResult
			m.mu.RUnlock()

			if bestResult != nil {
				if m.config.IsZeroPrefix() {
					m.logger.Printf("Progress: %d attempts, %.2f hashes/sec, Best so far: %s (salt: 0x%s)",
						attempts, rate, bestResult.Address, bestResult.Salt)
				} else {
					m.logger.Printf("Progress: %d attempts, %.2f hashes/sec, Best: %s (salt: 0x%s)",
						attempts, rate, bestResult.Address, bestResult.Salt)
				}
			} else {
				m.logger.Printf("Progress: %d attempts, %.2f hashes/sec, No match yet",
					attempts, rate)
			}
		case <-done:
			return
		}
	}
}
