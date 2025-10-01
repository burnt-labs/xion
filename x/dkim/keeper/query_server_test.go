package keeper_test

import (
	b64 "encoding/base64"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestQueryDkimPubKey(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)
	count := 10
	domain := "xion.burnt.com"
	pubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	createReq := CreateNDkimPubKey(t, domain, pubKey, types.Version_VERSION_DKIM1_UNSPECIFIED, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, count)
	hash, err := types.ComputePoseidonHash(pubKey)
	require.NoError(err)

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
			errType: collections.ErrNotFound,
		},
		{
			name: "success",
			request: &types.QueryDkimPubKeyRequest{
				Domain:   "xion.burnt.com",
				Selector: createReq[0].Selector,
			},
			err: false,
			result: &types.QueryDkimPubKeyResponse{
				DkimPubKey: &types.DkimPubKey{
					Domain:       domain,
					PubKey:       pubKey,
					Selector:     createReq[0].Selector,
					Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
					KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
					PoseidonHash: hash.Bytes(),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
				Authority:   f.govModAddr,
				DkimPubkeys: createReq,
			})
			require.NoError(err)
			res, err := f.queryServer.DkimPubKey(f.ctx, tc.request)
			if tc.err {
				require.Error(err)
				require.ErrorIs(err, tc.errType)
			} else if tc.result != nil {
				require.NoError(err)
				require.EqualValues(tc.result, res) // NOTE: we seem to be getting different msgs
			}
		})
	}
}

func TestQueryDkimPubKeysPagination(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)
	domain := "test.com"
	pubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	count := 5
	createReq := CreateNDkimPubKey(t, domain, pubKey, types.Version_VERSION_DKIM1_UNSPECIFIED, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, count)

	_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
		Authority:   f.govModAddr,
		DkimPubkeys: createReq,
	})
	require.NoError(err)

	t.Run("pagination with limit", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Pagination: &query.PageRequest{Limit: 2},
		})
		require.NoError(err)
		require.Len(res.DkimPubKeys, 2)
		require.NotNil(res.Pagination)
	})

	t.Run("pagination with offset", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Pagination: &query.PageRequest{Offset: 2, Limit: 2},
		})
		require.NoError(err)
		require.Len(res.DkimPubKeys, 2)
	})

	t.Run("filter by domain", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain: domain,
		})
		require.NoError(err)
		require.Len(res.DkimPubKeys, count)
	})

	t.Run("filter by non-existent domain", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain: "nonexistent.com",
		})
		require.NoError(err)
		require.Empty(res.DkimPubKeys)
	})

	t.Run("query by domain and selector", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:   domain,
			Selector: createReq[0].Selector,
		})
		require.NoError(err)
		require.Len(res.DkimPubKeys, 1)
		require.Equal(createReq[0].Selector, res.DkimPubKeys[0].Selector)
		require.Equal(domain, res.DkimPubKeys[0].Domain)
	})

	t.Run("query by domain and non-existent selector", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:   domain,
			Selector: "nonexistent-selector",
		})
		require.Error(err)
		require.Nil(res)
	})

	t.Run("filter by poseidon hash", func(t *testing.T) {
		// First get a key to extract its hash
		firstKey, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:   domain,
			Selector: createReq[0].Selector,
		})
		require.NoError(err)
		require.Len(firstKey.DkimPubKeys, 1)

		// Now query by that hash
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			PoseidonHash: firstKey.DkimPubKeys[0].PoseidonHash,
		})
		require.NoError(err)
		require.NotEmpty(res.DkimPubKeys)
		// Verify all returned keys have the same hash
		for _, key := range res.DkimPubKeys {
			require.Equal(firstKey.DkimPubKeys[0].PoseidonHash, key.PoseidonHash)
		}
	})

	t.Run("filter by non-existent poseidon hash", func(t *testing.T) {
		nonExistentHash := make([]byte, 32)
		for i := range nonExistentHash {
			nonExistentHash[i] = 0xFF
		}
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			PoseidonHash: nonExistentHash,
		})
		require.NoError(err)
		require.Empty(res.DkimPubKeys)
	})
}

func TestQueryProofVerify(t *testing.T) {
	f := SetupTest(t)

	email, err := b64.StdEncoding.DecodeString("sAcYdn1nulpzJIM0RMaX6Vn5GPPGXuHxM//AfW7b7yU=")
	require.NoError(t, err)
	var emailBz [32]byte
	copy(emailBz[:], email)
	_, err = fr.LittleEndian.Element(&emailBz)
	require.NoError(t, err)

	proof64 := "eyJwaV9hIjpbIjEyNTA3NDQ3MTEzNzIzNDEyMDAzMjI0MTg2NDI3NTAyMDIwNzk1MjMzMDY1NjAxNjk0NDc1OTQ4MzUzOTE2MzY3NDU4MDIzOTE3Mzc4IiwiMTE2NDc0OTIzMTc0MTEyMDM1MDEwMDUwMDA0MDA0NDUzNTc5NjA1MjYxNjQ2Nzg4NjAwNzMyNDA5MzQ1Mzk0MTIxNzI1Mzk2NjIzOTMiLCIxIl0sInBpX2IiOltbIjc5NDA1NDkzMTQzODQxMDU2OTYwMjkwNTg5OTY5NzA0NzcyMjIyNTA3MjIzMTg0NzMyMjIzMDYzNjU3NTc3MDk5NTAzMTYzMjg2MjIiLCIzODcxODAzODIyMzE1NzM3ODE0NTA3OTkwNjc4MzA1NzM4OTQ2OTYyNDM0MjkzNDg4MDEwMzE0NjE0NjczMDQ5ODcyMzYxMzU3ODI0Il0sWyI5MzIxNjI4MTQ3MjY4ODM3MzMyODI2Njg1NjkxNDk2NjE5OTc2MjE0MzQwNTI5NjI2ODkwNjA1OTAxNTMwMjA1NjQ0ODkxMTU3NjIzIiwiMTEwNTU2MzM2MjM4NjM1MjI1NTA4MzkyOTIyMTUxOTkyNTAxMDI2ODE2NzY5MjMyOTQ1NDI1NzE4NDU4NTY2MzY5MTI0NzgyMTUyNDkiXSxbIjEiLCIwIl1dLCJwaV9jIjpbIjE0NjU3MDc0NDIxNzY4NTE1ODM1ODU4OTg1OTE3NTc4NjIxMjM5NTY1MzI4OTY1NDAyODU0MTc5MzU2MjE4NDQ4NjU0ODYwNTcyODg5IiwiMTMyNzk0MDQ2MzkyNjg2MTQ3MjY0NTE2MDY5NTIyMzE0NjU4ODI2NDEzNjUxNDk2NzI4NDE5MjQ1NDE3NjgzOTkwMjU2NjEzMjEzMTEiLCIxIl19"
	proofData, err := b64.StdEncoding.DecodeString(proof64)
	require.NoError(t, err)

	txB4s := "CqIBCp8BChwvY29zbW9zLmJhbmsudjFiZXRhMS5Nc2dTZW5kEn8KP3hpb24xNG43OWVocGZ3aGRoNHN6dWRhZ2Q0bm14N2NsajU3bHk1dTBsenhzNm1nZjVxeTU1a3k5c21zenM0OBIreGlvbjFxYWYyeGZseDVqM2FndGx2cWs1dmhqcGV1aGw2ZzQ1aHhzaHdxahoPCgV1eGlvbhIGMTAwMDAwEmUKTQpDCh0vYWJzdHJhY3RhY2NvdW50LnYxLk5pbFB1YktleRIiCiCs/FzcKXXbesBcb1Daz2b2Pyp75Kcf8Roa2hNAEpSxCxIECgIIARgBEhQKDgoFdXhpb24SBTYwMDAwEICJehoGeGlvbi0xIAw="
	dkimBz, err := b64.StdEncoding.DecodeString("iEeNSGFNAiTctrIgoVuE40DFz/ATm+ip5RBx3HfHqQ4=")
	require.NoError(t, err)

	testCases := []struct {
		name        string
		proofBz     []byte
		txBz        string
		dkimBz      []byte
		emailBz     []byte
		shouldError bool
	}{
		{
			name:        "verify proof success",
			proofBz:     proofData,
			txBz:        txB4s,
			dkimBz:      dkimBz,
			emailBz:     email,
			shouldError: false,
		},
		{
			name:        "invalid proof data",
			proofBz:     []byte("invalid"),
			txBz:        txB4s,
			dkimBz:      dkimBz,
			emailBz:     email,
			shouldError: true,
		},
		{
			name:        "empty proof",
			proofBz:     []byte{},
			txBz:        txB4s,
			dkimBz:      dkimBz,
			emailBz:     email,
			shouldError: true,
		},
		{
			name:    "invalid email hash - all zeros",
			proofBz: proofData,
			txBz:    txB4s,
			dkimBz:  dkimBz,
			emailBz: func() []byte {
				b := make([]byte, 32)
				return b
			}(),
			shouldError: true, // Will fail proof verification
		},
		{
			name:    "invalid dkim hash - all zeros",
			proofBz: proofData,
			txBz:    txB4s,
			dkimBz: func() []byte {
				b := make([]byte, 32)
				return b
			}(),
			emailBz:     email,
			shouldError: true, // Will fail proof verification
		},
		{
			name:        "invalid transaction bytes",
			proofBz:     proofData,
			txBz:        "invalid-base64-tx!!!",
			dkimBz:      dkimBz,
			emailBz:     email,
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			r := &types.QueryVerifyRequest{
				TxBytes:   []byte(tc.txBz),
				Proof:     tc.proofBz,
				DkimHash:  tc.dkimBz,
				EmailHash: tc.emailBz,
			}
			res, err := f.queryServer.ProofVerify(f.ctx, r)
			if tc.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, res.Verified)
			}
		})
	}
}
