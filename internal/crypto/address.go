package crypto

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

const (
	// ERC-2470 Singleton Factory address
	FactoryAddress = "0xce0042B868300000d44A59004Da54A005ffdcf9f"
)

// CalculateCreate2Address calculates the CREATE2 address for the given init code and salt.
// - salt may be hex (with or without 0x) of up to 32 bytes (left-padded with zeros if shorter).
// - if salt is not valid hex, we keccak256(salt) to get a 32-byte salt.
func CalculateCreate2Address(initCode []byte, salt string) (string, error) {
	initCodeHash := keccak256Bytes(initCode)
	return CalculateCreate2AddressWithHash(initCodeHash, salt)
}

// CalculateCreate2AddressWithHash calculates the CREATE2 address using a pre-computed initcode hash.
// This is more efficient when the same initcode is used repeatedly.
func CalculateCreate2AddressWithHash(initCodeHash []byte, salt string) (string, error) {
	factoryBytes, err := mustAddressBytes(FactoryAddress)
	if err != nil {
		return "", err
	}

	saltBytes, err := normalizeSalt(salt)
	if err != nil {
		return "", err
	}

	// Convert factory bytes to common.Address
	factoryAddress := common.BytesToAddress(factoryBytes)
	
	// Convert salt bytes to [32]byte array
	var saltArray [32]byte
	copy(saltArray[:], saltBytes)

	// Use go-ethereum's CreateAddress2 function with pre-computed hash
	address := crypto.CreateAddress2(factoryAddress, saltArray, initCodeHash)
	
	return toChecksumAddress(address.Bytes()), nil
}

// CalculateCreate2AddressOptimized calculates the CREATE2 address using pre-computed factory bytes and initcode hash.
// This is the most efficient version for repeated calculations.
func CalculateCreate2AddressOptimized(factoryBytes, initCodeHash []byte, salt string) (string, error) {
	saltBytes, err := normalizeSalt(salt)
	if err != nil {
		return "", err
	}

	// Convert factory bytes to common.Address
	factoryAddress := common.BytesToAddress(factoryBytes)
	
	// Convert salt bytes to [32]byte array
	var saltArray [32]byte
	copy(saltArray[:], saltBytes)

	// Use go-ethereum's CreateAddress2 function with pre-computed values
	address := crypto.CreateAddress2(factoryAddress, saltArray, initCodeHash)
	
	return toChecksumAddress(address.Bytes()), nil
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

func mustAddressBytes(addr string) ([]byte, error) {
	h := strip0x(strings.TrimSpace(addr))
	if len(h) != 40 {
		return nil, fmt.Errorf("invalid address length: got %d hex chars, want 40", len(h))
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("invalid address hex: %w", err)
	}
	return b, nil
}

// MustAddressBytes is a public version of mustAddressBytes
func MustAddressBytes(addr string) ([]byte, error) {
	return mustAddressBytes(addr)
}

// normalizeSalt returns a 32-byte salt.
// If salt parses as hex (with optional 0x), we left-pad with zeros to 32 bytes.
// If it doesn't parse as hex, we use keccak256(salt) to obtain 32 bytes.
func normalizeSalt(s string) ([]byte, error) {
	raw := strings.TrimSpace(s)
	hexStr := strip0x(raw)

	// Try hex decode first
	if isHexString(hexStr) {
		// ensure even length
		if len(hexStr)%2 == 1 {
			hexStr = "0" + hexStr
		}
		b, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid salt hex: %w", err)
		}
		if len(b) > 32 {
			// Spec requires exactly 32 bytes; truncate left (keep rightmost 32 bytes) is safer than silent error
			b = b[len(b)-32:]
		} else if len(b) < 32 {
			pad := make([]byte, 32-len(b))
			b = append(pad, b...)
		}
		return b, nil
	}

	// Not hex: keccak256 to 32 bytes
	return keccak256Bytes([]byte(raw)), nil
}

func strip0x(s string) string {
	if len(s) >= 2 && (s[0:2] == "0x" || s[0:2] == "0X") {
		return s[2:]
	}
	return s
}

// simple check: all chars must be [0-9a-fA-F]
func isHexString(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
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
