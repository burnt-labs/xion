package keeper

import (
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/iden3/go-iden3-crypto/poseidon"
)

const TX_BODY_MAX_BYTES = 512

func fromLeBytesModOrder(bytes []byte) fr.Element {
	var field fr.Element
	field.SetBytes(ToLittleEndian(bytes)) // Little-endian, modulo BN254 scalar field order.
	return fr.Element(field)
}

// padBytes pads the input bytes to the specified length by appending zeros.
func padBytes(bytes []byte, length int) []byte {
	padded := make([]byte, length)
	copy(padded, bytes) // Copy input bytes; remaining bytes are zero-initialized.
	return padded
}

// packBytesIntoFields converts bytes into fr.Element field elements, using 31 bytes per element.
func packBytesIntoFields(bytes []byte) []fr.Element {
	var fields []fr.Element
	for i := 0; i < len(bytes); i += 31 {
		end := i + 31
		if end > len(bytes) {
			end = len(bytes)
		}
		chunk := bytes[i:end]
		field := fromLeBytesModOrder(chunk)
		fields = append(fields, field)
	}
	return fields
}

// calculateTxBodyCommitment computes the commitment for a transaction body.
func CalculateTxBodyCommitment(tx string) (fr.Element, error) {
	// Convert string to bytes.
	txBytes := []byte(tx)

	// Pad the transaction bytes.
	paddedTxBytes := padBytes(txBytes, TX_BODY_MAX_BYTES)

	// Pack bytes into field elements.
	txFields := packBytesIntoFields(paddedTxBytes)

	// Initialize commitment to zero.
	var commitment fr.Element // Zero by default for fr.Element.

	// Process chunks of 16 field elements.
	chunkSize := 16
	for i := 0; i < len(txFields); i += chunkSize {
		end := i + chunkSize
		if end > len(txFields) {
			end = len(txFields)
		}
		chunk := txFields[i:end]

		// Convert chunk to []*big.Int for Poseidon.
		chunkBigInts := make([]*big.Int, len(chunk))
		for j, field := range chunk {
			chunkBigInts[j] = field.BigInt(new(big.Int)) // Convert fr.Element to *big.Int.
		}

		// Compute chunk commitment.
		chunkCommitmentBI, err := poseidon.Hash(chunkBigInts)
		if err != nil {
			return fr.Element{}, fmt.Errorf("failed to hash chunk %d: %w", i/chunkSize, err)
		}
		var chunkCommitment fr.Element
		chunkCommitment.SetBigInt(chunkCommitmentBI) // Convert *big.Int to fr.Element.

		// Update commitment.
		if i == 0 {
			commitment = fr.Element(chunkCommitment)
		} else {
			combined := []*big.Int{commitment.BigInt(new(big.Int)), chunkCommitment.BigInt(new(big.Int))}
			commitmentBI, err := poseidon.Hash(combined)
			if err != nil {
				return fr.Element{}, fmt.Errorf("failed to hash commitment at chunk %d: %w", i/chunkSize, err)
			}
			var newCommitment fr.Element
			newCommitment.SetBigInt(commitmentBI)
			commitment = fr.Element(newCommitment)
		}
	}

	return commitment, nil
}

func ToLittleEndian(b []byte) []byte {
	le := make([]byte, len(b))
	for i, v := range b {
		le[len(b)-1-i] = v
	}
	return le
}
