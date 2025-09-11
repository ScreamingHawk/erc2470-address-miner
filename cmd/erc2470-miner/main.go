package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/screa/erc2470-address-miner/internal/config"
	logpkg "github.com/screa/erc2470-address-miner/internal/logger"
	minerpkg "github.com/screa/erc2470-address-miner/pkg/miner"
	"github.com/screa/erc2470-address-miner/pkg/types"
	"github.com/spf13/cobra"
)

var (
	cfg    = config.NewConfig()
	logger *logpkg.Logger
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "erc2470-miner",
		Short: "High-performance ERC-2470 address miner",
		Long: `A performant command line utility for mining ERC-2470 addresses.
This tool uses keccak256 hashing to find addresses with specific patterns.`,
		Run: runMiner,
	}

	rootCmd.Flags().IntVarP(&cfg.Workers, "workers", "w", runtime.NumCPU(), "Number of worker goroutines")
	rootCmd.Flags().StringVarP(&cfg.Target, "target", "t", "", "Target address pattern (hex, case-insensitive)")
	rootCmd.Flags().StringVarP(&cfg.Prefix, "prefix", "p", "", "Address prefix to match")
	rootCmd.Flags().StringVarP(&cfg.Suffix, "suffix", "s", "", "Address suffix to match")
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().StringVarP(&cfg.LogFile, "log-file", "l", "", "Log file for progress tracking (default: stdout)")
	rootCmd.Flags().StringVarP(&cfg.Bytecode, "bytecode", "B", "", "Contract bytecode for CREATE2 address calculation (hex) (required)")
	rootCmd.Flags().StringVarP(&cfg.BytecodeFile, "bytecode-file", "F", "", "File containing contract bytecode (hex) (required)")
	rootCmd.Flags().IntVarP(&cfg.LogInterval, "log-interval", "i", 5, "Logging interval in seconds (default: 5)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMiner(cmd *cobra.Command, args []string) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	setupLogging()
	logger.Printf("Starting ERC-2470 address miner with %d workers...", cfg.Workers)
	logger.Printf("Target: %s", cfg.GetTargetDescription())
	logger.Printf("Factory address: 0xce0042B868300000d44A59004Da54A005ffdcf9f")
	if cfg.BytecodeFile != "" {
		logger.Printf("Bytecode file: %s", cfg.BytecodeFile)
	} else if cfg.Bytecode != "" {
		logger.Printf("Bytecode: %s...", cfg.Bytecode[:min(20, len(cfg.Bytecode))])
	}

	// Create miner and start mining
	miner := minerpkg.NewMiner(cfg, logger)

	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start mining in a goroutine
	resultChan := make(chan *types.Result, 1)
	go func() {
		result := miner.Mine()
		resultChan <- result
	}()

	// Wait for either completion or signal
	select {
	case result := <-resultChan:
		// Mining completed normally
		if result != nil {
			logger.Printf("🎉 Found match!")
			logger.Printf("Salt: 0x%s", result.Salt)
			logger.Printf("Address: %s", result.Address)
			logger.Printf("Attempts: %d", result.Attempts)
			logger.Printf("Duration: %v", result.Duration)

			// Calculate rate safely
			rate := 0.0
			if result.Duration.Seconds() > 0 {
				rate = float64(result.Attempts) / result.Duration.Seconds()
			}
			logger.Printf("Rate: %.2f hashes/sec", rate)
		} else {
			logger.Println("No match found.")
		}
	case <-sigChan:
		// Interrupted by Ctrl+C
		logger.Println("\nReceived interrupt signal (Ctrl+C). Stopping miners...")

		// Stop the miner
		miner.Stop()

		// Wait for mining to stop
		<-resultChan

		// If prefix is zeros, output the current best result
		if cfg.IsZeroPrefix() {
			bestResult := miner.GetBestResult()
			if bestResult != nil {
				logger.Printf("Current best result (lowest address found):")
				logger.Printf("Salt: 0x%s", bestResult.Salt)
				logger.Printf("Address: %s", bestResult.Address)
				logger.Printf("Attempts: %d", bestResult.Attempts)
				logger.Printf("Duration: %v", bestResult.Duration)

				// Calculate rate safely
				rate := 0.0
				if bestResult.Duration.Seconds() > 0 {
					rate = float64(bestResult.Attempts) / bestResult.Duration.Seconds()
				}
				logger.Printf("Rate: %.2f hashes/sec", rate)
			} else {
				logger.Println("No addresses found matching the zero prefix.")
			}
		} else {
			logger.Println("Mining stopped by user.")
		}
	}
}

func setupLogging() {
	if cfg.LogFile != "" {
		// Log to file
		file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		// Set global log output
		logger = logpkg.NewWriter(file)
		logger.SetFlags(log.LstdFlags | log.Lmicroseconds)
	} else {
		// Log to stdout
		logger = logpkg.New()
		logger.SetFlags(log.LstdFlags)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
