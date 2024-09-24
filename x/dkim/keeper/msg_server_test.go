package keeper_test

import (
	"testing"

	"cosmossdk.io/orm/types/ormerrors"
	"github.com/stretchr/testify/require"

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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.msgServer.UpdateParams(f.ctx, tc.request)

			if tc.err {
				require.Error(err)
			} else {
				require.NoError(err)

				r, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
				require.NoError(err)

				require.EqualValues(&tc.request.Params, r.Params)
			}

		})
	}
}

func TestAddDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	testCases := []struct {
		name    string
		request *types.MsgAddDkimPubKey
		err     bool
	}{
		{
			name: "fail; invalid authority",
			request: &types.MsgAddDkimPubKey{
				Authority: f.addrs[0].String(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   "xion",
						Selector: "zkemail",
					},
				},
			},
			err: true,
		},
		{
			name: "success",
			request: &types.MsgAddDkimPubKey{
				Authority: f.govModAddr,
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   "xion",
						Selector: "zkemail",
					},
				},
			},
			err: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.msgServer.AddDkimPubKey(f.ctx, tc.request)

			if tc.err {
				require.Error(err)
			} else {
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

	domain := "xion.burnt.com"
	pubKey := "xion"
	selector := "zkemail"

	_, err := f.msgServer.AddDkimPubKey(f.ctx, &types.MsgAddDkimPubKey{
		Authority: f.govModAddr,
		DkimPubkeys: []types.DkimPubKey{
			{
				Domain:   domain,
				PubKey:   pubKey,
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
			name: "success",
			request: &types.MsgRemoveDkimPubKey{
				Authority: f.govModAddr,
				Domain:    domain,
				Selector:  selector,
			},
			err: false,
		},
		{
			name: "success: remove non existing key",
			request: &types.MsgRemoveDkimPubKey{
				Authority: f.govModAddr,
				Domain:    domain,
				Selector:  "non-existing",
			},
			err: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
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
