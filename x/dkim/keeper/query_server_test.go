package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestQueryDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	testCases := []struct {
		name    string
		request *types.QueryDkimPubKeyRequest
		err     bool
	}{
		{
			name: "fail; invalid authority",
			request: &types.QueryDkimPubKeyRequest{
				Selector: "zkemail",
				Domain:   "xion.burnt.com",
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
