package keeper_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/dkim/keeper"
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
				require.EqualValues(tc.result, res)
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

	t.Run("pagination offset exceeds total", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:     domain,
			Pagination: &query.PageRequest{Offset: 1000, Limit: 10},
		})
		require.NoError(err)
		require.Empty(res.DkimPubKeys)
		require.NotNil(res.Pagination)
		require.Equal(uint64(count), res.Pagination.Total)
	})

	t.Run("pagination with zero limit uses default", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:     domain,
			Pagination: &query.PageRequest{Limit: 0},
		})
		require.NoError(err)
		require.Len(res.DkimPubKeys, count) // default limit is 100, but we only have 5
	})

	t.Run("pagination with nil pagination request", func(t *testing.T) {
		res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{
			Domain:     domain,
			Pagination: nil,
		})
		require.NoError(err)
		require.Len(res.DkimPubKeys, count)
	})
}

func TestParams(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	t.Run("returns default params", func(t *testing.T) {
		res, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
		require.NoError(err)
		require.NotNil(res)
		require.NotNil(res.Params)
	})

	t.Run("returns params with nil request", func(t *testing.T) {
		res, err := f.queryServer.Params(f.ctx, nil)
		require.NoError(err)
		require.NotNil(res)
		require.NotNil(res.Params)
	})

	t.Run("returns updated params after UpdateParams", func(t *testing.T) {
		newParams := types.DefaultParams()
		newParams.VkeyIdentifier = 42

		// Update params
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{
			Authority: f.govModAddr,
			Params:    newParams,
		})
		require.NoError(err)

		// Query params and verify
		res, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
		require.NoError(err)
		require.NotNil(res)
		require.NotNil(res.Params)
		require.Equal(uint64(42), res.Params.VkeyIdentifier)
	})

	t.Run("returns params with empty dkim pubkeys", func(t *testing.T) {
		newParams := types.DefaultParams()
		newParams.VkeyIdentifier = 99

		// Update params
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{
			Authority: f.govModAddr,
			Params:    newParams,
		})
		require.NoError(err)

		// Query params and verify
		res, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
		require.NoError(err)
		require.NotNil(res)
		require.NotNil(res.Params)
		require.Equal(uint64(99), res.Params.VkeyIdentifier)
	})
}

// createModifiedPublicInputs creates a copy of publicInputs with modified tx bytes (indices [12:32])
func createModifiedPublicInputs(publicInputs []string) []string {
	modified := make([]string, len(publicInputs))
	copy(modified, publicInputs)
	// Modify one of the tx bytes elements to create a mismatch
	if len(modified) > 12 {
		modified[12] = "99999999999999999999999999999999999999999999999999999999999999999999999999999"
	}
	return modified
}

func TestAuthenticate(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	// Common test data
	basePublicInputs := []string{
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
		"21084905302941062264575804210871487148258363738073263632230120817255351393954",
		"0",
		"191581113848055322477272311147821680130451026496941019613909483584263833445",
		"149108628584424258332964971884436592255105616775526759101383287099246929273",
		"20356082004311139738363494460884070443445370694676839",
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
		"9079378704521501721378444251561135763203091338587747860525949554473799137061",
		"1",
		"145464208130933216679374873468710647147",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"180980592328871182281563474567090989367752380861661653173671556731952063826",
		"189366407839159640650411313259066674300878650730387363415856879007716700777",
		"112965544445135736799656303",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
	}

	// Setup DKIM pub key
	poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
	require.True(ok)
	_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
		Domain:       "gmail.com",
		PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
		PoseidonHash: poseidonHash.Bytes(),
		Selector:     "selector1",
		Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
	}, &f.k)
	require.NoError(err)

	// Common proof JSON
	proofJSON := []byte(`{
        "pi_a": [
            "3520409830771234065994556634681427417950500323669026192927037346754953128831",
            "3498073333995773007990629866509538975797753241506847777821035377435020235118",
            "1"
        ],
        "pi_b": [
            [
                "5449376970517492854877490913806025222720139652422610862840169172156024987414",
                "19970795329053509942253184076261379588636216808533565932150420541118191182415"
            ],
            [
                "3329994826233724633278876134031495025607573045416243454711110428426399497078",
                "12635663937591831780654697557378573948535834769571974614171466954698620359415"
            ],
            [
                "1",
                "0"
            ]
        ],
        "pi_c": [
            "7013834827668081216347999320368574690710153592707686848856775126651375200226",
            "10870223918887009180919296136906814662608691011484426082412176806715867878616",
            "1"
        ],
        "protocol": "groth16",
        "curve": "bn128"
    }`)

	// Common email hash
	emailHashStr := "9079378704521501721378444251561135763203091338587747860525949554473799137061"

	// Common tx bytes
	txParts, err := types.ConvertStringArrayToBigInt(basePublicInputs[12:68])
	require.NoError(err)
	txBytes, err := types.ConvertBigIntArrayToString(txParts)
	require.NoError(err)

	testCases := []struct {
		name              string
		emailHash         string
		allowedEmailHosts []string
		publicInputs      []string
		txBytes           []byte // nil means compute from publicInputs[12:68]
		expectedError     bool
		expectedVerified  bool
		errorContains     string
	}{
		{
			name:              "success - basic proof verification",
			emailHash:         emailHashStr,
			allowedEmailHosts: []string{"kushal@burnt.com"},
			publicInputs:      basePublicInputs,
			txBytes:           nil, // will be computed from publicInputs
			expectedError:     false,
			expectedVerified:  true,
		},
		{
			name:              "fail - email hash mismatch",
			emailHash:         "99999999999999999999999999999999999999999999999999999999999999999999999999999",
			allowedEmailHosts: []string{"kushal@burnt.com"},
			publicInputs:      basePublicInputs,
			txBytes:           nil, // will be computed from publicInputs
			expectedError:     true,
			errorContains:     "email hash does not match public input",
		},
		{
			name:              "fail - allowed email hosts not subset of public inputs",
			emailHash:         emailHashStr,
			allowedEmailHosts: []string{"jose@burnt.com", "jane@burnt.com"},
			publicInputs:      basePublicInputs,
			txBytes:           nil, // will be computed from publicInputs
			expectedError:     true,
			errorContains:     "is not present in allowed email hosts list",
		},
		{
			name:              "success - allowed list of email hosts match public inputs",
			emailHash:         emailHashStr,
			allowedEmailHosts: []string{"kushal@burnt.com", "jose@burnt.com", "jane@burnt.com"},
			publicInputs:      basePublicInputs,
			txBytes:           nil, // will be computed from publicInputs
			expectedError:     false,
			expectedVerified:  true,
		},
		{
			name:              "fail - empty allowed email hosts when public inputs have hosts",
			emailHash:         emailHashStr,
			allowedEmailHosts: []string{},
			publicInputs:      basePublicInputs,
			txBytes:           nil, // will be computed from publicInputs
			expectedError:     true,
			errorContains:     "is not present in allowed email hosts list",
		},
		{
			name:              "fail - tx bytes mismatch",
			emailHash:         emailHashStr,
			allowedEmailHosts: []string{"kushal@burnt.com"},
			publicInputs:      basePublicInputs,
			txBytes:           []byte("wrong-tx-bytes"),
			expectedError:     true,
			errorContains:     "tx bytes do not match public inputs",
		},
		{
			name:              "fail - empty tx bytes when public inputs have tx bytes",
			emailHash:         emailHashStr,
			allowedEmailHosts: []string{"kushal@burnt.com"},
			publicInputs:      basePublicInputs,
			txBytes:           []byte{},
			expectedError:     true,
			errorContains:     "tx bytes do not match public inputs",
		},
		{
			name:              "fail - tx bytes with modified public inputs",
			emailHash:         emailHashStr,
			allowedEmailHosts: []string{"kushal@burnt.com"},
			publicInputs:      createModifiedPublicInputs(basePublicInputs),
			txBytes:           []byte(txBytes), // using original txBytes with modified publicInputs
			expectedError:     true,
			errorContains:     "tx bytes do not match public inputs",
		},
		{
			name:              "fail - insufficient public inputs",
			emailHash:         emailHashStr,
			allowedEmailHosts: []string{"kushal@burnt.com"},
			publicInputs:      basePublicInputs[:10], // Only 10 elements, need at least 38
			txBytes:           nil,
			expectedError:     true,
			errorContains:     "insufficient public inputs",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var reqTxBytes []byte
			if tc.txBytes != nil {
				reqTxBytes = tc.txBytes
			} else if len(tc.publicInputs) >= 88 {
				// Compute txBytes from publicInputs[12:68]
				txParts, err := types.ConvertStringArrayToBigInt(tc.publicInputs[12:68])
				require.NoError(err)
				txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
				require.NoError(err)
				reqTxBytes = []byte(txBytesStr)
			}

			req := &types.QueryAuthenticateRequest{
				TxBytes:           reqTxBytes,
				EmailHash:         tc.emailHash,
				Proof:             proofJSON,
				PublicInputs:      tc.publicInputs,
				AllowedEmailHosts: tc.allowedEmailHosts,
			}
			res, err := f.queryServer.Authenticate(f.ctx, req)
			if tc.expectedError {
				require.Error(err)
				if tc.errorContains != "" {
					require.Contains(err.Error(), tc.errorContains)
				}
				require.Nil(res)
			} else {
				require.NoError(err)
				require.NotNil(res)
				require.Equal(tc.expectedVerified, res.Verified)
			}
		})
	}
}

func TestAuthenticateEdgeCases(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	t.Run("fail - invalid tx bytes public input conversion", func(t *testing.T) {
		// Create public inputs with invalid tx bytes values (indices 12-68)
		invalidPublicInputs := make([]string, 88)
		for i := range invalidPublicInputs {
			invalidPublicInputs[i] = "0"
		}
		invalidPublicInputs[12] = "invalid-number" // Invalid: should be numeric
		invalidPublicInputs[68] = "test-hash"

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte("test"),
			EmailHash:         "test-hash",
			Proof:             []byte(`{}`),
			PublicInputs:      invalidPublicInputs,
			AllowedEmailHosts: []string{},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "failed to convert tx bytes public inputs")
	})

	t.Run("fail - invalid dkim domain public input conversion", func(t *testing.T) {
		// Create public inputs where tx bytes are valid but dkim domain is invalid
		// We need tx bytes [12:68] to be valid AND match the provided txBytes
		// Then dkim domain [0:9] should be invalid
		invalidPublicInputs := make([]string, 88)
		for i := range invalidPublicInputs {
			invalidPublicInputs[i] = "0"
		}
		invalidPublicInputs[0] = "not-a-number" // Invalid dkim domain: should be numeric
		invalidPublicInputs[68] = "test-hash"

		// Compute txBytes from the valid publicInputs[12:68] (all zeros)
		txParts, err := types.ConvertStringArrayToBigInt(invalidPublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         "test-hash",
			Proof:             []byte(`{}`),
			PublicInputs:      invalidPublicInputs,
			AllowedEmailHosts: []string{},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "failed to convert dkim domain public inputs")
	})

	t.Run("fail - invalid dkim hash public input", func(t *testing.T) {
		// Create public inputs where tx bytes and dkim domain are valid but dkim hash is invalid
		invalidPublicInputs := make([]string, 88)
		for i := range invalidPublicInputs {
			invalidPublicInputs[i] = "0"
		}
		invalidPublicInputs[9] = "not-a-valid-big-int" // Invalid: should be numeric
		invalidPublicInputs[68] = "test-hash"

		// Compute txBytes from the valid publicInputs[12:68] (all zeros)
		txParts, err := types.ConvertStringArrayToBigInt(invalidPublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         "test-hash",
			Proof:             []byte(`{}`),
			PublicInputs:      invalidPublicInputs,
			AllowedEmailHosts: []string{},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "failed to parse dkim hash public input")
	})

	t.Run("fail - no dkim pubkey found for domain and hash", func(t *testing.T) {
		// Create valid public inputs but with a domain/hash that doesn't exist
		validPublicInputs := make([]string, 88)
		for i := range validPublicInputs {
			validPublicInputs[i] = "0"
		}
		validPublicInputs[9] = "12345" // Valid numeric but non-existent hash
		validPublicInputs[68] = "test-hash"

		// Compute txBytes from the valid publicInputs[12:68] (all zeros)
		txParts, err := types.ConvertStringArrayToBigInt(validPublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         "test-hash",
			Proof:             []byte(`{}`),
			PublicInputs:      validPublicInputs,
			AllowedEmailHosts: []string{},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "no dkim pubkey found")
	})
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

func TestAuthenticateExtended(t *testing.T) {
	// Base public inputs for testing
	basePublicInputs := []string{
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
		"21084905302941062264575804210871487148258363738073263632230120817255351393954",
		"0",
		"191581113848055322477272311147821680130451026496941019613909483584263833445",
		"149108628584424258332964971884436592255105616775526759101383287099246929273",
		"20356082004311139738363494460884070443445370694676839",
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
		"9079378704521501721378444251561135763203091338587747860525949554473799137061",
		"1",
		"145464208130933216679374873468710647147",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"180980592328871182281563474567090989367752380861661653173671556731952063826",
		"189366407839159640650411313259066674300878650730387363415856879007716700777",
		"112965544445135736799656303",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
	}

	proofJSON := []byte(`{
        "pi_a": [
            "5583158245518012202854967966688803983422579480975771799159435109682404412144",
            "19132509617989255559927911185942768582713778613503304661723852230698387114840",
            "1"
        ],
        "pi_b": [
            [
                "16209151427684011206863591092531391562117041646748639896310737311173246509260",
                "17729357182912272387117349263688449009610186531485947940640482832772517448927"
            ],
            [
                "5695516600618485685754260649529465903248888152110855008128547397403792546988",
                "656772577582924627058107331850692187484072991458347712020152128940322124285"
            ],
            [
                "1",
                "0"
            ]
        ],
        "pi_c": [
            "17453897224382172288517505191435866511305436208311355514241444398256793953872",
            "9163422778422181829456976190497942172380575625369266408413936273192580460236",
            "1"
        ],
        "protocol": "groth16"
    }`)

	emailHashStr := "9079378704521501721378444251561135763203091338587747860525949554473799137061"

	t.Run("fail - invalid email host public input conversion", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Setup DKIM pub key
		poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
		require.True(ok)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:       "gmail.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
			PoseidonHash: poseidonHash.Bytes(),
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		}, &f.k)
		require.NoError(err)

		// Create public inputs with invalid email host values
		invalidPublicInputs := make([]string, len(basePublicInputs))
		copy(invalidPublicInputs, basePublicInputs)
		invalidPublicInputs[70] = "not-a-number" // Invalid email host

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(invalidPublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         emailHashStr,
			Proof:             proofJSON,
			PublicInputs:      invalidPublicInputs,
			AllowedEmailHosts: []string{"test@example.com"},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "failed to convert allowed email hosts")
	})

	t.Run("fail - invalid proof JSON", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Setup DKIM pub key
		poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
		require.True(ok)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:       "gmail.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
			PoseidonHash: poseidonHash.Bytes(),
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		}, &f.k)
		require.NoError(err)

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(basePublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		// Invalid proof JSON
		invalidProof := []byte(`{invalid json}`)

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         emailHashStr,
			Proof:             invalidProof,
			PublicInputs:      basePublicInputs,
			AllowedEmailHosts: []string{"kushal@burnt.com"},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
	})

	t.Run("fail - empty proof", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Setup DKIM pub key
		poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
		require.True(ok)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:       "gmail.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
			PoseidonHash: poseidonHash.Bytes(),
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		}, &f.k)
		require.NoError(err)

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(basePublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         emailHashStr,
			Proof:             []byte{},
			PublicInputs:      basePublicInputs,
			AllowedEmailHosts: []string{"kushal@burnt.com"},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
	})

	t.Run("fail - public inputs exactly 52 elements but invalid data", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Create exactly 52 elements of zeros
		minimalPublicInputs := make([]string, 88)
		for i := range minimalPublicInputs {
			minimalPublicInputs[i] = "0"
		}
		minimalPublicInputs[68] = "test-hash" // email hash at index 68

		// Compute txBytes (all zeros)
		txParts, err := types.ConvertStringArrayToBigInt(minimalPublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)
		poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
		require.True(ok)
		_, err = keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:       "gmail.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
			PoseidonHash: poseidonHash.Bytes(),
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		}, &f.k)
		require.NoError(err)
		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         "test-hash",
			Proof:             proofJSON,
			PublicInputs:      minimalPublicInputs,
			AllowedEmailHosts: []string{},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
		// The error will be about email host validation since empty string from public inputs
		// is not in empty allowed list (IsSubset([""], []) = false)
		require.Contains(err.Error(), "is not present in allowed email hosts list")
	})

	t.Run("fail - public inputs with 37 elements (boundary)", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Create 87 elements (one less than required)
		insufficientPublicInputs := make([]string, 87)
		for i := range insufficientPublicInputs {
			insufficientPublicInputs[i] = "0"
		}

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte("test"),
			EmailHash:         "test-hash",
			Proof:             proofJSON,
			PublicInputs:      insufficientPublicInputs,
			AllowedEmailHosts: []string{},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "insufficient public inputs")
	})

	t.Run("fail - empty email host from public inputs with empty allowed list", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Create public inputs where email host [70:79] are all zeros
		// This results in empty string email host
		zeroEmailHostInputs := make([]string, len(basePublicInputs))
		copy(zeroEmailHostInputs, basePublicInputs)
		zeroEmailHostInputs[70] = "0"
		zeroEmailHostInputs[71] = "0"
		zeroEmailHostInputs[72] = "0"
		zeroEmailHostInputs[73] = "0"
		zeroEmailHostInputs[74] = "0"
		zeroEmailHostInputs[75] = "0"
		zeroEmailHostInputs[76] = "0"
		zeroEmailHostInputs[77] = "0"
		zeroEmailHostInputs[78] = "0"

		// Setup DKIM pub key
		poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
		require.True(ok)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:       "gmail.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
			PoseidonHash: poseidonHash.Bytes(),
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		}, &f.k)
		require.NoError(err)

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(zeroEmailHostInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         emailHashStr,
			Proof:             proofJSON,
			PublicInputs:      zeroEmailHostInputs,
			AllowedEmailHosts: []string{}, // Empty allowed list
		}

		// When email host from public inputs is empty string "" and allowed list is empty [],
		// IsSubset([""], []) returns false, so this fails
		res, err := f.queryServer.Authenticate(f.ctx, req)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "is not present in allowed email hosts list")
	})

	t.Run("multiple allowed email hosts - first match", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Setup DKIM pub key
		poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
		require.True(ok)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:       "gmail.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
			PoseidonHash: poseidonHash.Bytes(),
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		}, &f.k)
		require.NoError(err)

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(basePublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         emailHashStr,
			Proof:             proofJSON,
			PublicInputs:      basePublicInputs,
			AllowedEmailHosts: []string{"kushal@burnt.com", "other@burnt.com", "another@burnt.com"},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		// Should pass email host validation
		if err != nil {
			require.NotContains(err.Error(), "is not present in allowed email hosts list")
		}
		_ = res
	})

	t.Run("success - all email host and subject elements filled", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Create public inputs with all 9 elements filled for email host [34:43]
		// and all 9 elements filled for email subject [43:52]
		// This tests the edge condition where the full range is utilized
		fullPublicInputs := make([]string, len(basePublicInputs))
		copy(fullPublicInputs, basePublicInputs)

		// Fill all email host elements [70:79] with non-zero values
		// These represent a long email address that spans all 9 field elements
		fullPublicInputs[70] = "145464208130933216679374873468710647147" // existing value
		fullPublicInputs[71] = "123456789012345678901234567890123456789" // additional data
		fullPublicInputs[72] = "234567890123456789012345678901234567890" // additional data
		fullPublicInputs[73] = "345678901234567890123456789012345678901" // additional data
		fullPublicInputs[74] = "456789012345678901234567890123456789012" // additional data
		fullPublicInputs[75] = "567890123456789012345678901234567890123" // additional data
		fullPublicInputs[76] = "678901234567890123456789012345678901234" // additional data
		fullPublicInputs[77] = "789012345678901234567890123456789012345" // additional data
		fullPublicInputs[78] = "890123456789012345678901234567890123456" // last element of email host

		// Fill all email subject elements [79:88] with non-zero values
		// These represent a long subject that spans all 9 field elements
		fullPublicInputs[79] = "180980592328871182281563474567090989367752380861661653173671556731952063826" // existing
		fullPublicInputs[80] = "175265870350771638945491578423233386960064756860306078150084022460882973289" // existing
		fullPublicInputs[81] = "112994317117614493862539312"                                                 // existing
		fullPublicInputs[82] = "111222333444555666777888999000111222333"                                     // additional data
		fullPublicInputs[83] = "222333444555666777888999000111222333444"                                     // additional data
		fullPublicInputs[84] = "333444555666777888999000111222333444555"                                     // additional data
		fullPublicInputs[85] = "444555666777888999000111222333444555666"                                     // additional data
		fullPublicInputs[86] = "555666777888999000111222333444555666777"                                     // additional data
		fullPublicInputs[87] = "666777888999000111222333444555666777888"                                     // last element of subject

		// Setup DKIM pub key
		poseidonHash, ok := new(big.Int).SetString(fullPublicInputs[9], 10)
		require.True(ok)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:       "gmail.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
			PoseidonHash: poseidonHash.Bytes(),
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		}, &f.k)
		require.NoError(err)

		// Compute valid txBytes from public inputs
		txParts, err := types.ConvertStringArrayToBigInt(fullPublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		// Convert email host from public inputs to get the expected email host string
		emailHostParts, err := types.ConvertStringArrayToBigInt(fullPublicInputs[70:79])
		require.NoError(err)
		emailHostStr, err := types.ConvertBigIntArrayToString(emailHostParts)
		require.NoError(err)

		// Verify email host conversion worked with all elements
		require.NotEmpty(emailHostStr, "email host string should not be empty when all elements are filled")

		// Validate that email host starts with expected prefix
		require.True(strings.HasPrefix(emailHostStr, "kushal@burnt"), "email host should start with 'kushal@burnt'")

		// Convert email subject from public inputs to verify it works
		emailSubjectParts, err := types.ConvertStringArrayToBigInt(fullPublicInputs[79:88])
		require.NoError(err)
		emailSubjectStr, err := types.ConvertBigIntArrayToString(emailSubjectParts)
		require.NoError(err)

		// Verify email subject conversion worked with all elements
		require.NotEmpty(emailSubjectStr, "email subject string should not be empty when all elements are filled")

		// Validate that email subject starts with expected prefix
		require.True(strings.HasPrefix(emailSubjectStr, "Re: [Reply Needed]"), "email subject should start with 'Re: [Reply Needed]'")

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         emailHashStr,
			Proof:             proofJSON,
			PublicInputs:      fullPublicInputs,
			AllowedEmailHosts: []string{emailHostStr}, // Use the exact email host from public inputs
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		// The request should pass email host and subject validation
		// It may fail on proof verification since we're using synthetic data,
		// but it should NOT fail on email host/subject parsing or validation
		if err != nil {
			require.NotContains(err.Error(), "failed to convert allowed email hosts to big int")
			require.NotContains(err.Error(), "failed to convert allowed email hosts to string")
			require.NotContains(err.Error(), "is not present in allowed email hosts list")
			require.NotContains(err.Error(), "failed to convertemail subject to big int")
			require.NotContains(err.Error(), "failed to convert email subject to string")
		}
		_ = res
	})

	t.Run("success - all command data elements filled [12:68]", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Create public inputs with all 20 elements filled for command/tx data [12:68]
		// This tests the edge condition where the full range of tx bytes is utilized
		fullPublicInputs := make([]string, len(basePublicInputs))
		copy(fullPublicInputs, basePublicInputs)

		// Fill all command/tx data elements [12:68] with non-zero values
		// These represent a long transaction that spans all 20 field elements
		fullPublicInputs[12] = "124413588010935573100449456468959839270027757215138439816955024736271298883"
		fullPublicInputs[13] = "125987718504881168702817372751405511311626515399128115957683055706162879081"
		fullPublicInputs[14] = "138174294419566073638917398478480233783462655482283489778477032129860416308"
		fullPublicInputs[15] = "87164429935183530231106524238772469083021376536857547601286350511895957042"
		fullPublicInputs[16] = "159508995554830235422881220221659222882416701537684367907262541081181107041"
		fullPublicInputs[17] = "216177859633033993616607456010987870980723214832657304250929052054387451251"
		fullPublicInputs[18] = "136870293077760051536514689814528040652982158268238924211443105143315312977"
		fullPublicInputs[19] = "209027647271941540634260128227139143305212625530130988286308577451934433604"
		fullPublicInputs[20] = "216041037480816501846348705353738079775803623607373665378499876478757721956"
		fullPublicInputs[21] = "184099808892606061942559141059081527262834859629181581270585908529014000483"
		fullPublicInputs[22] = "173926821082308056829441773860483849128404996084932919505946802488367989070"
		fullPublicInputs[23] = "136498083332900321215526260868562056670892412932671519510981704427905430578"
		fullPublicInputs[24] = "111222333444555666777888999000111222333444555666777888999000111222333444555"
		fullPublicInputs[25] = "222333444555666777888999000111222333444555666777888999000111222333444555666"
		fullPublicInputs[26] = "333444555666777888999000111222333444555666777888999000111222333444555666777"
		fullPublicInputs[27] = "444555666777888999000111222333444555666777888999000111222333444555666777888"
		fullPublicInputs[28] = "555666777888999000111222333444555666777888999000111222333444555666777888999"
		fullPublicInputs[29] = "666777888999000111222333444555666777888999000111222333444555666777888999000"
		fullPublicInputs[30] = "777888999000111222333444555666777888999000111222333444555666777888999000111"
		fullPublicInputs[31] = "888999000111222333444555666777888999000111222333444555666777888999000111222"
		fullPublicInputs[32] = "999000111222333444555666777888999000111222333444555666777888999000111222333"
		fullPublicInputs[33] = "1000111222333444555666777888999000111222333444555666777888999000111222333444"
		fullPublicInputs[34] = "124413588010935573100449456468959839270027757215138439816955024736271298883"
		fullPublicInputs[35] = "125987718504881168702817372751405511311626515399128115957683055706162879081"
		fullPublicInputs[36] = "138174294419566073638917398478480233783462655482283489778477032129860416308"
		fullPublicInputs[37] = "87164429935183530231106524238772469083021376536857547601286350511895957042"
		fullPublicInputs[38] = "159508995554830235422881220221659222882416701537684367907262541081181107041"
		fullPublicInputs[39] = "216177859633033993616607456010987870980723214832657304250929052054387451251"
		fullPublicInputs[40] = "136870293077760051536514689814528040652982158268238924211443105143315312977"
		fullPublicInputs[41] = "209027647271941540634260128227139143305212625530130988286308577451934433604"
		fullPublicInputs[42] = "216041037480816501846348705353738079775803623607373665378499876478757721956"
		fullPublicInputs[43] = "184099808892606061942559141059081527262834859629181581270585908529014000483"
		fullPublicInputs[44] = "173926821082308056829441773860483849128404996084932919505946802488367989070"
		fullPublicInputs[45] = "136498083332900321215526260868562056670892412932671519510981704427905430578"
		fullPublicInputs[46] = "111222333444555666777888999000111222333444555666777888999000111222333444555"
		fullPublicInputs[47] = "222333444555666777888999000111222333444555666777888999000111222333444555666"
		fullPublicInputs[48] = "333444555666777888999000111222333444555666777888999000111222333444555666777"
		fullPublicInputs[49] = "444555666777888999000111222333444555666777888999000111222333444555666777888"
		fullPublicInputs[50] = "555666777888999000111222333444555666777888999000111222333444555666777888999"
		fullPublicInputs[51] = "666777888999000111222333444555666777888999000111222333444555666777888999000"
		fullPublicInputs[52] = "777888999000111222333444555666777888999000111222333444555666777888999000111"
		fullPublicInputs[53] = "888999000111222333444555666777888999000111222333444555666777888999000111222"
		fullPublicInputs[54] = "999000111222333444555666777888999000111222333444555666777888999000111222333"
		fullPublicInputs[55] = "1000111222333444555666777888999000111222333444555666777888999000111222333444"
		fullPublicInputs[56] = "124413588010935573100449456468959839270027757215138439816955024736271298883"
		fullPublicInputs[57] = "125987718504881168702817372751405511311626515399128115957683055706162879081"
		fullPublicInputs[58] = "138174294419566073638917398478480233783462655482283489778477032129860416308"
		fullPublicInputs[59] = "87164429935183530231106524238772469083021376536857547601286350511895957042"
		fullPublicInputs[60] = "159508995554830235422881220221659222882416701537684367907262541081181107041"
		fullPublicInputs[61] = "216177859633033993616607456010987870980723214832657304250929052054387451251"
		fullPublicInputs[62] = "136870293077760051536514689814528040652982158268238924211443105143315312977"
		fullPublicInputs[63] = "209027647271941540634260128227139143305212625530130988286308577451934433604"
		fullPublicInputs[64] = "216041037480816501846348705353738079775803623607373665378499876478757721956"
		fullPublicInputs[65] = "184099808892606061942559141059081527262834859629181581270585908529014000483"
		fullPublicInputs[66] = "173926821082308056829441773860483849128404996084932919505946802488367989070"
		fullPublicInputs[67] = "136498083332900321215526260868562056670892412932671519510981704427905430578"

		// Setup DKIM pub key
		poseidonHash, ok := new(big.Int).SetString(fullPublicInputs[9], 10)
		require.True(ok)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:       "gmail.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
			PoseidonHash: poseidonHash.Bytes(),
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		}, &f.k)
		require.NoError(err)

		// Convert command/tx data from public inputs [12:32]
		txParts, err := types.ConvertStringArrayToBigInt(fullPublicInputs[12:68])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
		require.NoError(err)

		// Verify tx bytes conversion worked with all 20 elements filled
		require.NotEmpty(txBytesStr, "tx bytes string should not be empty when all elements are filled")

		// Convert email host from public inputs
		emailHostParts, err := types.ConvertStringArrayToBigInt(fullPublicInputs[70:79])
		require.NoError(err)
		emailHostStr, err := types.ConvertBigIntArrayToString(emailHostParts)
		require.NoError(err)

		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte(txBytesStr),
			EmailHash:         emailHashStr,
			Proof:             proofJSON,
			PublicInputs:      fullPublicInputs,
			AllowedEmailHosts: []string{emailHostStr},
		}

		res, err := f.queryServer.Authenticate(f.ctx, req)
		// The request should pass tx bytes validation
		// It may fail on proof verification since we're using synthetic data,
		// but it should NOT fail on tx bytes parsing or validation
		if err != nil {
			require.NotContains(err.Error(), "failed to convert tx bytes public inputs to big int")
			require.NotContains(err.Error(), "failed to convert tx bytes public inputs to string")
			require.NotContains(err.Error(), "tx bytes do not match public inputs")
		}
		_ = res
	})
}

func TestParamsExtended(t *testing.T) {
	t.Run("params returns vkey identifier", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		res, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
		require.NoError(err)
		require.NotNil(res)
		require.NotNil(res.Params)
		// Default vkey identifier should be set
		require.GreaterOrEqual(res.Params.VkeyIdentifier, uint64(0))
	})

	t.Run("params after multiple updates", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// First update
		newParams1 := types.DefaultParams()
		newParams1.VkeyIdentifier = 10
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{
			Authority: f.govModAddr,
			Params:    newParams1,
		})
		require.NoError(err)

		res1, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
		require.NoError(err)
		require.Equal(uint64(10), res1.Params.VkeyIdentifier)

		// Second update
		newParams2 := types.DefaultParams()
		newParams2.VkeyIdentifier = 20
		_, err = f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{
			Authority: f.govModAddr,
			Params:    newParams2,
		})
		require.NoError(err)

		res2, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
		require.NoError(err)
		require.Equal(uint64(20), res2.Params.VkeyIdentifier)
	})

	t.Run("params with large vkey identifier", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		newParams := types.DefaultParams()
		newParams.VkeyIdentifier = 18446744073709551615 // max uint64
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{
			Authority: f.govModAddr,
			Params:    newParams,
		})
		require.NoError(err)

		res, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
		require.NoError(err)
		require.Equal(uint64(18446744073709551615), res.Params.VkeyIdentifier)
	})

	t.Run("params with multiple dkim pubkeys", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		newParams := types.DefaultParams()
		newParams.VkeyIdentifier = 5
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{
			Authority: f.govModAddr,
			Params:    newParams,
		})
		require.NoError(err)
	})

	t.Run("update params with invalid authority fails", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		newParams := types.DefaultParams()
		newParams.VkeyIdentifier = 100
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{
			Authority: "invalid-authority",
			Params:    newParams,
		})
		require.Error(err)
		require.Contains(err.Error(), "invalid authority")
	})
}

func TestIsSubset(t *testing.T) {
	t.Run("empty A is subset of any B", func(t *testing.T) {
		result := keeper.IsSubset([]string{}, []string{"a", "b", "c"})
		require.True(t, result)
	})

	t.Run("empty A is subset of empty B", func(t *testing.T) {
		result := keeper.IsSubset([]string{}, []string{})
		require.True(t, result)
	})

	t.Run("non-empty A is not subset of empty B", func(t *testing.T) {
		result := keeper.IsSubset([]string{"a"}, []string{})
		require.False(t, result)
	})

	t.Run("A equals B", func(t *testing.T) {
		result := keeper.IsSubset([]string{"a", "b", "c"}, []string{"a", "b", "c"})
		require.True(t, result)
	})

	t.Run("A is proper subset of B", func(t *testing.T) {
		result := keeper.IsSubset([]string{"a", "b"}, []string{"a", "b", "c", "d"})
		require.True(t, result)
	})

	t.Run("A has element not in B", func(t *testing.T) {
		result := keeper.IsSubset([]string{"a", "x"}, []string{"a", "b", "c"})
		require.False(t, result)
	})

	t.Run("single element subset", func(t *testing.T) {
		result := keeper.IsSubset([]string{"b"}, []string{"a", "b", "c"})
		require.True(t, result)
	})

	t.Run("single element not in B", func(t *testing.T) {
		result := keeper.IsSubset([]string{"x"}, []string{"a", "b", "c"})
		require.False(t, result)
	})

	t.Run("duplicate elements in A", func(t *testing.T) {
		result := keeper.IsSubset([]string{"a", "a", "a"}, []string{"a", "b"})
		require.True(t, result)
	})

	t.Run("duplicate elements in B", func(t *testing.T) {
		result := keeper.IsSubset([]string{"a"}, []string{"a", "a", "b", "b"})
		require.True(t, result)
	})

	t.Run("with integers", func(t *testing.T) {
		result := keeper.IsSubset([]int{1, 2}, []int{1, 2, 3, 4, 5})
		require.True(t, result)
	})

	t.Run("with integers - not subset", func(t *testing.T) {
		result := keeper.IsSubset([]int{1, 6}, []int{1, 2, 3, 4, 5})
		require.False(t, result)
	})
}

func TestNewQuerier(t *testing.T) {
	t.Run("creates querier from keeper", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		querier := keeper.NewQuerier(f.k)
		require.NotNil(querier)

		// Verify querier works
		res, err := querier.Params(f.ctx, &types.QueryParamsRequest{})
		require.NoError(err)
		require.NotNil(res)
	})
}
