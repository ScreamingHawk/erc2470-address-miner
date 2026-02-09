package types

import "time"

// Result represents a mining result
type Result struct {
	Salt     string
	Address  string
	Attempts int64
	Duration time.Duration
}

// WorkerConfig contains configuration for individual workers
type WorkerConfig struct {
	Initcode     []byte
	InitcodeHash []byte
	FactoryBytes []byte
	Prefix       string
	Suffix       string
	Verbose      bool

	// Pre-decoded for fast byte-level matching (hot path). Nil if not set.
	PrefixBytes   []byte // first N bytes of address must match
	SuffixBytes   []byte // last N bytes of address must match
	Create2Prefix []byte // 21 bytes: 0xff + factory, constant per run
	Create2Suffix []byte // 32 bytes: initcode hash, constant per run
}

// WorkerResult represents a result from a single worker
type WorkerResult struct {
	Salt         string    // hex-encoded, only set when needed for output
	SaltBytes    [32]byte  // raw salt for building Salt when updating best
	Address      string    // EIP-55 checksummed, only set when needed for output
	AddressBytes [20]byte  // raw 20-byte address for comparison
	Attempts     int64
	IsMatch      bool
}
