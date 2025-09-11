package miner

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/screa/erc2470-address-miner/internal/config"
	"github.com/screa/erc2470-address-miner/internal/crypto"
	"github.com/screa/erc2470-address-miner/internal/logger"
)

// Miner provides high-performance address mining with advanced optimizations
type Miner struct {
	config        *config.Config
	logger        *logger.Logger
	attempts      int64
	bestResult    *Result
	mu            sync.RWMutex
	done          chan bool
	wg            sync.WaitGroup
	once          sync.Once
	initcode      []byte
	initcodeHash  []byte
	factoryBytes  []byte
}

// Result represents a mining result
type Result struct {
	Salt     string
	Address  string
	Attempts int64
	Duration time.Duration
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
	
	return &Miner{
		config:        cfg,
		logger:        log,
		done:          make(chan bool),
		initcode:      initcode,
		initcodeHash:  initcodeHash,
		factoryBytes:  factoryBytes,
	}
}

// Mine starts the mining process
func (m *Miner) Mine() *Result {
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
	
	localAttempts := int64(0)
	batchSize := 1000 // Process in batches for better performance
	
	// Pre-allocate buffer for better performance
	saltBuffer := make([]byte, 32)
	
	for {
		select {
		case <-m.done:
			atomic.AddInt64(&m.attempts, localAttempts)
			return
		default:
			// Process a batch of attempts
			for i := 0; i < batchSize; i++ {
				// Check if we should stop before each attempt
				select {
				case <-m.done:
					atomic.AddInt64(&m.attempts, localAttempts)
					return
				default:
				}
				
				// Generate random salt
				if _, err := rand.Read(saltBuffer); err != nil {
					continue
				}
				
				salt := hex.EncodeToString(saltBuffer)
				address := m.hashToAddressOptimized(salt)
				
				localAttempts++
				
				// Check if this matches our criteria
				if m.matchesOptimized(address) {
					m.mu.Lock()
					if m.bestResult == nil || m.isBetterOptimized(address, m.bestResult.Address) {
						m.bestResult = &Result{
							Salt:     salt,
							Address:  address,
							Attempts: atomic.LoadInt64(&m.attempts) + localAttempts,
						}
						m.once.Do(func() { close(m.done) })
					}
					m.mu.Unlock()
					atomic.AddInt64(&m.attempts, localAttempts)
					return
				}
			}
			
			// Update global attempt counter after each batch
			atomic.AddInt64(&m.attempts, localAttempts)
			localAttempts = 0
		}
	}
}

// hashToAddressOptimized performs optimized address generation
func (m *Miner) hashToAddressOptimized(salt string) string {
	// Calculate CREATE2 address using pre-computed factory bytes and initcode hash
	address, err := crypto.CalculateCreate2AddressOptimized(m.factoryBytes, m.initcodeHash, salt)
	if err != nil {
		// This should not happen with valid bytecode
		panic("CREATE2 calculation failed: " + err.Error())
	}
	
	return address
}

// matchesOptimized performs optimized pattern matching
func (m *Miner) matchesOptimized(address string) bool {
	// Remove 0x prefix for comparison
	addr := address[2:] // Skip "0x"
	
	// Check target
	if m.config.Target != "" {
		targetClean := m.config.Target
		if len(targetClean) > 2 && targetClean[:2] == "0x" {
			targetClean = targetClean[2:]
		}
		return addr == targetClean
	}
	
	// Check prefix
	if m.config.Prefix != "" {
		prefixClean := m.config.Prefix
		if len(prefixClean) > 2 && prefixClean[:2] == "0x" {
			prefixClean = prefixClean[2:]
		}
		if len(addr) >= len(prefixClean) && addr[:len(prefixClean)] == prefixClean {
			return true
		}
	}
	
	// Check suffix
	if m.config.Suffix != "" {
		suffixClean := m.config.Suffix
		if len(suffixClean) > 2 && suffixClean[:2] == "0x" {
			suffixClean = suffixClean[2:]
		}
		if len(addr) >= len(suffixClean) && addr[len(addr)-len(suffixClean):] == suffixClean {
			return true
		}
	}
	
	return false
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
				m.logger.Printf("Progress: %d attempts, %.2f hashes/sec, Best: %s (salt: 0x%s)", 
					attempts, rate, bestResult.Address, bestResult.Salt)
			} else {
				m.logger.Printf("Progress: %d attempts, %.2f hashes/sec, No match yet", 
					attempts, rate)
			}
		case <-done:
			return
		}
	}
}

