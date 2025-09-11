package crypto

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"
)

const (
	// ERC-2470 Singleton Factory address
	FactoryAddress = "0xce0042B868300000d44A59004Da54A005ffdcf9f"
)

var (
	// Pre-primed factory address with 0xff prefix for CREATE2 optimization
	// This avoids the internal allocation in CreateAddress2
	primedFactoryAddress = []byte{0xff, 0xce, 0x00, 0x42, 0xb8, 0x68, 0x30, 0x00, 0x00, 0xd4, 0x4a, 0x59, 0x00, 0x4d, 0xa5, 0x4a, 0x00, 0x5f, 0xfd, 0xcf, 0x9f}
)

// CalculateCreate2Address calculates the CREATE2 address with minimal allocations
// This version uses pre-primed factory address to avoid internal allocations
func CalculateCreate2Address(initCodeHash []byte, saltBytes []byte) string {
	// Convert salt bytes to [32]byte array
	var saltArray [32]byte
	copy(saltArray[:], saltBytes)

	// Use optimized keccak256 with pre-primed factory address
	// This avoids the internal allocation in CreateAddress2 by pre-combining 0xff + factory address
	hash := keccak256Bytes(append(primedFactoryAddress, append(saltArray[:], initCodeHash...)...))

	// Extract last 20 bytes for address
	addressBytes := hash[12:]

	return toChecksumAddress(addressBytes)
}

// ---- helpers ----

func keccak256Bytes(b []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write(b)
	return h.Sum(nil)
}

// Keccak256 calculates the keccak256 hash of the input bytes
func Keccak256(data []byte) []byte {
	return keccak256Bytes(data)
}

// MustAddressBytes converts a hex address string to bytes
func MustAddressBytes(addr string) ([]byte, error) {
	h := strings.TrimSpace(addr)
	if len(h) >= 2 && (h[0:2] == "0x" || h[0:2] == "0X") {
		h = h[2:]
	}
	if len(h) != 40 {
		return nil, fmt.Errorf("invalid address length: got %d hex chars, want 40", len(h))
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("invalid address hex: %w", err)
	}
	return b, nil
}

// toChecksumAddress converts 20-byte address to EIP-55 checksummed string.
func toChecksumAddress(addr20 []byte) string {
	if len(addr20) != 20 {
		panic(errors.New("address must be 20 bytes"))
	}
	hexLower := hex.EncodeToString(addr20) // lowercase
	hash := keccak256Bytes([]byte(hexLower))
	// apply checksum casing
	var out strings.Builder
	out.Grow(2 + 40)
	out.WriteString("0x")
	for i, c := range hexLower {
		if c >= '0' && c <= '9' {
			out.WriteByte(byte(c))
			continue
		}
		// each nibble of the hash decides case of corresponding hex char
		// hash nibble at position i
		n := (hash[i/2] >> uint(4*(1-i%2))) & 0xF
		if n >= 8 {
			out.WriteByte(byte(strings.ToUpper(string(c))[0]))
		} else {
			out.WriteByte(byte(c))
		}
	}
	return out.String()
}
