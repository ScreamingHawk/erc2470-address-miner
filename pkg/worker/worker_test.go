package worker

import (
	"sync/atomic"
	"testing"

	"github.com/screa/erc2470-address-miner/pkg/types"
)

func TestNewWorker(t *testing.T) {
	config := &types.WorkerConfig{
		Initcode:     []byte{0x60, 0x80, 0x60, 0x40},
		InitcodeHash: []byte{0x12, 0x34, 0x56, 0x78},
		FactoryBytes: []byte{0xce, 0x00, 0x42, 0xb8},
		Prefix:       "0000",
	}

	attempts := int64(0)
	mu := &atomic.Value{}

	worker := NewWorker(config, &attempts, mu)
	if worker == nil {
		t.Fatal("NewWorker returned nil")
	}

	if worker.config != config {
		t.Error("Config not set correctly")
	}
}

func TestWorkerMatches(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		config   *types.WorkerConfig
		expected bool
	}{
		{
			name:    "prefix match",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			config: &types.WorkerConfig{
				Prefix: "1234",
			},
			expected: true,
		},
		{
			name:    "suffix match",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			config: &types.WorkerConfig{
				Suffix: "5678",
			},
			expected: true,
		},
		{
			name:    "target match",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			config: &types.WorkerConfig{
				Target: "1234567890abcdef1234567890abcdef12345678",
			},
			expected: true,
		},
		{
			name:    "no match",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			config: &types.WorkerConfig{
				Prefix: "9999",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := int64(0)
			mu := &atomic.Value{}
			worker := NewWorker(tt.config, &attempts, mu)
			result := worker.matchesOptimized(tt.address)
			if result != tt.expected {
				t.Errorf("matchesOptimized() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWorkerIsBetter(t *testing.T) {
	tests := []struct {
		name     string
		newAddr  string
		oldAddr  string
		expected bool
	}{
		{
			name:     "new address is better",
			newAddr:  "0x0000000000000000000000000000000000000001",
			oldAddr:  "0x0000000000000000000000000000000000000002",
			expected: true,
		},
		{
			name:     "old address is better",
			newAddr:  "0x0000000000000000000000000000000000000002",
			oldAddr:  "0x0000000000000000000000000000000000000001",
			expected: false,
		},
		{
			name:     "addresses are equal",
			newAddr:  "0x0000000000000000000000000000000000000001",
			oldAddr:  "0x0000000000000000000000000000000000000001",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &types.WorkerConfig{}
			attempts := int64(0)
			mu := &atomic.Value{}
			worker := NewWorker(config, &attempts, mu)
			result := worker.isBetterOptimized(tt.newAddr, tt.oldAddr)
			if result != tt.expected {
				t.Errorf("isBetterOptimized() = %v, want %v", result, tt.expected)
			}
		})
	}
}
