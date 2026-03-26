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
