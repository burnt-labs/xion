package keeper_test

import (
	"math/big"
	"testing"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/dkim/types"

	"github.com/stretchr/testify/require"
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

func TestAuthenticate(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	publicInputs := []string{
		"2018721414038404820327",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"6632353713085157925504008443078919716322386156160602218536961028046468237192",
		"7124795577407215906429701664882261509693262157925400448966588838950094204364",
		"1729865810",
		"43113996133614694763028116931624199507",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"15410845306913030315557266389633342942173932394505300961928199486942755829435",
		"0",
	}

	domainParts, err := types.ConvertStringArrayToBigInt(publicInputs[0:9])
	require.NoError(err)
	dkimDomain, err := types.ConvertBigIntArrayToString(domainParts)
	require.NoError(err)
	poseidonHash, ok := new(big.Int).SetString(publicInputs[9], 10)
	require.True(ok)
	_, err = f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
		Authority: f.govModAddr,
		DkimPubkeys: []types.DkimPubKey{
			{
				Domain:       "gmail.com",
				PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
				PoseidonHash: poseidonHash.Bytes(),
				Selector:     "selector1",
				Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			},
		},
	})
	require.NoError(err)

	txParts, err := types.ConvertStringArrayToBigInt(publicInputs[12:32])
	require.NoError(err)
	txBytes, err := types.ConvertBigIntArrayToString(txParts)
	require.NoError(err)

	emailHashStr := "15410845306913030315557266389633342942173932394505300961928199486942755829435"
	emailHash, ok := new(big.Int).SetString(emailHashStr, 10)
	require.True(ok)

	emailHashBz := emailHash.FillBytes(make([]byte, 32))
	for i, j := 0, len(emailHashBz)-1; i < j; i, j = i+1, j-1 {
		emailHashBz[i], emailHashBz[j] = emailHashBz[j], emailHashBz[i]
	}

	proofJSON := []byte(`{"pi_a":["9304174403335741259315518336518591609512748783415613804180487556630236265024","6713554026312915505201081736695954389800531920719678211757357553378458757296","1"],"pi_b":[["16283592920163593399978697946723173702052308100676484085110069910633272630880","9039537598426725764887907662186734325645584110941201061286098607080677326933"],["14554757751308593637715610618566027327163703939068223562934833009988299531481","3001065032657973076124882599580187748929460243547839080637309134158102691344"],["1","0"]],"pi_c":["4535679481576908067801207899542111592376066219403823905750205749141422222138","18664092154600451155500251476777053301690018888778183657315682284589674588917","1"],"protocol":"groth16","curve":"bn128"}`)

	res, err := f.queryServer.Authenticate(f.ctx, &types.QueryAuthenticateRequest{
		DkimDomain:   dkimDomain,
		TxBytes:      []byte(txBytes),
		EmailHash:    emailHashBz,
		Proof:        proofJSON,
		PublicInputs: publicInputs,
	})
	require.Nil(err)
	require.NotNil(res)
	require.True(res.Verified)
}

// this function converts a byte slice to little-endian format and trims leading zeros
func ToLittleEndianWithTrimming(b []byte) []byte {
	result := make([]byte, 0)
	skipZeros := true

	for i := range b {
		val := b[i]
		if skipZeros && val == 0 {
			continue
		}
		skipZeros = false
		result = append(result, val)
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}
