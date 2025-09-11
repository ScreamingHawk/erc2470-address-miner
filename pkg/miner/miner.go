package miner

import (
	"math/big"
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
	config       *config.Config
	logger       *logger.Logger
	attempts     int64
	bestResult   *types.Result
	mu           sync.RWMutex
	done         chan bool
	wg           sync.WaitGroup
	once         sync.Once
	workerConfig *types.WorkerConfig
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

	workerConfig := &types.WorkerConfig{
		Initcode:     initcode,
		InitcodeHash: initcodeHash,
		FactoryBytes: factoryBytes,
		Target:       cfg.Target,
		Prefix:       cfg.Prefix,
		Suffix:       cfg.Suffix,
		Verbose:      cfg.Verbose,
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
	w := worker.NewWorker(m.workerConfig, &m.attempts, &atomic.Value{})

	for {
		select {
		case <-m.done:
			return
		default:
			// Process a batch of attempts
			for i := 0; i < batchSize; i++ {
				// Check if we should stop before each attempt
				select {
				case <-m.done:
					return
				default:
				}

				result := w.GenerateAddress()
				if result == nil {
					continue
				}

				// For zero prefix, track the best (lowest) address found for all addresses
				if m.config.IsZeroPrefix() {
					m.mu.Lock()
					if m.bestResult == nil || m.isBetterOptimized(result.Address, m.bestResult.Address) {
						m.bestResult = &types.Result{
							Salt:     result.Salt,
							Address:  result.Address,
							Attempts: result.Attempts,
						}
					}
					m.mu.Unlock()
				}

				// Check if this matches our criteria
				if result.IsMatch {
					m.mu.Lock()
					if m.bestResult == nil || m.isBetterOptimized(result.Address, m.bestResult.Address) {
						m.bestResult = &types.Result{
							Salt:     result.Salt,
							Address:  result.Address,
							Attempts: result.Attempts,
						}
						m.once.Do(func() { close(m.done) })
					}
					m.mu.Unlock()
					return
				}
			}
		}
	}
}

// isBetterOptimized performs optimized address comparison
func (m *Miner) isBetterOptimized(newAddr, oldAddr string) bool {
	// Remove 0x prefix for comparison
	newClean := newAddr[2:] // Skip "0x"
	oldClean := oldAddr[2:] // Skip "0x"

	// Compare as big integers to find "lowest" address
	newInt, _ := new(big.Int).SetString(newClean, 16)
	oldInt, _ := new(big.Int).SetString(oldClean, 16)

	return newInt.Cmp(oldInt) < 0
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
