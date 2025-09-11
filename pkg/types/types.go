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
	Target       string
	Prefix       string
	Suffix       string
	Verbose      bool
}

// WorkerResult represents a result from a single worker
type WorkerResult struct {
	Salt     string
	Address  string
	Attempts int64
	IsMatch  bool
}
