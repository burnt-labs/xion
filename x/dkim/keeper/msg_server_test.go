package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/orm/types/ormerrors"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestParams(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	testCases := []struct {
		name    string
		request *types.MsgUpdateParams
		err     bool
	}{
		{
			name: "fail; invalid authority",
			request: &types.MsgUpdateParams{
				Authority: f.addrs[0].String(),
				Params:    types.DefaultParams(),
			},
			err: true,
		},
		{
			name: "success",
			request: &types.MsgUpdateParams{
				Authority: f.govModAddr,
				Params:    types.DefaultParams(),
			},
			err: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			_, err := f.msgServer.UpdateParams(f.ctx, tc.request)

			if tc.err {
				require.Error(err)
			} else {
				require.NoError(err)

				r, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
				require.NoError(err)

				require.EqualValues(tc.request.Params, *(r.Params))
			}
		})
	}
}

func TestAddDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)
	const PubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"

	testCases := []struct {
		name    string
		request *types.MsgAddDkimPubKeys
		err     bool
	}{
		{
			name: "fail; invalid authority",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.addrs[0].String(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   PubKey,
						Selector: "zkemail",
					},
				},
			},
			err: true,
		},
		{
			name: "fail; invalid keytype",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.addrs[0].String(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   PubKey,
						Selector: "zkemail",
						KeyType:  2,
					},
				},
			},
			err: true,
		},
		{
			name: "fail; invalid version",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.addrs[0].String(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   PubKey,
						Selector: "zkemail",
						Version:  2,
					},
				},
			},
			err: true,
		},
		{
			name: "fail; invalid pubkey",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.govModAddr,
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   "123456789",
						Selector: "zkemail",
						Version:  2,
					},
				},
			},
			err: true,
		},
		{
			name: "success",
			request: &types.MsgAddDkimPubKeys{
				Authority: f.govModAddr,
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   PubKey,
						Selector: "zkemail",
					},
				},
			},
			err: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {

			err := tc.request.ValidateBasic()
			if tc.err {
				require.Error(err)
			} else {
				_, err = f.msgServer.AddDkimPubKey(f.ctx, tc.request)
				require.NoError(err)

				r, err := f.queryServer.DkimPubKey(f.ctx, &types.QueryDkimPubKeyRequest{
					Domain:   tc.request.DkimPubkeys[0].Domain,
					Selector: tc.request.DkimPubkeys[0].Selector,
				})
				require.NoError(err)

				require.EqualValues(tc.request.DkimPubkeys[0].PubKey, r.DkimPubkey.PubKey)
				require.EqualValues(types.Version_DKIM1, r.DkimPubkey.Version)
				require.EqualValues(types.KeyType_RSA, r.DkimPubkey.KeyType)
			}
		})
	}
}

func TestRemoveDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	const domain = "xion.burnt.com"
	const PubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	selector := "zkemail"

	_, err := f.msgServer.AddDkimPubKey(f.ctx, &types.MsgAddDkimPubKeys{
		Authority: f.govModAddr,
		DkimPubkeys: []types.DkimPubKey{
			{
				Domain:   domain,
				PubKey:   PubKey,
				Selector: selector,
			},
		},
	})
	require.NoError(err)

	testCases := []struct {
		name    string
		request *types.MsgRemoveDkimPubKey
		err     bool
	}{
		{
			name: "fail; invalid authority",
			request: &types.MsgRemoveDkimPubKey{
				Authority: f.addrs[0].String(),
				Domain:    domain,
				Selector:  selector,
			},
			err: true,
		},
		{
			name: "fail: remove non existing key",
			request: &types.MsgRemoveDkimPubKey{
				Authority: f.govModAddr,
				Domain:    domain,
				Selector:  "non-existing",
			},
			err: true,
		},
		{
			name: "success",
			request: &types.MsgRemoveDkimPubKey{
				Authority: f.govModAddr,
				Domain:    domain,
				Selector:  selector,
			},
			err: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			_, err := f.msgServer.RemoveDkimPubKey(f.ctx, tc.request)

			if tc.err {
				require.Error(err)
				if tc.name == "success: remove non existing key" {
					require.True(ormerrors.IsNotFound(err))
				}
			} else {
				require.NoError(err)

				r, err := f.queryServer.DkimPubKey(f.ctx, &types.QueryDkimPubKeyRequest{
					Domain:   tc.request.Domain,
					Selector: tc.request.Selector,
				})
				require.Nil(r)
				require.Error(err)
			}
		})
	}
}
