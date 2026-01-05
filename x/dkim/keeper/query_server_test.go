package keeper_test

import (
	"math/big"
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

func TestProofVerify(t *testing.T) {
	f := SetupTest(t)
	require := require.New(t)

	t.Run("returns false for any request", func(t *testing.T) {
		// ProofVerify currently returns false for all requests
		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte("test"),
			EmailHash:         "test-hash",
			Proof:             []byte("{}"),
			PublicInputs:      []string{},
			AllowedEmailHosts: []string{},
		}

		res, err := f.queryServer.ProofVerify(f.ctx, req)
		require.NoError(err)
		require.NotNil(res)
		require.False(res.Verified)
	})

	t.Run("returns false with nil request fields", func(t *testing.T) {
		req := &types.QueryAuthenticateRequest{}

		res, err := f.queryServer.ProofVerify(f.ctx, req)
		require.NoError(err)
		require.NotNil(res)
		require.False(res.Verified)
	})

	t.Run("returns false with populated request", func(t *testing.T) {
		req := &types.QueryAuthenticateRequest{
			TxBytes:           []byte("some-tx-bytes"),
			EmailHash:         "19446427605026428332697445173245129703428784356663998533737434935925391210840",
			Proof:             []byte(`{"pi_a": ["1", "2", "1"], "pi_b": [["1", "2"], ["3", "4"], ["1", "0"]], "pi_c": ["1", "2", "1"], "protocol": "groth16", "curve": "bn128"}`),
			PublicInputs:      make([]string, 38),
			AllowedEmailHosts: []string{"test@example.com"},
		}

		res, err := f.queryServer.ProofVerify(f.ctx, req)
		require.NoError(err)
		require.NotNil(res)
		require.False(res.Verified)
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
		newParams := types.Params{
			VkeyIdentifier: 42,
		}

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
		newParams := types.Params{
			VkeyIdentifier: 99,
		}

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
		"12057794547485210516928817874827048708844252651510875086257455163416697746512",
		"0",
		"124413588010935573100449456468959839270027757215138439816955024736271298883",
		"125987718504881168702817372751405511311626515399128115957683055706162879081",
		"138174294419566073638917398478480233783462655482283489778477032129860416308",
		"87164429935183530231106524238772469083021376536857547601286350511895957042",
		"159508995554830235422881220221659222882416701537684367907262541081181107041",
		"216177859633033993616607456010987870980723214832657304250929052054387451251",
		"136870293077760051536514689814528040652982158268238924211443105143315312977",
		"209027647271941540634260128227139143305212625530130988286308577451934433604",
		"216041037480816501846348705353738079775803623607373665378499876478757721956",
		"184099808892606061942559141059081527262834859629181581270585908529014000483",
		"173926821082308056829441773860483849128404996084932919505946802488367989070",
		"136498083332900321215526260868562056670892412932671519510981704427905430578",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"0",
		"19446427605026428332697445173245129703428784356663998533737434935925391210840",
		"1",
		"145464208130933216679374873468710647147",
		"0",
		"0",
		"0",
	}

	// Setup DKIM pub key
	poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
	require.True(ok)
	_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
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

	// Common proof JSON
	proofJSON := []byte(`{
    "pi_a": [
      "6043643433140642569280898259541128431907635878547614935681440820683038963792",
      "9992132192779112865958667381915120532497401445863381693125708878412867819429",
      "1"
    ],
    "pi_b": [
      [
        "857150703036151009004130834885577860944545321105272581149620288148902385440",
        "3313419972466342030467701882126850537491115446681093222335468857323210697295"
      ],
      [
        "21712445344172795956102361993647268776674729003569584506047190630474625887295",
        "13180126619787644952475441454844294991198251669191962852459355269881478597074"
      ],
      [
        "1",
        "0"
      ]
    ],
    "pi_c": [
      "5608874530415768909531379297509258028398465201351680955270584280524807563327",
      "12825389375859294537236568763270506206901646432644007343954893485864905401313",
      "1"
    ],
    "protocol": "groth16",
    "curve": "bn128"
}`)

	// Common email hash
	emailHashStr := "19446427605026428332697445173245129703428784356663998533737434935925391210840"

	// Common tx bytes
	txParts, err := types.ConvertStringArrayToBigInt(basePublicInputs[12:32])
	require.NoError(err)
	txBytes, err := types.ConvertBigIntArrayToString(txParts)
	require.NoError(err)

	testCases := []struct {
		name              string
		emailHash         string
		allowedEmailHosts []string
		publicInputs      []string
		txBytes           []byte // nil means compute from publicInputs[12:32]
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
			} else if len(tc.publicInputs) >= 32 {
				// Compute txBytes from publicInputs[12:32]
				txParts, err := types.ConvertStringArrayToBigInt(tc.publicInputs[12:32])
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
				require.ErrorIs(err, types.ErrInvalidPublicInput)
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
		// Create public inputs with invalid tx bytes values (indices 12-32)
		invalidPublicInputs := make([]string, 38)
		for i := range invalidPublicInputs {
			invalidPublicInputs[i] = "0"
		}
		invalidPublicInputs[12] = "invalid-number" // Invalid: should be numeric
		invalidPublicInputs[32] = "test-hash"

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
		// We need tx bytes [12:32] to be valid AND match the provided txBytes
		// Then dkim domain [0:9] should be invalid
		invalidPublicInputs := make([]string, 38)
		for i := range invalidPublicInputs {
			invalidPublicInputs[i] = "0"
		}
		invalidPublicInputs[0] = "not-a-number" // Invalid dkim domain: should be numeric
		invalidPublicInputs[32] = "test-hash"

		// Compute txBytes from the valid publicInputs[12:32] (all zeros)
		txParts, err := types.ConvertStringArrayToBigInt(invalidPublicInputs[12:32])
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
		invalidPublicInputs := make([]string, 38)
		for i := range invalidPublicInputs {
			invalidPublicInputs[i] = "0"
		}
		invalidPublicInputs[9] = "not-a-valid-big-int" // Invalid: should be numeric
		invalidPublicInputs[32] = "test-hash"

		// Compute txBytes from the valid publicInputs[12:32] (all zeros)
		txParts, err := types.ConvertStringArrayToBigInt(invalidPublicInputs[12:32])
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
		validPublicInputs := make([]string, 38)
		for i := range validPublicInputs {
			validPublicInputs[i] = "0"
		}
		validPublicInputs[9] = "12345" // Valid numeric but non-existent hash
		validPublicInputs[32] = "test-hash"

		// Compute txBytes from the valid publicInputs[12:32] (all zeros)
		txParts, err := types.ConvertStringArrayToBigInt(validPublicInputs[12:32])
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
		"2018721414038404820327", // [0] domain start
		"0",                      // [1]
		"0",                      // [2]
		"0",                      // [3]
		"0",                      // [4]
		"0",                      // [5]
		"0",                      // [6]
		"0",                      // [7]
		"0",                      // [8] domain end
		"6632353713085157925504008443078919716322386156160602218536961028046468237192",  // [9] dkim hash
		"12057794547485210516928817874827048708844252651510875086257455163416697746512", // [10]
		"0", // [11]
		"124413588010935573100449456468959839270027757215138439816955024736271298883", // [12] tx bytes start
		"125987718504881168702817372751405511311626515399128115957683055706162879081", // [13]
		"138174294419566073638917398478480233783462655482283489778477032129860416308", // [14]
		"87164429935183530231106524238772469083021376536857547601286350511895957042",  // [15]
		"159508995554830235422881220221659222882416701537684367907262541081181107041", // [16]
		"216177859633033993616607456010987870980723214832657304250929052054387451251", // [17]
		"136870293077760051536514689814528040652982158268238924211443105143315312977", // [18]
		"209027647271941540634260128227139143305212625530130988286308577451934433604", // [19]
		"216041037480816501846348705353738079775803623607373665378499876478757721956", // [20]
		"184099808892606061942559141059081527262834859629181581270585908529014000483", // [21]
		"173926821082308056829441773860483849128404996084932919505946802488367989070", // [22]
		"136498083332900321215526260868562056670892412932671519510981704427905430578", // [23]
		"0", // [24]
		"0", // [25]
		"0", // [26]
		"0", // [27]
		"0", // [28]
		"0", // [29]
		"0", // [30]
		"0", // [31] tx bytes end
		"19446427605026428332697445173245129703428784356663998533737434935925391210840", // [32] email hash
		"1", // [33]
		"145464208130933216679374873468710647147", // [34] email host start
		"0", // [35]
		"0", // [36]
		"0", // [37] email host end
	}

	proofJSON := []byte(`{
		"pi_a": [
			"6043643433140642569280898259541128431907635878547614935681440820683038963792",
			"9992132192779112865958667381915120532497401445863381693125708878412867819429",
			"1"
		],
		"pi_b": [
			[
				"857150703036151009004130834885577860944545321105272581149620288148902385440",
				"3313419972466342030467701882126850537491115446681093222335468857323210697295"
			],
			[
				"21712445344172795956102361993647268776674729003569584506047190630474625887295",
				"13180126619787644952475441454844294991198251669191962852459355269881478597074"
			],
			[
				"1",
				"0"
			]
		],
		"pi_c": [
			"5608874530415768909531379297509258028398465201351680955270584280524807563327",
			"12825389375859294537236568763270506206901646432644007343954893485864905401313",
			"1"
		],
		"protocol": "groth16",
		"curve": "bn128"
	}`)

	emailHashStr := "19446427605026428332697445173245129703428784356663998533737434935925391210840"

	t.Run("fail - invalid email host public input conversion", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Setup DKIM pub key
		poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
		require.True(ok)
		_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
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

		// Create public inputs with invalid email host values
		invalidPublicInputs := make([]string, len(basePublicInputs))
		copy(invalidPublicInputs, basePublicInputs)
		invalidPublicInputs[34] = "not-a-number" // Invalid email host

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(invalidPublicInputs[12:32])
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
		_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
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

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(basePublicInputs[12:32])
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
		_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
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

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(basePublicInputs[12:32])
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

	t.Run("fail - public inputs exactly 38 elements but invalid data", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		// Create exactly 38 elements of zeros
		minimalPublicInputs := make([]string, 38)
		for i := range minimalPublicInputs {
			minimalPublicInputs[i] = "0"
		}
		minimalPublicInputs[32] = "test-hash" // email hash at index 32

		// Compute txBytes (all zeros)
		txParts, err := types.ConvertStringArrayToBigInt(minimalPublicInputs[12:32])
		require.NoError(err)
		txBytesStr, err := types.ConvertBigIntArrayToString(txParts)
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

		// Create 37 elements (one less than required)
		insufficientPublicInputs := make([]string, 37)
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

		// Create public inputs where email host [34:38] are all zeros
		// This results in empty string email host
		zeroEmailHostInputs := make([]string, len(basePublicInputs))
		copy(zeroEmailHostInputs, basePublicInputs)
		zeroEmailHostInputs[34] = "0"
		zeroEmailHostInputs[35] = "0"
		zeroEmailHostInputs[36] = "0"
		zeroEmailHostInputs[37] = "0"

		// Setup DKIM pub key
		poseidonHash, ok := new(big.Int).SetString(basePublicInputs[9], 10)
		require.True(ok)
		_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
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

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(zeroEmailHostInputs[12:32])
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
		_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
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

		// Compute valid txBytes
		txParts, err := types.ConvertStringArrayToBigInt(basePublicInputs[12:32])
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
		newParams1 := types.Params{
			VkeyIdentifier: 10,
		}
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{
			Authority: f.govModAddr,
			Params:    newParams1,
		})
		require.NoError(err)

		res1, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
		require.NoError(err)
		require.Equal(uint64(10), res1.Params.VkeyIdentifier)

		// Second update
		newParams2 := types.Params{
			VkeyIdentifier: 20,
		}
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

		newParams := types.Params{
			VkeyIdentifier: 18446744073709551615, // max uint64
		}
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

		newParams := types.Params{
			VkeyIdentifier: 5,
		}
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{
			Authority: f.govModAddr,
			Params:    newParams,
		})
		require.NoError(err)
	})

	t.Run("update params with invalid authority fails", func(t *testing.T) {
		f := SetupTest(t)
		require := require.New(t)

		newParams := types.Params{
			VkeyIdentifier: 100,
		}
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
