package keeper_test

import (
	"encoding/base64"
	"testing"

	"cosmossdk.io/orm/types/ormerrors"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestQueryDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)
	count := 10
	domain := "xion.burnt.com"
	pubKey := base64.RawStdEncoding.EncodeToString([]byte("test-pub-key"))
	createReq := CreateNDkimPubKey(domain, pubKey, types.Version_DKIM1, types.KeyType_RSA, count)

	testCases := []struct {
		name    string
		request *types.QueryDkimPubKeyRequest
		err     bool
		errType error
		result  *types.QueryDkimPubKeyResponse
	}{
		{
			name: "fail; no such selector",
			request: &types.QueryDkimPubKeyRequest{
				Selector: "no-such-selector",
				Domain:   domain,
			},
			err:     true,
			errType: ormerrors.NotFound,
		},
		{
			name: "success",
			request: &types.QueryDkimPubKeyRequest{
				Domain:   "xion.burnt.com",
				Selector: createReq[0].Selector,
			},
			err: false,
			result: &types.QueryDkimPubKeyResponse{
				DkimPubkey: &types.DkimPubKey{
					Domain:   domain,
					PubKey:   pubKey,
					Selector: createReq[0].Selector,
					Version:  types.Version_DKIM1,
					KeyType:  types.KeyType_RSA,
				},
				PoseidonHash: []byte(pubKey),
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.msgServer.AddDkimPubKey(f.ctx, &types.MsgAddDkimPubKey{
				Authority:   f.govModAddr,
				DkimPubkeys: createReq,
			})
			require.NoError(err)
			_, err = f.queryServer.DkimPubKey(f.ctx, tc.request)
			if tc.err {
				require.Error(err)
				require.Equal(err.Error(), tc.errType.Error())
			} else {
				if tc.result != nil {
					require.NoError(err)
					require.EqualValues(tc.result, tc.result)

				}
			}

		})
	}
}
