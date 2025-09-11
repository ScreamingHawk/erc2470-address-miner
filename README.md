# ERC-2470 Address Miner

A high-performance command line utility for mining ERC-2470 addresses using keccak256 hashing.

## Features

- **High Performance**: Optimized Go implementation with parallel processing
- **Memory Efficient**: Reduced garbage collection pressure with memory pools
- **Flexible Matching**: Support for exact matches, prefixes, and suffixes
- **Cross-Platform**: Pre-built binaries for Windows, Linux, and macOS
- **Docker Support**: Containerized deployment

## Installation

### Pre-built Binaries

Download the latest release from the [Releases](https://github.com/screa/erc2470-address-miner/releases) page.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/screa/erc2470-address-miner.git
cd erc2470-address-miner

# Install dependencies and build
make deps
make build

# Or build for all platforms
make build-all
```

### Docker

```bash
# Build and run with Docker
docker build -t erc2470-miner .
docker run --rm -it -v ./bytecode.txt:/home/miner/bytecode.txt erc2470-miner --prefix 0000 --workers 8 --bytecode-file bytecode.txt
```

## Usage

**Important**: You must provide either `--bytecode` or `--bytecode-file` as the miner requires contract bytecode for CREATE2 address calculation.

### Basic Usage

```bash
# Mine for a specific prefix
./erc2470-miner --prefix 0000 --bytecode 0x608060405234801561001057600080fd5b50600436106100365760003560e01c8063

# Mine for a specific suffix
./erc2470-miner --suffix 0000 --bytecode-file bytecode.txt

# Mine for an exact address match
./erc2470-miner --target 0x1234567890abcdef1234567890abcdef12345678 --bytecode 0x608060405234801561001057600080fd5b50600436106100365760003560e01c8063
```

### Advanced Options

```bash
# Verbose output with progress reporting
./erc2470-miner --prefix 0000 --workers 8 --verbose --bytecode 0x608060405234801561001057600080fd5b50600436106100365760003560e01c8063

# Log progress to file with custom interval
./erc2470-miner --prefix 0000 --workers 8 --log-file mining.log --log-interval 10 --verbose --bytecode 0x608060405234801561001057600080fd5b50600436106100365760003560e01c8063
```

### Command Line Options

| Option            | Short | Description                                                        | Default   |
| ----------------- | ----- | ------------------------------------------------------------------ | --------- |
| `--workers`       | `-w`  | Number of worker goroutines                                        | CPU count |
| `--target`        | `-t`  | Target address pattern (exact match)                               | -         |
| `--prefix`        | `-p`  | Address prefix to match                                            | -         |
| `--suffix`        | `-s`  | Address suffix to match                                            | -         |
| `--verbose`       | `-v`  | Verbose output with progress                                       | false     |
| `--log-file`      | `-l`  | Log file for progress tracking (default: stdout)                   | -         |
| `--log-interval`  | `-i`  | Logging interval in seconds (default: 5)                           | 5         |
| `--bytecode`      | `-B`  | Contract bytecode for CREATE2 address calculation (hex) (required) | -         |
| `--bytecode-file` | `-F`  | File containing contract bytecode (hex) (required)                 | -         |

## Examples

### Mining for a Vanity Address

```bash
# Find an address starting with "0000"
./erc2470-miner --prefix 0000 --workers 8 --verbose --bytecode 0x608060405234801561001057600080fd5b50600436106100365760003560e01c8063

# Output:
# 2024-01-15 10:30:00 Starting ERC-2470 address miner with 8 workers...
# 2024-01-15 10:30:00 Target: prefix: 0000
# 2024-01-15 10:30:10 Progress: 5000000 attempts, 500000.00 hashes/sec
# 2024-01-15 10:30:25 Found potential match: 0x00001234567890abcdef1234567890abcdef123456
#
# 🎉 Found match!
# Salt: 0000000100000000000000000000000000000000000000000000000000000000
# Address: 0x00001234567890abcdef1234567890abcdef123456
# Attempts: 12345678
# Duration: 25.123s
# Rate: 491234.56 hashes/sec
```

### Using Bytecode Files

```bash
# Create a bytecode file
echo "608060405234801561001057600080fd5b50600436106100365760003560e01c8063" > bytecode.txt

# Use the bytecode file
./erc2470-miner --prefix 0000 --bytecode-file bytecode.txt --workers 8
```

## Development

### Building

```bash
# Install dependencies
make deps

# Build the application
make build

# Build for all platforms
make build-all

# Run tests
make test

# Clean build artifacts
make clean
```

### Testing

```bash
# Run unit tests
go test -v ./...

# Run tests with race detection
go test -race ./...
```

### Available Make Targets

```bash
make help
```

Available targets:

- `build` - Build the application
- `build-all` - Build for multiple platforms (Linux, macOS, Windows)
- `deps` - Install dependencies
- `test` - Run tests
- `clean` - Clean build artifacts
- `install` - Install binary to GOPATH/bin
- `help` - Show this help message

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Go](https://golang.org/) for the excellent runtime and crypto libraries
- [Cobra](https://github.com/spf13/cobra) for the CLI framework
- [Docker](https://www.docker.com/) for containerization support
