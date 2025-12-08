package types_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func CreateNDkimPubKey(domain string, pubKey string, version types.Version, keyType types.KeyType, count int) []types.DkimPubKey {
	var dkimPubKeys []types.DkimPubKey
	for i := 0; i < count; i++ {
		selector := uuid.NewString()
		hash, err := types.ComputePoseidonHash(pubKey)
		if err != nil {
			panic(err)
		}
		dkimPubKeys = append(dkimPubKeys, types.DkimPubKey{
			Domain:       domain,
			PubKey:       pubKey,
			PoseidonHash: []byte(hash.String()),
			Selector:     selector,
			Version:      version,
			KeyType:      keyType,
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
				Params: types.Params{DkimPubkeys: CreateNDkimPubKey("xion.burnt.com", "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB", types.Version_VERSION_DKIM1_UNSPECIFIED, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, 10)},
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
