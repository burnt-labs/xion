package keeper

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"

	"github.com/burnt-labs/xion/x/xion/types"
)

// Tests exercise error paths via malformed inputs only.

// Tests exercise only basic error paths reachable via public API.
func TestWebAuthNQueries_ErrorPaths(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx
	// Minimal keeper with only store key (methods under test only access store via context if at all)
	k := Keeper{storeKey: key}
	_, err := k.WebAuthNVerifyRegister(ctx, &types.QueryWebAuthNVerifyRegisterRequest{Rp: "://bad", Addr: "a", Challenge: "c", Data: []byte("{}")})
	require.Error(t, err)
	_, err = k.WebAuthNVerifyAuthenticate(ctx, &types.QueryWebAuthNVerifyAuthenticateRequest{Rp: "://bad", Addr: "a", Challenge: "c", Data: []byte("{}")})
	require.Error(t, err)
	_, err = k.WebAuthNVerifyRegister(ctx, &types.QueryWebAuthNVerifyRegisterRequest{Rp: "https://ok", Addr: "a", Challenge: "c", Data: []byte(`{"some":1}`)})
	require.Error(t, err)
	_, err = k.WebAuthNVerifyAuthenticate(ctx, &types.QueryWebAuthNVerifyAuthenticateRequest{Rp: "https://ok", Addr: "a", Challenge: "c", Data: []byte(`{"some":1}`)})
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

func TestWebAuthNVerifyRegister_SizeLimit(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx
	k := Keeper{storeKey: key}

	// Test data within limit should pass validation step (may fail later for other reasons)
	smallData := make([]byte, 1024) // 1KB - within limit
	req1 := &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        "https://example.com",
		Addr:      "test_addr",
		Challenge: "test_challenge",
		Data:      smallData,
	}
	_, err := k.WebAuthNVerifyRegister(ctx, req1)
	// Should not fail due to size limit (may fail for other validation reasons)
	if err != nil {
		require.NotContains(t, err.Error(), "data size")
		require.NotContains(t, err.Error(), "exceeds maximum")
	}

	// Test data at exact limit should pass validation step
	limitData := make([]byte, types.MaxWebAuthDataSize)
	req2 := &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        "https://example.com",
		Addr:      "test_addr",
		Challenge: "test_challenge",
		Data:      limitData,
	}
	_, err = k.WebAuthNVerifyRegister(ctx, req2)
	// Should not fail due to size limit
	if err != nil {
		require.NotContains(t, err.Error(), "data size")
		require.NotContains(t, err.Error(), "exceeds maximum")
	}

	// Test data exceeding limit should be rejected immediately
	oversizeData := make([]byte, types.MaxWebAuthDataSize+1)
	req3 := &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        "https://example.com",
		Addr:      "test_addr",
		Challenge: "test_challenge",
		Data:      oversizeData,
	}
	_, err = k.WebAuthNVerifyRegister(ctx, req3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "data size")
	require.Contains(t, err.Error(), "exceeds maximum")

	t.Logf("Successfully rejected oversized data (%d bytes): %v", len(oversizeData), err)
}

func TestWebAuthNVerifyAuthenticate_SizeLimit(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx
	k := Keeper{storeKey: key}

	// Test data exceeding limit should be rejected immediately
	oversizeData := make([]byte, types.MaxWebAuthDataSize+1)
	req := &types.QueryWebAuthNVerifyAuthenticateRequest{
		Rp:         "https://example.com",
		Addr:       "test_addr",
		Challenge:  "test_challenge",
		Credential: []byte("{}"),
		Data:       oversizeData,
	}
	_, err := k.WebAuthNVerifyAuthenticate(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "data size")
	require.Contains(t, err.Error(), "exceeds maximum")

	t.Logf("Successfully rejected oversized auth data (%d bytes): %v", len(oversizeData), err)
}
