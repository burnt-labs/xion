package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestMsgCreateAudience(t *testing.T) {
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	validKey := `{"kty":"RSA","use":"sig","kid":"test","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw","e":"AQAB"}`

	tests := []struct {
		name      string
		admin     string
		aud       string
		key       string
		expectErr bool
	}{
		{
			name:      "valid message",
			admin:     admin,
			aud:       "test-audience",
			key:       validKey,
			expectErr: false,
		},
		{
			name:      "invalid admin address",
			admin:     "invalid-address",
			aud:       "test-audience",
			key:       validKey,
			expectErr: true,
		},
		{
			name:      "empty admin",
			admin:     "",
			aud:       "test-audience",
			key:       validKey,
			expectErr: true,
		},
		{
			name:      "invalid key format",
			admin:     admin,
			aud:       "test-audience",
			key:       "invalid-key",
			expectErr: true,
		},
		{
			name:      "symmetric key algorithm (invalid)",
			admin:     admin,
			aud:       "test-audience",
			key:       `{"kty":"oct","alg":"HS256","k":"AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow"}`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := types.NewMsgCreateAudience(tt.admin, tt.aud, tt.key)

			// Test basic properties
			require.Equal(t, types.RouterKey, msg.Route())
			require.Equal(t, types.TypeMsgCreateAudience, msg.Type())
			require.Equal(t, tt.admin, msg.Admin)
			require.Equal(t, tt.aud, msg.Aud)
			require.Equal(t, tt.key, msg.Key)

			// Test GetSignBytes
			signBytes := msg.GetSignBytes()
			require.NotNil(t, signBytes)

			// Test GetSigners
			if tt.admin != "" && tt.admin != "invalid-address" {
				signers := msg.GetSigners()
				require.Len(t, signers, 1)
				require.Equal(t, tt.admin, signers[0].String())
			} else if tt.admin == "invalid-address" {
				require.Panics(t, func() {
					msg.GetSigners()
				})
			}

			// Test ValidateBasic
			err := msg.ValidateBasic()
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgUpdateAudience(t *testing.T) {
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	newAdmin := authtypes.NewModuleAddress("test").String()
	validKey := `{"kty":"RSA","use":"sig","kid":"test","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw","e":"AQAB"}`

	msg := types.NewMsgUpdateAudience(admin, newAdmin, "old-aud", "new-aud", validKey)

	// Test basic properties
	require.Equal(t, types.RouterKey, msg.Route())
	require.Equal(t, types.TypeMsgUpdateAudience, msg.Type())
	require.Equal(t, admin, msg.Admin)
	require.Equal(t, newAdmin, msg.NewAdmin)
	require.Equal(t, "old-aud", msg.Aud)
	require.Equal(t, "new-aud", msg.NewAud)
	require.Equal(t, validKey, msg.Key)

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)

	// Test GetSigners
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, admin, signers[0].String())

	// Test ValidateBasic
	err := msg.ValidateBasic()
	require.NoError(t, err)

	// Test with invalid admin
	invalidMsg := types.NewMsgUpdateAudience("invalid", newAdmin, "old-aud", "new-aud", validKey)
	err = invalidMsg.ValidateBasic()
	require.Error(t, err)
}

func TestMsgDeleteAudience(t *testing.T) {
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	msg := types.NewMsgDeleteAudience(admin, "test-audience")

	// Test basic properties
	require.Equal(t, types.RouterKey, msg.Route())
	require.Equal(t, types.TypeMsgDeleteAudience, msg.Type())
	require.Equal(t, admin, msg.Admin)
	require.Equal(t, "test-audience", msg.Aud)

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)

	// Test GetSigners
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, admin, signers[0].String())

	// Test ValidateBasic
	err := msg.ValidateBasic()
	require.NoError(t, err)

	// Test with invalid admin
	invalidMsg := types.NewMsgDeleteAudience("invalid", "test-audience")
	err = invalidMsg.ValidateBasic()
	require.Error(t, err)
}

func TestMsgCreateAudienceClaim(t *testing.T) {
	adminAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	admin := adminAddr.String()
	// Create a proper 32-byte hash (SHA256)
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i % 256)
	}

	msg := types.NewMsgCreateAudienceClaim(adminAddr, hash)

	// Test basic properties
	require.Equal(t, types.RouterKey, msg.Route())
	require.Equal(t, types.TypeMsgCreateAudienceClaim, msg.Type())
	require.Equal(t, admin, msg.Admin)
	require.Equal(t, hash, msg.AudHash)

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)

	// Test GetSigners
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, admin, signers[0].String())

	// Test ValidateBasic
	err := msg.ValidateBasic()
	require.NoError(t, err)

	// Test with invalid admin (create directly to test validation)
	invalidMsg := &types.MsgCreateAudienceClaim{
		Admin:   "invalid",
		AudHash: hash,
	}
	err = invalidMsg.ValidateBasic()
	require.Error(t, err)
}

func TestMsgDeleteAudienceClaim(t *testing.T) {
	adminAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	admin := adminAddr.String()
	hash := []byte("test-hash")

	msg := types.NewMsgDeleteAudienceClaim(adminAddr, hash)

	// Test basic properties
	require.Equal(t, types.RouterKey, msg.Route())
	require.Equal(t, types.TypeMsgDeleteAudienceClaim, msg.Type())
	require.Equal(t, admin, msg.Admin)
	require.Equal(t, hash, msg.AudHash)

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)

	// Test GetSigners
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, admin, signers[0].String())

	// Test ValidateBasic
	err := msg.ValidateBasic()
	require.NoError(t, err)

	// Test with invalid admin (create directly to test validation)
	invalidMsg := &types.MsgDeleteAudienceClaim{
		Admin:   "invalid",
		AudHash: hash,
	}
	err = invalidMsg.ValidateBasic()
	require.Error(t, err)

	// Test GetSigners with invalid address (should panic)
	require.Panics(t, func() {
		invalidMsg.GetSigners()
	})
}

func TestMsgGetSignersPanics(t *testing.T) {
	// Test GetSigners panics for UpdateAudience
	updateMsg := &types.MsgUpdateAudience{
		Admin: "invalid-address",
	}
	require.Panics(t, func() {
		updateMsg.GetSigners()
	})

	// Test GetSigners panics for DeleteAudience
	deleteMsg := &types.MsgDeleteAudience{
		Admin: "invalid-address",
	}
	require.Panics(t, func() {
		deleteMsg.GetSigners()
	})

	// Test GetSigners panics for CreateAudienceClaim
	claimMsg := &types.MsgCreateAudienceClaim{
		Admin: "invalid-address",
	}
	require.Panics(t, func() {
		claimMsg.GetSigners()
	})

	// Test GetSigners panics for DeleteAudienceClaim
	deleteClaimMsg := &types.MsgDeleteAudienceClaim{
		Admin: "invalid-address",
	}
	require.Panics(t, func() {
		deleteClaimMsg.GetSigners()
	})
}

func TestMsgUpdateAudienceValidation(t *testing.T) {
	validAdmin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	validNewAdmin := authtypes.NewModuleAddress("test").String()

	tests := []struct {
		name    string
		msg     types.MsgUpdateAudience
		wantErr bool
		errType error
	}{
		{
			name: "valid message",
			msg: types.MsgUpdateAudience{
				Admin:    validAdmin,
				NewAdmin: validNewAdmin,
				Aud:      "test-audience",
				NewAud:   "new-audience",
				Key:      `{"kty":"RSA","use":"sig","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
			},
			wantErr: false,
		},
		{
			name: "invalid new admin address",
			msg: types.MsgUpdateAudience{
				Admin:    validAdmin,
				NewAdmin: "invalid-address",
				Aud:      "test-audience",
				Key:      `{"kty":"RSA","use":"sig","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
			},
			wantErr: true,
			errType: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid admin address",
			msg: types.MsgUpdateAudience{
				Admin:    "invalid-address",
				NewAdmin: validNewAdmin,
				Aud:      "test-audience",
				Key:      `{"kty":"RSA","use":"sig","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
			},
			wantErr: true,
			errType: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid key format",
			msg: types.MsgUpdateAudience{
				Admin:    validAdmin,
				NewAdmin: validNewAdmin,
				Aud:      "test-audience",
				Key:      `{"invalid":"key"}`,
			},
			wantErr: true,
			errType: types.ErrInvalidJWK,
		},
		{
			name: "symmetric key algorithm HS256 (invalid)",
			msg: types.MsgUpdateAudience{
				Admin:    validAdmin,
				NewAdmin: validNewAdmin,
				Aud:      "test-audience",
				Key:      `{"kty":"oct","use":"sig","alg":"HS256","k":"test-key"}`,
			},
			wantErr: true,
		},
		{
			name: "no signature algorithm (invalid)",
			msg: types.MsgUpdateAudience{
				Admin:    validAdmin,
				NewAdmin: validNewAdmin,
				Aud:      "test-audience",
				Key:      `{"kty":"RSA","use":"sig","alg":"none","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
			},
			wantErr: true,
		},
		{
			name: "malformed JSON key",
			msg: types.MsgUpdateAudience{
				Admin:    validAdmin,
				NewAdmin: validNewAdmin,
				Aud:      "test-audience",
				Key:      `{"kty":"RSA","malformed"`,
			},
			wantErr: true,
		},
		{
			name: "invalid algorithm type",
			msg: types.MsgUpdateAudience{
				Admin:    validAdmin,
				NewAdmin: validNewAdmin,
				Aud:      "test-audience",
				Key:      `{"kty":"RSA","use":"sig","alg":"INVALID","n":"test","e":"AQAB"}`,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgCreateAudienceValidationEdgeCases(t *testing.T) {
	validAdmin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	tests := []struct {
		name    string
		msg     types.MsgCreateAudience
		wantErr bool
	}{
		{
			name: "key with unknown algorithm",
			msg: types.MsgCreateAudience{
				Admin: validAdmin,
				Aud:   "test-aud",
				Key:   `{"kty":"RSA","use":"sig","alg":"UNKNOWN","n":"test","e":"AQAB"}`,
			},
			wantErr: true,
		},
		{
			name: "empty aud field",
			msg: types.MsgCreateAudience{
				Admin: validAdmin,
				Aud:   "",
				Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"test","e":"AQAB"}`,
			},
			wantErr: false, // Empty aud is allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgCreateAudienceClaimValidation(t *testing.T) {
	validAdmin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	tests := []struct {
		name    string
		msg     types.MsgCreateAudienceClaim
		wantErr bool
	}{
		{
			name: "valid 32-byte hash",
			msg: types.MsgCreateAudienceClaim{
				Admin:   validAdmin,
				AudHash: make([]byte, 32),
			},
			wantErr: false,
		},
		{
			name: "invalid hash length (31 bytes)",
			msg: types.MsgCreateAudienceClaim{
				Admin:   validAdmin,
				AudHash: make([]byte, 31),
			},
			wantErr: true,
		},
		{
			name: "invalid hash length (33 bytes)",
			msg: types.MsgCreateAudienceClaim{
				Admin:   validAdmin,
				AudHash: make([]byte, 33),
			},
			wantErr: true,
		},
		{
			name: "empty hash",
			msg: types.MsgCreateAudienceClaim{
				Admin:   validAdmin,
				AudHash: []byte{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
