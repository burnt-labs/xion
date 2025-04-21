package keeper

import (
	b64 "encoding/base64"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vocdoni/circom2gnark/parser"
	"gotest.tools/assert"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/iden3/go-iden3-crypto/poseidon"
)

const TX_BODY_MAX_BYTES = 512

func TestVerify(t *testing.T) {
	inputs := []string{"14090993295699715988323746151124968754442881127975012622569574997457449326200", "17159366307350401517208657413587014704131356894001302493847352957889395820464", "6632353713085157925504008443078919716322386156160602218536961028046468237192"}

	vkey64 := "eyJ2a19hbHBoYV8xIjpbIjIwNDkxMTkyODA1MzkwNDg1Mjk5MTUzMDA5NzczNTk0NTM0OTQwMTg5MjYxODY2MjI4NDQ3OTE4MDY4NjU4NDcxOTcwNDgxNzYzMDQyIiwiOTM4MzQ4NTM2MzA1MzI5MDIwMDkxODM0NzE1NjE1NzgzNjU2NjU2Mjk2Nzk5NDAzOTcxMjI3MzQ0OTkwMjYyMTI2NjE3ODU0NTk1OCIsIjEiXSwidmtfYmV0YV8yIjpbWyI2Mzc1NjE0MzUxNjg4NzI1MjA2NDAzOTQ4MjYyODY4OTYyNzkzNjI1NzQ0MDQzNzk0MzA1NzE1MjIyMDExNTI4NDU5NjU2NzM4NzMxIiwiNDI1MjgyMjg3ODc1ODMwMDg1OTEyMzg5Nzk4MTQ1MDU5MTM1MzUzMzA3MzQxMzE5Nzc3MTc2ODY1MTQ0MjY2NTc1MjI1OTM5NzEzMiJdLFsiMTA1MDUyNDI2MjYzNzAyNjIyNzc1NTI5MDEwODIwOTQzNTY2OTc0MDk4MzU2ODAyMjA1OTA5NzE4NzMxNzExNDAzNzEzMzEyMDY4NTYiLCIyMTg0NzAzNTEwNTUyODc0NTQwMzI4ODIzMjY5MTE0NzU4NDcyODE5MTE2MjczMjI5OTg2NTMzODM3NzE1OTY5MjM1MDA1OTEzNjY3OSJdLFsiMSIsIjAiXV0sInZrX2dhbW1hXzIiOltbIjEwODU3MDQ2OTk5MDIzMDU3MTM1OTQ0NTcwNzYyMjMyODI5NDgxMzcwNzU2MzU5NTc4NTE4MDg2OTkwNTE5OTkzMjg1NjU1ODUyNzgxIiwiMTE1NTk3MzIwMzI5ODYzODcxMDc5OTEwMDQwMjEzOTIyODU3ODM5MjU4MTI4NjE4MjExOTI1MzA5MTc0MDMxNTE0NTIzOTE4MDU2MzQiXSxbIjg0OTU2NTM5MjMxMjM0MzE0MTc2MDQ5NzMyNDc0ODkyNzI0Mzg0MTgxOTA1ODcyNjM2MDAxNDg3NzAyODA2NDkzMDY5NTgxMDE5MzAiLCI0MDgyMzY3ODc1ODYzNDMzNjgxMzMyMjAzNDAzMTQ1NDM1NTY4MzE2ODUxMzI3NTkzNDAxMjA4MTA1NzQxMDc2MjE0MTIwMDkzNTMxIl0sWyIxIiwiMCJdXSwidmtfZGVsdGFfMiI6W1siNTY4MTAwNjE2NDMwODI1MTk1MzAwMjExMzkyNTU4NTQ5MDE1OTM4MjM2NTQ1OTkwMDYxNjY5NTA2OTczODA5OTYxNjIyMTAwNzQ0MCIsIjE3ODQzNzI1MzIwMjMyODUyMzQ3ODgxMTI5MTI2MDUwMTY0MTg3ODU4NTMwODI2Nzg0MzAxNjI4NDE4MzM2ODg5NzI0MDQ3NTQ1MjYiXSxbIjEzMjA3MzcyNzAwMzc0OTUxMjE3OTcyNzM0MzA1OTg3OTg0MjExMjk5NTQxOTUzMjcyMTk1ODA0NDAzMDE5Mjk0Mjg1MDE1NDMyMjMwIiwiMTEyMDgwOTg4MTU5MzE0OTgyNTg1NjYzMzU5NTMyNTU3Njg5MTU3ODEwMjQ1Njg0MzE0NzI5NzMwMDM3NDAzNTEyNDIxNDUyNzQ4ODIiXSxbIjEiLCIwIl1dLCJJQyI6W1siMTM1NTI3OTYxNTkzMjE1MDA0NDY1NDIwOTIzMzM5MjU3ODk2MjcyMjE3NjM5OTU5MTcwNTc3MDUwMjUwMjQ3MzY4NDU3MjE5NDIzNTMiLCI5NjgwOTg1Mjg1MDQwMjMwNzUxNjQ0NDk1ODkzMjE1MzgyNjc2MDgxNTM1ODc2MzYzNTUzNjYwMzc0MTE1NTg4ODYyMDYwNDg3MDM0IiwiMSJdLFsiMTUyNTQ3NzY3NzI2MTA1MzMzMjc0ODg4MTQyODE5MzEyODc2NjU4OTM5NDMxNzc5NzczMTAxNjgwNTY3MjIzNjgyODE3Mjg2MTg2NjIiLCIxNDM2NzY1NzcwMTI0OTU0NzkxODUxMzY3NTUzMzA5MTI4NjM4MzM1ODE4NTY2Mjg2OTg4NzMxNjQ2NjgxMzA2MjE0MjY3NDQ5OTI2MCIsIjEiXSxbIjE0ODY2ODU5MTc3NzU4NjM1MDMwMDc5MjI2MzQxODYyNjAxMTEyOTgzNDg3ODM4NTExNjkwNTY3ODk1NTc0MDMwNjcyMjEyNDQxNjc2IiwiMTEzMTQ1NDIyOTM1MzM5NzMzMjg0MTY0NDE2NjMxNzYwMzIzOTk4NDU1MTkzMDA0NDY4OTQ0MDc1MjAwODM0MTIwMzY4ODgzNDI5OCIsIjEiXSxbIjE0NDUyOTAyNDgyODI4NTU4MjcwMjk4Mzc0NzM3NzA1NjU4MzUxMjcwNzg4NDI4MDE5NTg1MjQxMDI2OTAxMzMxMzY4ODU4NTI3OTQ4IiwiMTkwOTc5OTc3NzQ2MDY1MjM1MjcxNTg1Mjk1NTI2NDU5NjEyNTcxNTAxODkzNTczNTc2MDEwMzUwNzgxMTE2NDM5NTMzODUxNzEzODQiLCIxIl1dfQ=="
	vkey64Bz, err := b64.StdEncoding.DecodeString(vkey64)
	require.NoError(t, err)

	snarkVk, err := parser.UnmarshalCircomVerificationKeyJSON(vkey64Bz)
	require.NoError(t, err)

	proof64 := "ewogICJwaV9hIjogWwogICAgIjE3ODUwMjk5MjUyMzk1OTE2MTAzNDQ1NjIzMDI0NDA2OTk3MTg3MzI1NDIwMTYyMDQxNTYyOTQxMjg4OTAzMzI2NTYxNTcwNzI2OTEzIiwKICAgICIxMTAwNTYyNjA4NjY3MjQwMjM3NzA5OTE2NjYyMzg3MTUxNzU5NDUyOTY0NTcxMDA3MjY0ODc1Njc0Njk0Mzk3NzgwOTk0MDgxOTIxNiIsCiAgICAiMSIKICBdLAogICJwaV9iIjogWwogICAgWwogICAgICAiMTE0OTIyODU1Mzg4NDkzNTA0MDk1Njg1NDY4OTQ3MjgzNTE5Mjk3MzI2OTY2NzI1NTUxNzc4NzYxNDgzMzA0MjA3MzY2MTQ2ODE5MDEiLAogICAgICAiMTIxMzExOTk0ODExNDQ3MjUwNTA2NzIyNTM2ODA1NzU0NzMxNDE5ODkyNzE5ODMzMDQ3MTkzODM3OTAwNjU3NTI0ODI0MDExODQ3NDUiCiAgICBdLAogICAgWwogICAgICAiMTMwNDk2MDgyMzMxMjQwNDY3ODk4OTYwNTg3MTY2OTc4MjI1OTM4Mjg4NjgzMjYyODc5ODUwMzExMzY5MDg4Mjk1Nzk0Mzk0Mjc2MjIiLAogICAgICAiNjk5Njc4NzU3NzY1NTk5NTA3NjU0OTczNDQ2MDQ2NzIyODM4NzA2ODU3NTg5NTA4MDcxNzU4NTk4NjkyMjk0ODQ4MTg4MTk4NTMwMSIKICAgIF0sCiAgICBbCiAgICAgICIxIiwKICAgICAgIjAiCiAgICBdCiAgXSwKICAicGlfYyI6IFsKICAgICI0ODcxMDAyNzMzNzM4Nzg2NTkxMzc5NDkyODk2Mjk5MDY2NzczOTczMTg1MjI3NzU2ODkxNzM0Mjc5Mzk1NTQwNjM0MDg3MTA3ODk2IiwKICAgICIxOTcxNjY1OTc2OTI4MjY4NDM3OTg3NzczMTE3NDg2MjcxOTc4NzIyMzEzMTI3NTI3MTI0MTY2NzQyNjIzMjk3MzEyODQ4MzI5OTc5NiIsCiAgICAiMSIKICBdLAogICJwcm90b2NvbCI6ICJncm90aDE2IiwKICAiY3VydmUiOiAiYm4xMjgiCn0="
	proofData, err := b64.StdEncoding.DecodeString(proof64)
	require.NoError(t, err)

	// Unmarshal the JSON data
	snarkProof, err := parser.UnmarshalCircomProofJSON(proofData)
	require.NoError(t, err)

	gnarkProof, err := parser.ConvertCircomToGnark(snarkProof, snarkVk, inputs)
	require.NoError(t, err)

	verified, err := parser.VerifyProof(gnarkProof)
	require.NoError(t, err)
	require.True(t, verified, "proof verification failed")
	t.Log("success")
}

func TestCalculateTxBodyCommitment(t *testing.T) {
	txb64 := "CqIBCp8BChwvY29zbW9zLmJhbmsudjFiZXRhMS5Nc2dTZW5kEn8KP3hpb24xczN1YWU1MDUydTVnNzd3ZmxheHd5aDJ1dHg5OWU3aDZjNjJhNjI5M3J5cnh0Z3F2djhncXhlZ3J0axIreGlvbjFxYWYyeGZseDVqM2FndGx2cWs1dmhqcGV1aGw2ZzQ1aHhzaHdxahoPCgV1eGlvbhIGMTAwMDAwEhYSFAoOCgV1eGlvbhIFNjAwMDAQwJoMGgZ4aW9uLTEgCw=="
	decodedTx, err := b64.StdEncoding.DecodeString(txb64)
	require.NoError(t, err)
	received, err := calculateTxBodyCommitment(string(decodedTx))
	require.NoError(t, err)

	assert.Equal(t, "18446744073709551615", received.String())
}

func fromLeBytesModOrder(bytes []byte) fr.Element {
	var field fr.Element
	field.SetBytes(bytes) // Little-endian, modulo BN254 scalar field order.
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
func calculateTxBodyCommitment(tx string) (fr.Element, error) {
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
