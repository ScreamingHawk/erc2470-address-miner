package miner

import (
	"testing"

	"github.com/screa/erc2470-address-miner/internal/config"
	"github.com/screa/erc2470-address-miner/internal/logger"
)

func TestNewMiner(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Prefix = "0000"
	cfg.Bytecode = "608060405234801561001057600080fd5b50600436106100365760003560e01c8063"
	logger := logger.New()
	miner := NewMiner(cfg, logger)
	if miner == nil {
		t.Fatal("NewMiner returned nil")
	}

	if miner.config != cfg {
		t.Error("Config not set correctly")
	}
}

func TestMinerIsBetter(t *testing.T) {
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
			cfg := config.NewConfig()
			cfg.Bytecode = "608060405234801561001057600080fd5b50600436106100365760003560e01c8063"
			logger := logger.New()
			miner := NewMiner(cfg, logger)
			result := miner.isBetter(tt.newAddr, tt.oldAddr)
			if result != tt.expected {
				t.Errorf("isBetter() = %v, want %v", result, tt.expected)
			}
		})
	}
}
