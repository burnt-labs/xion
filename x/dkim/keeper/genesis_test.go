package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestGenesis(t *testing.T) {
	f := SetupTest(t)
	const PubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	hash, err := types.ComputePoseidonHash(PubKey)
	require.NoError(t, err)
	testCases := []struct {
		name    string
		request *types.GenesisState
		err     bool
	}{
		{
			name:    "success, only default params",
			request: &types.GenesisState{},
			err:     false,
		},
		{
			name: "success, with dkim records",
			request: &types.GenesisState{
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:       "x.com",
						Selector:     "test",
						PubKey:       PubKey,
						PoseidonHash: []byte(hash.String()),
					},
				},
			},
			err: false,
		},
		{
			name: "fail, invalid dkim record",
			request: &types.GenesisState{
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:       "x.com",
						Selector:     "test",
						PubKey:       "invalid",
						PoseidonHash: hash.Bytes(),
					},
				},
			},
			err: true,
		},
		{
			name: "fail, invalid data",
			request: &types.GenesisState{
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:       "x.com",
						PubKey:       "test",
						Selector:     "test",
						PoseidonHash: []byte("test"),
					},
				},
			},
			err: true,
		},
	}
	// this line is used by starport scaffolding # genesis/test/state
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err {
				require.Error(t, f.k.InitGenesis(f.ctx, tc.request))
			} else {
				require.NoError(t, f.k.InitGenesis(f.ctx, tc.request))
			}
		})
	}

	got := f.k.ExportGenesis(f.ctx)
	require.NotNil(t, got)

	// this line is used by starport scaffolding # genesis/test/assert
}
