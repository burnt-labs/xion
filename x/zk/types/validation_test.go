package types_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/burnt-labs/barretenberg-go/barretenberg"
	"github.com/stretchr/testify/require"
	"github.com/vocdoni/circom2gnark/parser"

	"github.com/burnt-labs/xion/x/zk/types"
)

const bn254FieldPrime = "21888242871839275222246405745257275088696311157297823662689037894645226208583"

// vkeyWith returns validVKeyJSON with one field rewritten.
func vkeyWith(t *testing.T, field string, value any) []byte {
	t.Helper()
	var vk map[string]any
	require.NoError(t, json.Unmarshal(validVKeyJSON, &vk))
	vk[field] = value
	bz, err := json.Marshal(vk)
	require.NoError(t, err)
	return bz
}

func TestMsgAddVKey_ValidateBasicLimits(t *testing.T) {
	validAuthority := getValidAuthority()

	t.Run("name too long", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority: validAuthority,
			Name:      strings.Repeat("n", types.MaxVKeyNameLen+1),
			VkeyBytes: encodedValidVKeyBytes(),
		}
		require.ErrorContains(t, msg.ValidateBasic(), "name length")
	})

	t.Run("description too long", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority:   validAuthority,
			Name:        "test-vkey",
			Description: strings.Repeat("d", types.MaxVKeyDescLen+1),
			VkeyBytes:   encodedValidVKeyBytes(),
		}
		require.ErrorContains(t, msg.ValidateBasic(), "description length")
	})

	t.Run("unsupported proof system", func(t *testing.T) {
		msg := &types.MsgAddVKey{
			Authority:   validAuthority,
			Name:        "test-vkey",
			VkeyBytes:   encodedValidVKeyBytes(),
			ProofSystem: types.ProofSystem(99),
		}
		require.ErrorContains(t, msg.ValidateBasic(), "unsupported proof_system")
	})
}

func TestMsgUpdateVKey_ValidateBasicLimits(t *testing.T) {
	validAuthority := getValidAuthority()

	t.Run("name too long", func(t *testing.T) {
		msg := &types.MsgUpdateVKey{
			Authority: validAuthority,
			Name:      strings.Repeat("n", types.MaxVKeyNameLen+1),
			VkeyBytes: encodedValidVKeyBytes(),
		}
		require.ErrorContains(t, msg.ValidateBasic(), "name length")
	})

	t.Run("description too long", func(t *testing.T) {
		msg := &types.MsgUpdateVKey{
			Authority:   validAuthority,
			Name:        "test-vkey",
			Description: strings.Repeat("d", types.MaxVKeyDescLen+1),
			VkeyBytes:   encodedValidVKeyBytes(),
		}
		require.ErrorContains(t, msg.ValidateBasic(), "description length")
	})

	t.Run("unsupported proof system", func(t *testing.T) {
		msg := &types.MsgUpdateVKey{
			Authority:   validAuthority,
			Name:        "test-vkey",
			VkeyBytes:   encodedValidVKeyBytes(),
			ProofSystem: types.ProofSystem(99),
		}
		require.ErrorContains(t, msg.ValidateBasic(), "unsupported proof_system")
	})
}

func TestValidateVKeyForProofSystem(t *testing.T) {
	t.Run("gnark key with gnark proof system", func(t *testing.T) {
		require.NoError(t, types.ValidateVKeyForProofSystem(loadGnarkVKeyTestdata(t), 0, types.ProofSystem_PROOF_SYSTEM_GROTH16_GNARK))
	})

	t.Run("ultrahonk key too small", func(t *testing.T) {
		err := types.ValidateVKeyForProofSystem([]byte("tiny"), 0, types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK)
		require.ErrorIs(t, err, barretenberg.ErrInvalidVKey)
	})

	t.Run("ultrahonk key from testdata", func(t *testing.T) {
		if strings.HasPrefix(barretenberg.Version(), "stub") {
			t.Skip("stub barretenberg library cannot validate UltraHonk keys")
		}
		vk, err := os.ReadFile("../keeper/testdata/barretenberg/vk")
		require.NoError(t, err)
		require.NoError(t, types.ValidateVKeyForProofSystem(vk, 0, types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK))
	})

	t.Run("unknown proof system", func(t *testing.T) {
		err := types.ValidateVKeyForProofSystem(validVKeyJSON, 0, types.ProofSystem(42))
		require.ErrorContains(t, err, "proof_system must be")
	})
}

func TestValidateVKeyBytesFieldCoordinates(t *testing.T) {
	var vk map[string]any
	require.NoError(t, json.Unmarshal(validVKeyJSON, &vk))

	t.Run("non integer coordinate", func(t *testing.T) {
		alpha := []any{"abc", "1", "1"}
		err := types.ValidateVKeyBytes(vkeyWith(t, "vk_alpha_1", alpha), 0)
		require.ErrorIs(t, err, types.ErrInvalidVKey)
		require.ErrorContains(t, err, "is not a valid integer")
	})

	g2Fields := []string{"vk_beta_2", "vk_gamma_2", "vk_delta_2"}
	for _, field := range g2Fields {
		t.Run(field+" coordinate outside the field", func(t *testing.T) {
			rows := []any{[]any{bn254FieldPrime, "1"}, []any{"1", "1"}, []any{"1", "0"}}
			err := types.ValidateVKeyBytes(vkeyWith(t, field, rows), 0)
			require.ErrorIs(t, err, types.ErrInvalidVKey)
			require.ErrorContains(t, err, ">= BN254 field prime")
		})
	}

	t.Run("IC coordinate outside the field", func(t *testing.T) {
		ic := []any{
			[]any{"1", bn254FieldPrime, "1"},
			[]any{"1", "1", "1"},
			[]any{"1", "1", "1"},
		}
		err := types.ValidateVKeyBytes(vkeyWith(t, "IC", ic), 0)
		require.ErrorIs(t, err, types.ErrInvalidVKey)
		require.ErrorContains(t, err, "IC[0][1]")
	})
}

func TestNewVKeyFromCircomRoundTrip(t *testing.T) {
	vk, err := parser.UnmarshalCircomVerificationKeyJSON(validVKeyJSON)
	require.NoError(t, err)

	vkey, err := types.NewVKeyFromCircom(vk, "circuit", "round trip")
	require.NoError(t, err)
	require.Equal(t, "circuit", vkey.Name)
	require.Equal(t, "round trip", vkey.Description)
	require.NoError(t, types.ValidateVKeyBytes(vkey.KeyBytes, 0))
}

func TestGasCostForSizeEdgeCases(t *testing.T) {
	t.Run("invalid params are rejected", func(t *testing.T) {
		params := types.DefaultParams()
		params.UploadChunkSize = 0
		_, err := params.GasCostForSize(1)
		require.ErrorIs(t, err, types.ErrInvalidParams)
	})

	t.Run("gas cost overflow is detected", func(t *testing.T) {
		params := types.DefaultParams()
		params.MaxVkeySizeBytes = types.MaxAllowedVKeySizeBytes
		params.UploadChunkSize = 1
		params.UploadChunkGas = 1 << 63
		_, err := params.GasCostForSize(types.MaxAllowedVKeySizeBytes)
		require.ErrorIs(t, err, types.ErrInvalidParams)
		require.ErrorContains(t, err, "overflow")
	})
}

func TestGenesisStateValidateInvalidParams(t *testing.T) {
	gs := types.GenesisState{Params: types.Params{MaxVkeySizeBytes: 1}}
	require.ErrorIs(t, gs.Validate(), types.ErrInvalidParams)
}
