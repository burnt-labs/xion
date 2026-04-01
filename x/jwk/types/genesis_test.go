package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

// validRSAKey is a well-formed RSA JWK with RS256 used across genesis tests.
const validRSAKey = `{"kty":"RSA","use":"sig","kid":"test","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw","e":"AQAB"}`

func TestGenesisState_Validate(t *testing.T) {
	adminAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Aud:   "0",
						Admin: adminAddr,
					},
					{
						Aud:   "1",
						Admin: adminAddr,
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "valid genesis state with JWK key",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Aud:   "test-audience",
						Admin: adminAddr,
						Key:   validRSAKey,
					},
				},
			},
			valid: true,
		},
		{
			desc: "duplicated audience",
			genState: &types.GenesisState{
				AudienceList: []types.Audience{
					{
						Aud:   "0",
						Admin: adminAddr,
					},
					{
						Aud:   "0",
						Admin: adminAddr,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid admin address",
			genState: &types.GenesisState{
				AudienceList: []types.Audience{
					{
						Aud:   "test-audience",
						Admin: "invalid-address",
					},
				},
			},
			valid: false,
		},
		{
			desc: "key exceeding MaxJWKKeySize is rejected before parsing",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Aud:   "big-key-audience",
						Admin: adminAddr,
						Key:   strings.Repeat("a", types.MaxJWKKeySize+1),
					},
				},
			},
			valid: false,
		},
		{
			desc: "HMAC HS256 key is rejected",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Aud:   "hmac-audience",
						Admin: adminAddr,
						Key:   `{"kty":"oct","alg":"HS256","k":"AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow"}`,
					},
				},
			},
			valid: false,
		},
		{
			desc: "HMAC HS384 key is rejected",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Aud:   "hmac384-audience",
						Admin: adminAddr,
						Key:   `{"kty":"oct","alg":"HS384","k":"AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow"}`,
					},
				},
			},
			valid: false,
		},
		{
			desc: "HMAC HS512 key is rejected",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Aud:   "hmac512-audience",
						Admin: adminAddr,
						Key:   `{"kty":"oct","alg":"HS512","k":"AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow"}`,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid JWK format is rejected",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Aud:   "bad-key-audience",
						Admin: adminAddr,
						Key:   `{"not":"a valid jwk"}`,
					},
				},
			},
			valid: false,
		},
		{
			desc: "RSA private key in genesis is rejected",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Aud:   "privkey-audience",
						Admin: adminAddr,
						// Minimal RSA-2048 private key JWK (RFC 7517 §C.2 test vector)
						Key: `{"kty":"RSA","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw","e":"AQAB","d":"X4cTteJY_gn4FYPsXB8rdXix5vwsg1FLN5E3EaG6RJoVH-HLLKD9M7dx5oo7GURknchnrRweUkC7hT5fJLM0WbFAKNLWY2vv7B6NqXSzUvxT0_YSfqijwp3RTzlBaCxWp4doFk5N2o8Gy1XEIAwyBLlypnARQj9PJWQ","p":"83i-7IvMGXoMXCskv73TKr8637FiO7Z27zv8oj6pbWUQyLPQBQxtPVnwD20R-60eTDmD2ujnMt5PoqMrm8RfmNhVWDtjjMmCMjOpSXicFHj7XOuVIYQyqVWlWEh6dN36GVZYk93N8Bc9vY41xy8B9RzzOGVQzXvNEvn7O0nVbfs","q":"3dfOR9cuYq-0S-mkFLzgItgMEfFzB2q3hWehMuG0oCuqnb3vobLyumqjVZQO1dIrdwgTnCdpYzBcOfW5r370AFXjiWft_NGEiovonizhKpo9VVS78TzFgxkIdrecRezsZ-1kYd_s1qDbxtkDEgfAITAG9LUnADun4vIcb6yelIU","dp":"G4sPXkc6Ya9y8oJW9_ILj4xuppu0lzi_H7VTkS8xj5SdX3coE0oimYwxIi2emRAse6Gha0U7U_6c8WKrPa5kC3oXl2C7B8Vx2SVKGF-3CYC7U0_bvhK8hWq2NMDW5Ww","dq":"s9lAH9fggBsoFR33509CCVY1hc_2kflF8KHCzwF4YjEm0-4T5UNuFKlsYUkQdQg1QX2Rz2nBHiWPK7T6Ks_YQ","qi":"GyM_p6JrXySiz1toFgKbWV-JdI3jT4s9TwMvhLVQhP6z7PK1Y7iIw3RvGQWFfvhqYbYHCMN4TNHF8vxSBrY"}`,
					},
				},
			},
			valid: false,
		},
		{
			desc: "kty/alg mismatch in genesis is rejected",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Aud:   "mismatch-audience",
						Admin: adminAddr,
						// RSA public key with EC algorithm — kty=RSA but alg=ES256
						Key: `{"kty":"RSA","alg":"ES256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw","e":"AQAB"}`,
					},
				},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
