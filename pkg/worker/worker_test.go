package worker

import (
	"testing"

	"github.com/screa/erc2470-address-miner/pkg/types"
)

func TestNewWorker(t *testing.T) {
	config := &types.WorkerConfig{
		Initcode:      []byte{0x60, 0x80, 0x60, 0x40},
		InitcodeHash:  []byte{0x12, 0x34, 0x56, 0x78},
		FactoryBytes:  []byte{0xce, 0x00, 0x42, 0xb8},
		Prefix:        "0000",
		PrefixBytes:   []byte{0x00, 0x00},
		Create2Prefix: make([]byte, 21),
		Create2Suffix: make([]byte, 32),
	}

	attempts := int64(0)
	worker := NewWorker(config, &attempts)
	if worker == nil {
		t.Fatal("NewWorker returned nil")
	}

	if worker.config != config {
		t.Error("Config not set correctly")
	}
}

func TestWorkerMatches(t *testing.T) {
	// 20-byte address for byte-level matching
	addr20 := []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78}
	tests := []struct {
		name     string
		addr     []byte
		config   *types.WorkerConfig
		expected bool
	}{
		{
			name: "prefix match",
			addr: addr20,
			config: &types.WorkerConfig{
				Prefix:      "1234",
				PrefixBytes: []byte{0x12, 0x34},
				Create2Prefix: make([]byte, 21),
				Create2Suffix: make([]byte, 32),
			},
			expected: true,
		},
		{
			name: "suffix match",
			addr: addr20,
			config: &types.WorkerConfig{
				Suffix:      "5678",
				SuffixBytes: []byte{0x56, 0x78},
				Create2Prefix: make([]byte, 21),
				Create2Suffix: make([]byte, 32),
			},
			expected: true,
		},
		{
			name: "target match",
			addr: addr20,
			config: &types.WorkerConfig{
				Target:      "1234567890abcdef1234567890abcdef12345678",
				TargetBytes: addr20,
				Create2Prefix: make([]byte, 21),
				Create2Suffix: make([]byte, 32),
			},
			expected: true,
		},
		{
			name: "no match",
			addr: addr20,
			config: &types.WorkerConfig{
				Prefix:      "9999",
				PrefixBytes: []byte{0x99, 0x99},
				Create2Prefix: make([]byte, 21),
				Create2Suffix: make([]byte, 32),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := int64(0)
			worker := NewWorker(tt.config, &attempts)
			result := worker.matchesBytes(tt.addr)
			if result != tt.expected {
				t.Errorf("matchesBytes() = %v, want %v", result, tt.expected)
			}
		})
	}
}
