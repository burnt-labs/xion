package keeper

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
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

// Reproduces the malformed AuthenticatorData case to ensure no panic occurs and an error is returned.
func TestWebAuthNVerifyRegister_DoesNotPanicOnMalformedAuthenticatorData(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx
	k := Keeper{storeKey: key}

	// Crafted raw authenticator data from PoC
	attestRawData := []byte("00000000000000000000000000000000\xe200000000000000000000\x00\x00\xf900")

	attestObj := protocol.AttestationObject{
		RawAuthData: attestRawData,
	}

	attestMarshal, err := webauthncbor.Marshal(attestObj)
	require.NoError(t, err)

	credentialJSON := map[string]interface{}{
		"id":    base64.RawURLEncoding.EncodeToString([]byte("toto")),
		"type":  "public-key",
		"rawId": "dG90bw==",
		"response": map[string]interface{}{
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString([]byte("{}")),
			"attestationObject": base64.RawURLEncoding.EncodeToString(attestMarshal),
		},
		"transports": []string{"joetkt"},
	}

	credentialJSONBytes, err := json.Marshal(credentialJSON)
	require.NoError(t, err)

	// Call the query method and ensure it returns an error, not a panic
	_, err = k.WebAuthNVerifyRegister(ctx, &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        "https://fuzzinglabs.com",
		Addr:      "contract",
		Challenge: "challenge",
		Data:      credentialJSONBytes,
	})
	require.Error(t, err)
}

// Reproduces the PoC for the Authenticate path to ensure no panic occurs and an error is returned.
func TestWebAuthNVerifyAuthenticate_DoesNotPanicOnMalformedAuthenticatorData(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx
	k := Keeper{storeKey: key}

	// Crafted raw authenticator data from PoC
	attestRawData := []byte("00000000000000000000000000000000\xe200000000000000000000\x00\x00\xf900")

	// Build a minimal assertion response JSON similar to the PoC
	jsonBody := map[string]interface{}{
		"id":    "AOx6vFGGITtlwjhqFFvAkJmBzSzfwE1dBa1fVR_Ltq5L35FJRNdgkXe84v3-0TEVNCSp",
		"rawId": "AOx6vFGGITtlwjhqFFvAkJmBzSzfwE1dBa1fVR_Ltq5L35FJRNdgkXe84v3-0TEVNCSp",
		"response": map[string]interface{}{
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString([]byte("{}")),
			"authenticatorData": base64.RawURLEncoding.EncodeToString(attestRawData),
			"signature":         nil,
			"userHandle":        nil,
		},
		"type":       "public-key",
		"transports": []string{"joetkt"},
	}

	jsonBz, err := json.Marshal(jsonBody)
	require.NoError(t, err)

	// Pass an empty credential object; we expect to fail prior to its usage
	_, err = k.WebAuthNVerifyAuthenticate(ctx, &types.QueryWebAuthNVerifyAuthenticateRequest{
		Rp:         "https://fuzzinglabs.com",
		Addr:       "contract",
		Challenge:  "challenge",
		Credential: []byte("{}"),
		Data:       jsonBz,
	})
	require.Error(t, err)
}
