package keeper_test

import (
	"encoding/base64"
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
						PubKey:   base64.RawStdEncoding.EncodeToString([]byte("test-pub-key")),
						Selector: "zkemail",
					},
				},
			},
			err: true,
		},
		{
			name: "fail; invalid keytype",
			request: &types.MsgAddDkimPubKey{
				Authority: f.addrs[0].String(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   base64.RawStdEncoding.EncodeToString([]byte("test-pub-key")),
						Selector: "zkemail",
						KeyType:  2,
					},
				},
			},
			err: true,
		},
		{
			name: "fail; invalid version",
			request: &types.MsgAddDkimPubKey{
				Authority: f.addrs[0].String(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   base64.RawStdEncoding.EncodeToString([]byte("test-pub-key")),
						Selector: "zkemail",
						Version:  2,
					},
				},
			},
			err: true,
		},
		{
			name: "fail; invalid pubkey",
			request: &types.MsgAddDkimPubKey{
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
			request: &types.MsgAddDkimPubKey{
				Authority: f.govModAddr,
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "xion.burnt.com",
						PubKey:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAntvSKT1hkqhKe0xcaZ0x+QbouDsJuBfby/S82jxsoC/SodmfmVs2D1KAH3mi1AqdMdU12h2VfETeOJkgGYq5ljd996AJ7ud2SyOLQmlhaNHH7Lx+Mdab8/zDN1SdxPARDgcM7AsRECHwQ15R20FaKUABGu4NTbR2fDKnYwiq5jQyBkLWP+LgGOgfUF4T4HZb2PY2bQtEP6QeqOtcW4rrsH24L7XhD+HSZb1hsitrE0VPbhJzxDwI4JF815XMnSVjZgYUXP8CxI1Y0FONlqtQYgsorZ9apoW1KPQe8brSSlRsi9sXB/tu56LmG7tEDNmrZ5XUwQYUUADBOu7t1niwXwIDAQAB",
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
	pubKey := base64.RawStdEncoding.EncodeToString([]byte("test-pub-key"))
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
