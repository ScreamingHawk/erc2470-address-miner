package worker

import (
	"crypto/rand"
	"encoding/hex"
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

	// Pre-allocated buffers for performance
	saltBuffer [32]byte
	hexBuffer  [64]byte // 32 bytes * 2 for hex encoding

}

// NewWorker creates a new worker instance
func NewWorker(config *types.WorkerConfig, attempts *int64, mu *atomic.Value) *Worker {
	return &Worker{
		config:   config,
		attempts: attempts,
		mu:       mu,
	}
}

// fastHexEncode encodes bytes to hex string using pre-allocated buffer
func (w *Worker) fastHexEncode(data []byte) string {
	hex.Encode(w.hexBuffer[:], data)
	return string(w.hexBuffer[:len(data)*2])
}

// fastRandomSaltBytes generates a random salt as bytes (most efficient)
func (w *Worker) fastRandomSaltBytes() []byte {
	if _, err := rand.Read(w.saltBuffer[:]); err != nil {
		return nil
	}
	return w.saltBuffer[:]
}

// GenerateAddress generates a single address and checks if it matches criteria
func (w *Worker) GenerateAddress() *types.WorkerResult {
	// Generate random salt as bytes (most efficient)
	saltBytes := w.fastRandomSaltBytes()
	if saltBytes == nil {
		return nil
	}

	// Convert to hex string for the result
	salt := w.fastHexEncode(saltBytes)

	// Calculate address using the fast method
	address := crypto.CalculateCreate2Address(w.config.InitcodeHash, saltBytes)

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
	for i := 0; i < batchSize; i++ {
		// Generate random salt as bytes (most efficient)
		saltBytes := w.fastRandomSaltBytes()
		if saltBytes == nil {
			continue
		}

		// Convert to hex string for the result
		salt := w.fastHexEncode(saltBytes)

		// Calculate address using the fast method
		address := crypto.CalculateCreate2Address(w.config.InitcodeHash, saltBytes)

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
	}

	return nil
}

// matchesOptimized performs optimized pattern matching
func (w *Worker) matchesOptimized(address string) bool {
	// Remove 0x prefix for comparison - use unsafe to avoid allocation
	addr := address[2:] // Skip "0x"

	// Check target
	if w.config.Target != "" {
		targetClean := w.config.Target
		if len(targetClean) > 2 && targetClean[:2] == "0x" {
			targetClean = targetClean[2:]
		}
		return addr == targetClean
	}

	// Check prefix - use optimized string comparison
	if w.config.Prefix != "" {
		prefixClean := w.config.Prefix
		if len(prefixClean) > 2 && prefixClean[:2] == "0x" {
			prefixClean = prefixClean[2:]
		}
		prefixLen := len(prefixClean)
		if len(addr) >= prefixLen {
			return addr[:prefixLen] == prefixClean
		}
	}

	// Check suffix - use optimized string comparison
	if w.config.Suffix != "" {
		suffixClean := w.config.Suffix
		if len(suffixClean) > 2 && suffixClean[:2] == "0x" {
			suffixClean = suffixClean[2:]
		}
		suffixLen := len(suffixClean)
		if len(addr) >= suffixLen {
			return addr[len(addr)-suffixLen:] == suffixClean
		}
	}

	return false
}
