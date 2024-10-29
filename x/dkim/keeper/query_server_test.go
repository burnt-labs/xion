package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/orm/types/ormerrors"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestQueryDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)
	count := 10
	domain := "xion.burnt.com"
	pubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
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
		t.Run(tc.name, func(_ *testing.T) {
			_, err := f.msgServer.AddDkimPubKey(f.ctx, &types.MsgAddDkimPubKey{
				Authority:   f.govModAddr,
				DkimPubkeys: createReq,
			})
			require.NoError(err)
			_, err = f.queryServer.DkimPubKey(f.ctx, tc.request)
			if tc.err {
				require.Error(err)
				require.Equal(err.Error(), tc.errType.Error())
			} else if tc.result != nil {
				require.NoError(err)
				require.EqualValues(tc.result, tc.result)
			}
		})
	}
}

func TestQueryPoseidonHash(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)
	pubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	res := "1983664618407009423875829639306275185491946247764487749439145140682408188330"

	testCases := []struct {
		name    string
		request *types.PoseidonHashRequest
		err     bool
		errType error
		result  *types.PoseidonHashResponse
	}{
		{
			name: "fail; invalid public key",
			request: &types.PoseidonHashRequest{
				PublicKey: "invalid-pub-key",
			},
			err: true,
		},
		{
			name: "success",
			request: &types.PoseidonHashRequest{
				PublicKey: pubKey,
			},
			err: false,
			result: &types.PoseidonHashResponse{
				PoseidonHash: []byte(res),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			result, err := f.queryServer.PoseidonHash(f.ctx, tc.request)
			if tc.err {
				require.Error(err)
			} else if tc.result != nil {
				require.NoError(err)
				require.EqualValues(result, tc.result)
			}
		})
	}
}
