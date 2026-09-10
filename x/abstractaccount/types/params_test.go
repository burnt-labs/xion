package types_test

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func TestValidateParams(t *testing.T) {
	addressHash := bytes.Repeat([]byte{0xA5}, sha256.Size)
	for _, tc := range []struct {
		desc   string
		params *types.Params
		expErr bool
	}{
		{
			desc:   "max gas is nil",
			params: &types.Params{},
			expErr: true,
		},
		{
			desc:   "max gas before is zero",
			params: &types.Params{MaxGasBefore: 0, MaxGasAfter: types.DefaultMaxGas},
			expErr: true,
		},
		{
			desc:   "max gas after is nil",
			params: &types.Params{MaxGasBefore: types.DefaultMaxGas, MaxGasAfter: 0},
			expErr: true,
		},
		{
			desc: "allow list is not empty when AllowAllCodeIDs is true",
			params: &types.Params{
				AllowAllCodeIDs: true,
				AllowedCodeIDs:  []uint64{1, 2, 3},
				MaxGasBefore:    types.DefaultMaxGas,
				MaxGasAfter:     types.DefaultMaxGas,
			},
			expErr: true,
		},
		{
			desc: "allow list contains zero code ID",
			params: &types.Params{
				AllowAllCodeIDs: false,
				AllowedCodeIDs:  []uint64{0, 1, 2},
				MaxGasBefore:    types.DefaultMaxGas,
				MaxGasAfter:     types.DefaultMaxGas,
			},
			expErr: true,
		},
		{
			desc: "allow list contains duplicate code IDs",
			params: &types.Params{
				AllowAllCodeIDs: false,
				AllowedCodeIDs:  []uint64{1, 2, 2, 3},
				MaxGasBefore:    types.DefaultMaxGas,
				MaxGasAfter:     types.DefaultMaxGas,
			},
			expErr: true,
		},
		{
			desc: "allow list contains unsorted code IDs",
			params: &types.Params{
				AllowAllCodeIDs: false,
				AllowedCodeIDs:  []uint64{1, 2, 3, 2},
				MaxGasBefore:    types.DefaultMaxGas,
				MaxGasAfter:     types.DefaultMaxGas,
			},
			expErr: true,
		},
		{
			desc: "valid params - all code IDs allowed",
			params: &types.Params{
				AllowAllCodeIDs: true,
				AllowedCodeIDs:  []uint64{},
				MaxGasBefore:    types.DefaultMaxGas,
				MaxGasAfter:     types.DefaultMaxGas,
			},
			expErr: false,
		},
		{
			desc: "valid params - not all code IDs allowed",
			params: &types.Params{
				AllowAllCodeIDs: false,
				AllowedCodeIDs:  []uint64{1, 2, 3},
				MaxGasBefore:    types.DefaultMaxGas,
				MaxGasAfter:     types.DefaultMaxGas,
			},
			expErr: false,
		},
		{
			desc: "address hash configured while registration remains paused",
			params: &types.Params{
				AllowAllCodeIDs:       true,
				MaxGasBefore:          types.DefaultMaxGas,
				MaxGasAfter:           types.DefaultMaxGas,
				AddressDerivationHash: addressHash,
			},
			expErr: false,
		},
		{
			desc: "registration enabled before code IDs are configured",
			params: &types.Params{
				AllowAllCodeIDs:     true,
				MaxGasBefore:        types.DefaultMaxGas,
				MaxGasAfter:         types.DefaultMaxGas,
				RegistrationEnabled: true,
			},
			expErr: true,
		},
		{
			desc: "invalid address hash length",
			params: &types.Params{
				AllowedCodeIDs:        []uint64{2},
				MaxGasBefore:          types.DefaultMaxGas,
				MaxGasAfter:           types.DefaultMaxGas,
				AddressDerivationHash: []byte("too-short"),
				RegistrationEnabled:   false,
			},
			expErr: true,
		},
		{
			desc: "valid enabled fixed-hash registration",
			params: &types.Params{
				AllowedCodeIDs:        []uint64{2},
				MaxGasBefore:          types.DefaultMaxGas,
				MaxGasAfter:           types.DefaultMaxGas,
				AddressDerivationHash: addressHash,
				RegistrationEnabled:   true,
			},
			expErr: false,
		},
	} {
		err := tc.params.Validate()

		if tc.expErr {
			require.Error(t, err, tc.desc)
		} else {
			require.NoError(t, err, tc.desc)
		}
	}
}

func TestRegistrationParams(t *testing.T) {
	addressHash := bytes.Repeat([]byte{0xA5}, sha256.Size)
	params, err := types.NewParamsWithAddressDerivationHash(
		false, []uint64{2}, types.DefaultMaxGas, types.DefaultMaxGas, addressHash,
	)
	require.NoError(t, err)
	require.True(t, params.RegistrationEnabled)
	require.True(t, params.RegistrationConfigured())
	require.Equal(t, addressHash, params.AddressDerivationHash)
	addressHash[0] = 0
	require.Equal(t, byte(0xA5), params.AddressDerivationHash[0])
	require.True(t, params.IsAllowed(2))

	_, err = types.NewParamsWithAddressDerivationHash(
		false, []uint64{1}, types.DefaultMaxGas, types.DefaultMaxGas, nil,
	)
	require.ErrorIs(t, err, types.ErrRegistrationNotConfigured)
}

func TestDeterminedAllowedCodeID(t *testing.T) {
	for _, tc := range []struct {
		allowAllCodeIDs bool
		allowedCodeIDs  []uint64
		codeID          uint64
		expAllowed      bool
	}{
		{
			allowAllCodeIDs: true,
			allowedCodeIDs:  []uint64{},
			codeID:          69420,
			expAllowed:      true,
		},
		{
			allowAllCodeIDs: false,
			allowedCodeIDs:  []uint64{12345, 42069, 69420},
			codeID:          69420,
			expAllowed:      true,
		},
		{
			allowAllCodeIDs: false,
			allowedCodeIDs:  []uint64{12345, 42069, 69420},
			codeID:          88888,
			expAllowed:      false,
		},
	} {
		params, err := types.NewParams(tc.allowAllCodeIDs, tc.allowedCodeIDs, types.DefaultMaxGas, types.DefaultMaxGas)
		require.NoError(t, err)

		allowed := params.IsAllowed(tc.codeID)
		require.Equal(t, tc.expAllowed, allowed)
	}
}
