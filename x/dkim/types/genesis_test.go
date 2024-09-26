package types_test

import (
	"encoding/base64"
	"testing"

	"github.com/burnt-labs/xion/x/dkim/types"
	"github.com/google/uuid"

	"github.com/stretchr/testify/require"
)

func CreateNDkimPubKey(domain string, pubKey string, version types.Version, keyType types.KeyType, count int) []types.DkimPubKey {
	var dkimPubKeys []types.DkimPubKey
	for i := 0; i < count; i++ {
		selector := uuid.NewString()
		dkimPubKeys = append(dkimPubKeys, types.DkimPubKey{
			Domain:   domain,
			PubKey:   pubKey,
			Selector: selector,
			Version:  version,
			KeyType:  keyType,
		})
	}
	return dkimPubKeys
}

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params:      types.DefaultParams(),
				DkimPubkeys: CreateNDkimPubKey("xion.burnt.com", base64.RawStdEncoding.EncodeToString([]byte("test-pub-key")), types.Version_DKIM1, types.KeyType_RSA, 10),
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
