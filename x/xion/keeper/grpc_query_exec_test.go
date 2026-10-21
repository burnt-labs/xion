package keeper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

// A registration and a matching authentication captured from a browser
// against the relying party below, with attestation format "none". The same
// capture is used by the x/xion/types tests.
const (
	webAuthnRelyingParty = "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app"
	webAuthnAccount      = "xion1ncx0a0jnsyay7udd03ah2gf64772g02qswj52996dy80qfvgnmzq6eplqq"
	webAuthnLoginChal    = "MfaOZjuIdKFbXkLKWbPghSL8w41RsK2Issp4i0TwzvU="

	//nolint:gosec // G101: captured WebAuthn ceremony fixture, not a credential
	webAuthnRegisterFixture = `{"id":"UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg","type":"public-key","rawId":"VVd4WS15UmRJbHM4SVQtdnlNUzZsYTFaaXFFU09BZmY3YldaX0xXVjBQZw","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoiZUdsdmJqRnVZM2d3WVRCcWJuTjVZWGszZFdSa01ETmhhREpuWmpZME56Y3laekF5Y1hOM2FqVXlPVGsyWkhrNE1IRm1kbWR1YlhweE5tVndiSEZ4Iiwib3JpZ2luIjoiaHR0cHM6Ly94aW9uLWRhcHAtZXhhbXBsZS1naXQtZmVhdC1mYWNlaWQtYnVybnRmaW5hbmNlLnZlcmNlbC5hcHAifQ","attestationObject":"o2NmbXRkbm9uZWhBdXRoRGF0YaVkcnBpZFggsGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1lZmxhZ3MYRWhhdHRfZGF0YaNmYWFndWlkUEFBR1VJREFBR1VJREFBPT1qcHVibGljX2tleVkCEKQBAwM5AQAgWQIAolg7TF3aai-wR4HTDe5oR-WRhEsdW3u-O3IJHl0BiHkmR4MLskHG9HzivWoXsloUBnBMrFNxOH0x5cNMI07oi4PeRbHySiogRW9CXPjJaNlTi-pT_IgKFsyJNXsLyzrnajLkDbQU6pRsHmNeL0hAOUv48rtXv8VVWWN8okJehD2q9N7LHoFAOmIUEPg_VTHTt8K__O-9eMZKN4eMjh_4-sxRX6NXPSPT87XRlrK4GZ4pUdp86K0tOFLhwO4Uj0JkMNfI82eVZ1tAbDlqjd8jFnAb8fWm8wtdaTNbL_AAXmbDhswwJOyrw8fARZIhrXSdKBWa6e4k7sLwTIy-OO8saebnlARsjGst7ZCzmw5KCm2ctEVl3hYhHwyXu_A5rOblMrV3H0G7WqeKMCMVSJ11ssrlsmfVhNIwu1Qlt5GYmPTTJiCgGUGRxZkgDyOyjFNHglYpZamCGyJ9oyofsukEGoqMQ6WzjFi_hjVapzXi7Li-Q0OjEopIUUDDgeUrgjbGY0eiHI6sAz5hoaD0Qjc9e3Hk6-y7VcKCTCAanZOlJV0vJkHB98LBLh9qAoVUei_VaLFe2IcfVlrL_43aXlsHhr_SUQY5pHPlUMbQihE_57dpPRh31qDX_w6ye8dilniP8JmpKM2uIwnJ0x7hfJ45Qa0oLHmrGlzY9wi-RGP0YUkhQwEAAW1jcmVkZW50aWFsX2lkWCtVV3hZLXlSZElsczhJVC12eU1TNmxhMVppcUVTT0FmZjdiV1pfTFdWMFBnaGV4dF9kYXRh9mpzaWduX2NvdW50AGhhdXRoRGF0YVkCcrBjAYg3BKaYjH8UNdEzwntvhWiqy3k5L6c8XM54EzMNRQAAAABBQUdVSURBQUdVSURBQT09ACtVV3hZLXlSZElsczhJVC12eU1TNmxhMVppcUVTT0FmZjdiV1pfTFdWMFBnpAEDAzkBACBZAgCiWDtMXdpqL7BHgdMN7mhH5ZGESx1be747cgkeXQGIeSZHgwuyQcb0fOK9aheyWhQGcEysU3E4fTHlw0wjTuiLg95FsfJKKiBFb0Jc-Mlo2VOL6lP8iAoWzIk1ewvLOudqMuQNtBTqlGweY14vSEA5S_jyu1e_xVVZY3yiQl6EPar03ssegUA6YhQQ-D9VMdO3wr_87714xko3h4yOH_j6zFFfo1c9I9PztdGWsrgZnilR2nzorS04UuHA7hSPQmQw18jzZ5VnW0BsOWqN3yMWcBvx9abzC11pM1sv8ABeZsOGzDAk7KvDx8BFkiGtdJ0oFZrp7iTuwvBMjL447yxp5ueUBGyMay3tkLObDkoKbZy0RWXeFiEfDJe78Dms5uUytXcfQbtap4owIxVInXWyyuWyZ9WE0jC7VCW3kZiY9NMmIKAZQZHFmSAPI7KMU0eCVillqYIbIn2jKh-y6QQaioxDpbOMWL-GNVqnNeLsuL5DQ6MSikhRQMOB5SuCNsZjR6IcjqwDPmGhoPRCNz17ceTr7LtVwoJMIBqdk6UlXS8mQcH3wsEuH2oChVR6L9VosV7Yhx9WWsv_jdpeWweGv9JRBjmkc-VQxtCKET_nt2k9GHfWoNf_DrJ7x2KWeI_wmakoza4jCcnTHuF8njlBrSgseasaXNj3CL5EY_RhSSFDAQAB"}}`

	//nolint:gosec // G101: captured WebAuthn ceremony fixture, not a credential
	webAuthnAuthenticateFixture = `{"id":"UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg","type":"public-key","rawId":"VVd4WS15UmRJbHM4SVQtdnlNUzZsYTFaaXFFU09BZmY3YldaX0xXVjBQZw","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uZ2V0IiwiY2hhbGxlbmdlIjoiTWZhT1pqdUlkS0ZiWGtMS1diUGdoU0w4dzQxUnNLMklzc3A0aTBUd3p2VT0iLCJvcmlnaW4iOiJodHRwczovL3hpb24tZGFwcC1leGFtcGxlLWdpdC1mZWF0LWZhY2VpZC1idXJudGZpbmFuY2UudmVyY2VsLmFwcCJ9","authenticatorData":"sGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1FAAAAAEFBR1VJREFBR1VJREFBPT0AK1VXeFkteVJkSWxzOElULXZ5TVM2bGExWmlxRVNPQWZmN2JXWl9MV1YwUGekAQMDOQEAIFkCAKJYO0xd2movsEeB0w3uaEflkYRLHVt7vjtyCR5dAYh5JkeDC7JBxvR84r1qF7JaFAZwTKxTcTh9MeXDTCNO6IuD3kWx8koqIEVvQlz4yWjZU4vqU_yIChbMiTV7C8s652oy5A20FOqUbB5jXi9IQDlL-PK7V7_FVVljfKJCXoQ9qvTeyx6BQDpiFBD4P1Ux07fCv_zvvXjGSjeHjI4f-PrMUV-jVz0j0_O10ZayuBmeKVHafOitLThS4cDuFI9CZDDXyPNnlWdbQGw5ao3fIxZwG_H1pvMLXWkzWy_wAF5mw4bMMCTsq8PHwEWSIa10nSgVmunuJO7C8EyMvjjvLGnm55QEbIxrLe2Qs5sOSgptnLRFZd4WIR8Ml7vwOazm5TK1dx9Bu1qnijAjFUiddbLK5bJn1YTSMLtUJbeRmJj00yYgoBlBkcWZIA8jsoxTR4JWKWWpghsifaMqH7LpBBqKjEOls4xYv4Y1Wqc14uy4vkNDoxKKSFFAw4HlK4I2xmNHohyOrAM-YaGg9EI3PXtx5Ovsu1XCgkwgGp2TpSVdLyZBwffCwS4fagKFVHov1WixXtiHH1Zay_-N2l5bB4a_0lEGOaRz5VDG0IoRP-e3aT0Yd9ag1_8OsnvHYpZ4j_CZqSjNriMJydMe4XyeOUGtKCx5qxpc2PcIvkRj9GFJIUMBAAE","signature":"HoWSrIL-9keuWgvywoD9fxv-AMdGZdw7bYJP2cNnYv_0vKQ6iSmU3WVjE3MvdUDuruE9wYwIuZ-nqUve-56ZTBYmowzZ79PGgCUUNEFFScgH7ShD8McLK90XLKJGEyiTODPlFv2erCCi7pw2o9L3IWDK_B_yFlkYBkhkHI2h3kwcs8aDxcn_hMjHZonxYqm3eB4Syj-FNseCneVYUw8HljSyBVzrMpa4PkukUWTlo46p6HLoe51XMK_UPpXKFnutQkF_DPcwrUzWdgyEZe4B96TZazcRi8-EZtMRKDLrRgzQ1QYe6srqT74FDuMNI8w-0_aUQBUMWPvGGCHZOAUvQV-TnmY5tsAPFpYH5A0Wi5xHw6r5-Gvw9PZH5zss65zA1nHC085w9KGFjhBEkUE_TmzrZTBX6vogt4YIMinA-YxwGUJyF-gbM8-9BkElSSYY3OsAhwlYDERRAE_gw4hoWSNIf2gjZKH0RhLnZY6eViOiqEdnJWnVWbBVL3UMaYvcLvhNakh59OwB0DO2CEGZziw1qQJeN-3d9Rez7ef_gOO5zT1HSYIPHg9Br9z63e0C3abAsg1iNz8kWtvQ_mjypvCL28vaFoXrcYaUHZQogzaqEEGQ-zSwQK-NAsXI_ZKzYSXmbgAv0wFibBMCG_FzE_hYAGHKSQj9tsdxXicBinY","userHandle":"eGlvbjFuY3gwYTBqbnN5YXk3dWRkMDNhaDJnZjY0NzcyZzAycXN3ajUyOTk2ZHk4MHFmdmdubXpxNmVwbHFx"}}`
)

// The registration challenge is the account address, base64url encoded
// without padding, as the clientDataJSON of the fixture records it.
func webAuthnRegisterChallenge() string {
	return base64.RawURLEncoding.EncodeToString([]byte(webAuthnAccount))
}

func newWebAuthnKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	return Keeper{storeKey: key}, testCtx.Ctx.WithBlockTime(time.Now())
}

func registerWebAuthnFixture(t *testing.T, k Keeper, ctx sdk.Context) []byte {
	t.Helper()
	res, err := k.WebAuthNVerifyRegister(ctx, &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        webAuthnRelyingParty,
		Addr:      webAuthnAccount,
		Challenge: webAuthnRegisterChallenge(),
		Data:      []byte(webAuthnRegisterFixture),
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	return res.Credential
}

func TestWebAuthNVerifyRegister_ReturnsCredential(t *testing.T) {
	k, ctx := newWebAuthnKeeper(t)

	credentialBz := registerWebAuthnFixture(t, k, ctx)

	var credential webauthn.Credential
	require.NoError(t, json.Unmarshal(credentialBz, &credential))
	// The credential ID is the decoded rawId of the capture.
	rawID, err := base64.RawURLEncoding.DecodeString("VVd4WS15UmRJbHM4SVQtdnlNUzZsYTFaaXFFU09BZmY3YldaX0xXVjBQZw")
	require.NoError(t, err)
	require.Equal(t, rawID, credential.ID)
	require.NotEmpty(t, credential.PublicKey)
	require.Equal(t, "none", credential.AttestationType)
}

func TestWebAuthNVerifyRegister_RejectsWrongChallenge(t *testing.T) {
	k, ctx := newWebAuthnKeeper(t)

	_, err := k.WebAuthNVerifyRegister(ctx, &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        webAuthnRelyingParty,
		Addr:      webAuthnAccount,
		Challenge: "not-the-challenge-that-was-signed",
		Data:      []byte(webAuthnRegisterFixture),
	})
	require.ErrorIs(t, err, types.ErrNoValidWebAuth)
}

func TestWebAuthNVerifyRegister_ParseErrorAfterValidation(t *testing.T) {
	k, ctx := newWebAuthnKeeper(t)

	// Dropping the credential ID keeps the attestation object and client data
	// intact, so the structural pre-validation passes and the error comes from
	// the full parse of the creation response.
	withoutID := strings.Replace(webAuthnRegisterFixture, `"id":"UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg",`, "", 1)
	require.NotEqual(t, webAuthnRegisterFixture, withoutID)

	_, err := k.WebAuthNVerifyRegister(ctx, &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        webAuthnRelyingParty,
		Addr:      webAuthnAccount,
		Challenge: webAuthnRegisterChallenge(),
		Data:      []byte(withoutID),
	})
	require.ErrorIs(t, err, types.ErrNoValidWebAuth)
	require.Contains(t, err.Error(), "Parse error for Registration")
}

func TestWebAuthNVerifyRegister_RecoversFromPanic(t *testing.T) {
	k, _ := newWebAuthnKeeper(t)

	// A context that does not carry an sdk.Context makes the unwrap after
	// parsing panic; the handler must turn that into an error.
	res, err := k.WebAuthNVerifyRegister(context.Background(), &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        webAuthnRelyingParty,
		Addr:      webAuthnAccount,
		Challenge: webAuthnRegisterChallenge(),
		Data:      []byte(webAuthnRegisterFixture),
	})
	require.Nil(t, res)
	require.ErrorIs(t, err, types.ErrNoValidWebAuth)
	require.Contains(t, err.Error(), "panic during WebAuthn verification")
}

func TestWebAuthNVerifyAuthenticate_AcceptsValidAssertion(t *testing.T) {
	k, ctx := newWebAuthnKeeper(t)
	credentialBz := registerWebAuthnFixture(t, k, ctx)

	res, err := k.WebAuthNVerifyAuthenticate(ctx, &types.QueryWebAuthNVerifyAuthenticateRequest{
		Rp:         webAuthnRelyingParty,
		Addr:       webAuthnAccount,
		Challenge:  webAuthnLoginChal,
		Credential: credentialBz,
		Data:       []byte(webAuthnAuthenticateFixture),
	})
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestWebAuthNVerifyAuthenticate_RejectsWrongChallenge(t *testing.T) {
	k, ctx := newWebAuthnKeeper(t)
	credentialBz := registerWebAuthnFixture(t, k, ctx)

	_, err := k.WebAuthNVerifyAuthenticate(ctx, &types.QueryWebAuthNVerifyAuthenticateRequest{
		Rp:         webAuthnRelyingParty,
		Addr:       webAuthnAccount,
		Challenge:  "not-the-challenge-that-was-signed",
		Credential: credentialBz,
		Data:       []byte(webAuthnAuthenticateFixture),
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, types.ErrNoValidWebAuth)
	require.Contains(t, err.Error(), "challenge")
}

func TestWebAuthNVerifyAuthenticate_ParseErrorAfterValidation(t *testing.T) {
	k, ctx := newWebAuthnKeeper(t)

	// 37 bytes of authenticator data satisfy the structural pre-validation,
	// but the assertion has no credential ID, so the full parse rejects it.
	authData := base64.RawURLEncoding.EncodeToString(make([]byte, 37))
	_, err := k.WebAuthNVerifyAuthenticate(ctx, &types.QueryWebAuthNVerifyAuthenticateRequest{
		Rp:         webAuthnRelyingParty,
		Addr:       webAuthnAccount,
		Challenge:  webAuthnLoginChal,
		Credential: []byte("{}"),
		Data:       []byte(`{"response":{"authenticatorData":"` + authData + `"}}`),
	})
	require.ErrorIs(t, err, types.ErrNoValidWebAuth)
	require.Contains(t, err.Error(), "ID missing")
}
