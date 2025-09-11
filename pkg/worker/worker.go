package worker

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"sync/atomic"

	"github.com/screa/erc2470-address-miner/internal/crypto"
	"github.com/screa/erc2470-address-miner/pkg/types"
)

// Worker handles individual address generation and matching
type Worker struct {
	config     *types.WorkerConfig
	attempts   *int64
	bestResult *types.Result
	mu         *atomic.Value // For thread-safe best result updates
}

// NewWorker creates a new worker instance
func NewWorker(config *types.WorkerConfig, attempts *int64, mu *atomic.Value) *Worker {
	return &Worker{
		config:   config,
		attempts: attempts,
		mu:       mu,
	}
}

// GenerateAddress generates a single address and checks if it matches criteria
func (w *Worker) GenerateAddress() *types.WorkerResult {
	// Generate random salt
	saltBuffer := make([]byte, 32)
	if _, err := rand.Read(saltBuffer); err != nil {
		return nil
	}

	salt := hex.EncodeToString(saltBuffer)
	address := w.hashToAddressOptimized(salt)

	// Increment attempt counter
	atomic.AddInt64(w.attempts, 1)

	// Check if this matches our criteria
	isMatch := w.matchesOptimized(address)

	return &types.WorkerResult{
		Salt:     salt,
		Address:  address,
		Attempts: atomic.LoadInt64(w.attempts),
		IsMatch:  isMatch,
	}
}

// ProcessBatch processes a batch of address generations
func (w *Worker) ProcessBatch(batchSize int) *types.WorkerResult {
	// Pre-allocate buffer for better performance
	saltBuffer := make([]byte, 32)

	for i := 0; i < batchSize; i++ {
		// Generate random salt
		if _, err := rand.Read(saltBuffer); err != nil {
			continue
		}

		salt := hex.EncodeToString(saltBuffer)
		address := w.hashToAddressOptimized(salt)

		// Increment attempt counter
		atomic.AddInt64(w.attempts, 1)

		// Check if this matches our criteria
		if w.matchesOptimized(address) {
			return &types.WorkerResult{
				Salt:     salt,
				Address:  address,
				Attempts: atomic.LoadInt64(w.attempts),
				IsMatch:  true,
			}
		}

		// For zero prefix, track the best (lowest) address found
		if w.config.Prefix == "0000" {
			w.updateBestResult(salt, address)
		}
	}

	return nil
}

// updateBestResult updates the best result if the new address is better
func (w *Worker) updateBestResult(salt, address string) {
	// This is a simplified version - in practice, you'd want proper synchronization
	// For now, we'll let the miner handle this logic
}

// hashToAddressOptimized performs optimized address generation
func (w *Worker) hashToAddressOptimized(salt string) string {
	// Calculate CREATE2 address using pre-computed factory bytes and initcode hash
	address, err := crypto.CalculateCreate2AddressOptimized(w.config.FactoryBytes, w.config.InitcodeHash, salt)
	if err != nil {
		// This should not happen with valid bytecode
		panic("CREATE2 calculation failed: " + err.Error())
	}

	return address
}

// matchesOptimized performs optimized pattern matching
func (w *Worker) matchesOptimized(address string) bool {
	// Remove 0x prefix for comparison
	addr := address[2:] // Skip "0x"

	// Check target
	if w.config.Target != "" {
		targetClean := w.config.Target
		if len(targetClean) > 2 && targetClean[:2] == "0x" {
			targetClean = targetClean[2:]
		}
		return addr == targetClean
	}

	// Check prefix
	if w.config.Prefix != "" {
		prefixClean := w.config.Prefix
		if len(prefixClean) > 2 && prefixClean[:2] == "0x" {
			prefixClean = prefixClean[2:]
		}
		if len(addr) >= len(prefixClean) && addr[:len(prefixClean)] == prefixClean {
			return true
		}
	}

	// Check suffix
	if w.config.Suffix != "" {
		suffixClean := w.config.Suffix
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
func (w *Worker) isBetterOptimized(newAddr, oldAddr string) bool {
	// Remove 0x prefix for comparison
	newClean := newAddr[2:] // Skip "0x"
	oldClean := oldAddr[2:] // Skip "0x"

	// Compare as big integers to find "lowest" address
	newInt, _ := new(big.Int).SetString(newClean, 16)
	oldInt, _ := new(big.Int).SetString(oldClean, 16)

	return newInt.Cmp(oldInt) < 0
}
