package worker

import (
	"crypto/rand"
	"encoding/hex"
	"hash"
	"sync/atomic"

	"github.com/screa/erc2470-address-miner/internal/crypto"
	"github.com/screa/erc2470-address-miner/pkg/types"
	"golang.org/x/crypto/sha3"
)

// Worker handles individual address generation and matching
type Worker struct {
	config   *types.WorkerConfig
	attempts *int64

	// Per-worker hasher and buffers (zero allocations in hot path)
	hasher   hash.Hash
	inputBuf [crypto.Create2InputLen]byte
	hashBuf  [32]byte
	addrBuf  [20]byte
	saltBuf  [32]byte
	hexBuf   [64]byte

	// Fast PRNG state (wyrand-like) for salt generation without syscalls
	prngState uint64
}

// NewWorker creates a new worker instance
func NewWorker(config *types.WorkerConfig, attempts *int64) *Worker {
	w := &Worker{
		config:   config,
		attempts: attempts,
		hasher:   sha3.NewLegacyKeccak256(),
	}
	// Seed PRNG with crypto randomness once
	var seed [8]byte
	if _, err := rand.Read(seed[:]); err == nil {
		w.prngState = uint64(seed[0]) | uint64(seed[1])<<8 | uint64(seed[2])<<16 | uint64(seed[3])<<24 |
			uint64(seed[4])<<32 | uint64(seed[5])<<40 | uint64(seed[6])<<48 | uint64(seed[7])<<56
	}
	if w.prngState == 0 {
		w.prngState = 1
	}
	return w
}

// fastRandUint64 returns a random uint64 from the worker's PRNG (non-crypto, for salt exploration)
func (w *Worker) fastRandUint64() uint64 {
	w.prngState += 0x60bee2b3d4d4a6c5 // wyrand constant
	return w.prngState * (w.prngState >> 32)
}

// fastSaltBytes fills w.saltBuf with 32 random bytes from the fast PRNG
func (w *Worker) fastSaltBytes() {
	for i := 0; i < 32; i += 8 {
		u := w.fastRandUint64()
		w.saltBuf[i] = byte(u)
		w.saltBuf[i+1] = byte(u >> 8)
		w.saltBuf[i+2] = byte(u >> 16)
		w.saltBuf[i+3] = byte(u >> 24)
		w.saltBuf[i+4] = byte(u >> 32)
		w.saltBuf[i+5] = byte(u >> 40)
		w.saltBuf[i+6] = byte(u >> 48)
		w.saltBuf[i+7] = byte(u >> 56)
	}
}

func (w *Worker) saltHexString() string {
	hex.Encode(w.hexBuf[:], w.saltBuf[:])
	return string(w.hexBuf[:64])
}

// GenerateAddress generates a single address and checks if it matches criteria (fast path).
func (w *Worker) GenerateAddress() *types.WorkerResult {
	w.fastSaltBytes()
	// Build CREATE2 input: prefix(21) + salt(32) + suffix(32)
	copy(w.inputBuf[0:crypto.Create2PrefixLen], w.config.Create2Prefix)
	copy(w.inputBuf[crypto.Create2PrefixLen:crypto.Create2PrefixLen+32], w.saltBuf[:])
	copy(w.inputBuf[crypto.Create2PrefixLen+32:], w.config.Create2Suffix)

	crypto.Create2AddressInto(w.hasher, w.inputBuf[:], w.hashBuf[:], w.addrBuf[:])

	// Batched atomic: add 1 to global every attempt (keep exact count for simplicity; could batch later)
	atomic.AddInt64(w.attempts, 1)

	isMatch := w.matchesBytes(w.addrBuf[:])
	if !isMatch {
		return &types.WorkerResult{
			SaltBytes:    w.saltBuf,
			AddressBytes: w.addrBuf,
			Attempts:     atomic.LoadInt64(w.attempts),
			IsMatch:      false,
		}
	}
	return &types.WorkerResult{
		Salt:         w.saltHexString(),
		SaltBytes:    w.saltBuf,
		Address:      crypto.AddressBytesToChecksumString(w.addrBuf[:]),
		AddressBytes: w.addrBuf,
		Attempts:     atomic.LoadInt64(w.attempts),
		IsMatch:      true,
	}
}

// matchesBytes performs pattern matching on raw 20-byte address (no string allocation)
func (w *Worker) matchesBytes(addr []byte) bool {
	if len(addr) != 20 {
		return false
	}
	if len(w.config.TargetBytes) == 20 {
		return equalBytes(addr, w.config.TargetBytes)
	}
	if len(w.config.PrefixBytes) > 0 {
		n := len(w.config.PrefixBytes)
		if n > 20 {
			n = 20
		}
		if !equalBytes(addr[:n], w.config.PrefixBytes) {
			return false
		}
	}
	if len(w.config.SuffixBytes) > 0 {
		n := len(w.config.SuffixBytes)
		if n > 20 {
			n = 20
		}
		if !equalBytes(addr[20-n:], w.config.SuffixBytes) {
			return false
		}
	}
	return true
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ProcessBatch processes a batch of address generations (legacy; miner uses GenerateAddress in loop)
func (w *Worker) ProcessBatch(batchSize int) *types.WorkerResult {
	for i := 0; i < batchSize; i++ {
		r := w.GenerateAddress()
		if r.IsMatch {
			return r
		}
	}
	return nil
}
