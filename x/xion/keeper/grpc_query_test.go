package keeper

import (
	"bytes"
	"encoding/base64"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Tests exercise error paths via malformed inputs only.

// Tests exercise only basic error paths reachable via public API.
func TestWebAuthNQueries_ErrorPaths(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx
	// Minimal keeper with only store key (methods under test only access store via context if at all)
	k := Keeper{storeKey: key}
	_, err := k.WebAuthNVerifyRegister(sdktypes.WrapSDKContext(ctx), &types.QueryWebAuthNVerifyRegisterRequest{Rp: "://bad", Addr: "a", Challenge: "c", Data: []byte("{}")})
	require.Error(t, err)
	_, err = k.WebAuthNVerifyAuthenticate(sdktypes.WrapSDKContext(ctx), &types.QueryWebAuthNVerifyAuthenticateRequest{Rp: "://bad", Addr: "a", Challenge: "c", Data: []byte("{}")})
	require.Error(t, err)
	_, err = k.WebAuthNVerifyRegister(sdktypes.WrapSDKContext(ctx), &types.QueryWebAuthNVerifyRegisterRequest{Rp: "https://ok", Addr: "a", Challenge: "c", Data: []byte(`{"some":1}`)})
	require.Error(t, err)
	_, err = k.WebAuthNVerifyAuthenticate(sdktypes.WrapSDKContext(ctx), &types.QueryWebAuthNVerifyAuthenticateRequest{Rp: "https://ok", Addr: "a", Challenge: "c", Data: []byte(`{"some":1}`)})
	require.Error(t, err)
}

// Additional focused tests to cover validation helpers.
func TestWebAuthN_ValidationHelpers(t *testing.T) {
	if err := validateCredentialCreation(bytes.NewReader([]byte("not-json"))); err == nil {
		t.Fatal("expected error")
	}
	if err := validateCredentialRequest(bytes.NewReader([]byte("not-json"))); err == nil {
		t.Fatal("expected error")
	}
}

// TestValidateCredentialRequestBranches covers all internal branches of validateCredentialRequest
func TestValidateCredentialRequestBranches(t *testing.T) {
	// 1. Decode error (invalid JSON)
	if err := validateCredentialRequest(bytes.NewReader([]byte("not-json"))); err == nil {
		t.Fatal("expected decode error")
	}

	// 2. Missing authenticator data -> explicit error branch
	// correct JSON uses "response" per struct tag
	missingAuth := []byte(`{"response":{}}`)
	if err := validateCredentialRequest(bytes.NewReader(missingAuth)); err == nil {
		t.Fatal("expected missing auth data error")
	}

	// 3. Success path: provide minimally valid authenticator data (37 bytes) base64 encoded inside JSON
	authData := make([]byte, 37)
	for i := 0; i < 32; i++ {
		authData[i] = 1
	}
	// flags (no extensions) already zeroed; counter bytes zeroed
	encoded := base64.RawStdEncoding.EncodeToString(authData)
	good := []byte(`{"response":{"authenticatorData":"` + encoded + `"}}`)
	if err := validateCredentialRequest(bytes.NewReader(good)); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}
