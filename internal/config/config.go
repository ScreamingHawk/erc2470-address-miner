package config

import (
	"encoding/hex"
	"errors"
	"os"
	"runtime"
	"strings"
)

// Errors
var (
	ErrNoTargetSpecified   = errors.New("must specify either --target, --prefix, or --suffix")
	ErrNoBytecodeSpecified = errors.New("must specify either --bytecode or --bytecode-file")
)

// Config holds the application configuration
type Config struct {
	Workers      int
	Target       string
	Prefix       string
	Suffix       string
	Verbose      bool
	LogFile      string
	Bytecode     string
	BytecodeFile string
	LogInterval  int // Logging interval in seconds
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		Workers:     runtime.NumCPU(),
		LogInterval: 5, // Default 5 seconds
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Target == "" && c.Prefix == "" && c.Suffix == "" {
		return ErrNoTargetSpecified
	}
	if c.Bytecode == "" && c.BytecodeFile == "" {
		return ErrNoBytecodeSpecified
	}
	return nil
}

// GetTargetDescription returns a human-readable description of the target
func (c *Config) GetTargetDescription() string {
	if c.Target != "" {
		return "exact match: " + c.Target
	}
	if c.Prefix != "" {
		return "prefix: " + c.Prefix
	}
	if c.Suffix != "" {
		return "suffix: " + c.Suffix
	}
	return "unknown"
}

// IsZeroPrefix returns true if the prefix is a series of 0's
func (c *Config) IsZeroPrefix() bool {
	if c.Prefix == "" {
		return false
	}

	// Remove 0x prefix if present
	prefix := c.Prefix
	if len(prefix) > 2 && prefix[:2] == "0x" {
		prefix = prefix[2:]
	}

	// Check if all characters are '0'
	for _, char := range prefix {
		if char != '0' {
			return false
		}
	}

	return true
}

// GetBytecode returns the bytecode to use for address calculation
func (c *Config) GetBytecode() ([]byte, error) {
	// Check if bytecode file is specified
	if c.BytecodeFile != "" {
		return readBytecodeFromFile(c.BytecodeFile)
	}

	// Check if bytecode is provided directly
	if c.Bytecode != "" {
		// Remove 0x prefix if present
		code := c.Bytecode
		if len(code) > 2 && code[:2] == "0x" {
			code = code[2:]
		}

		// Decode hex string to bytes
		bytes, err := hex.DecodeString(code)
		if err != nil {
			return nil, err
		}
		return bytes, nil
	}

	// This should not happen if validation passes
	return nil, ErrNoBytecodeSpecified
}

// readBytecodeFromFile reads bytecode from a file
func readBytecodeFromFile(filename string) ([]byte, error) {
	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Convert to string and clean up
	code := string(content)
	code = strings.TrimSpace(code)

	// Remove 0x prefix if present
	if len(code) > 2 && code[:2] == "0x" {
		code = code[2:]
	}

	// Ensure even length by padding with 0 if necessary
	if len(code)%2 != 0 {
		code = code + "0"
	}

	// Decode hex string to bytes
	bytes, err := hex.DecodeString(code)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
