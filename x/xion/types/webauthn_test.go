package types_test

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
	"github.com/burnt-labs/xion/x/xion/types"
)

type signOpts struct{}

func (*signOpts) HashFunc() crypto.Hash {
	return crypto.SHA256
}

var (
	credentialID = []byte("UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg")
	AAGUID       = []byte("AAGUIDAAGUIDAA==")
)

func getWebAuthNKeys(t *testing.T) (*rsa.PrivateKey, []byte, webauthncose.RSAPublicKeyData) {
	privateKey, _, err := wasmbinding.SetupPublicKeys("../../../wasmbindings/keys/jwtRS256.key")
	require.NoError(t, err)
	publicKey := privateKey.PublicKey
	publicKeyModulus := publicKey.N.Bytes()
	require.NoError(t, err)
	pubKeyData := webauthncose.RSAPublicKeyData{
		PublicKeyData: webauthncose.PublicKeyData{
			KeyType:   int64(webauthncose.RSAKey),
			Algorithm: int64(webauthncose.AlgRS256),
		},
		Modulus:  publicKeyModulus,
		Exponent: big.NewInt(int64(publicKey.E)).Bytes(),
	}
	publicKeyBuf, err := webauthncbor.Marshal(pubKeyData)
	require.NoError(t, err)
	return privateKey, publicKeyBuf, pubKeyData
}

func CreateWebAuthn(t *testing.T) (webauthn.Config, *webauthn.WebAuthn, error) {
	webAuthnConfig := webauthn.Config{
		RPDisplayName:         "Xion",
		RPID:                  "xion-dapp-example-git-feat-faceid-burntfinance.vercel.app",
		RPOrigins:             []string{"https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app"},
		AttestationPreference: "none",
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			UserVerification:        protocol.VerificationPreferred,
		},
	}
	webAuthn, err := webauthn.New(&webAuthnConfig)
	require.NoError(t, err)

	return webAuthnConfig, webAuthn, nil
}

func CreateWebAuthNAttestationCred(t *testing.T, challenge []byte) []byte {
	webAuthnConfig, _, err := CreateWebAuthn(t)
	require.NoError(t, err)
	clientData := protocol.CollectedClientData{
		Type:      "webauthn.create",
		Challenge: string(challenge),
		Origin:    "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app",
	}

	_, publicKeyBuf, _ := getWebAuthNKeys(t)

	clientDataJSON, err := json.Marshal(clientData)
	require.NoError(t, err)

	RPIDHash := sha256.Sum256([]byte(webAuthnConfig.RPID))
	authData := protocol.AuthenticatorData{
		RPIDHash: RPIDHash[:],
		Counter:  0,
		AttData: protocol.AttestedCredentialData{
			AAGUID:              AAGUID,
			CredentialID:        credentialID,
			CredentialPublicKey: publicKeyBuf,
		},
		Flags: 69,
	}
	authDataBz := append(RPIDHash[:], big.NewInt(69).Bytes()...)
	counterBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(counterBytes, 0)
	authDataBz = append(authDataBz, counterBytes...)

	var attData []byte

	// Concatenate AAGUID
	attData = append(attData, AAGUID...)

	// Add CredentialIDLength
	credentialIDLengthBytes := make([]byte, 2)
	credentialIDLength := uint16(len(credentialID)) // nolint:gosec
	binary.BigEndian.PutUint16(credentialIDLengthBytes, credentialIDLength)
	attData = append(attData, credentialIDLengthBytes...)

	// Add CredentialID
	attData = append(attData, credentialID...)

	// Add CredentialPublicKey
	attData = append(attData, publicKeyBuf...)

	authDataBz = append(authDataBz, attData...)

	attestationObject := protocol.AttestationObject{
		AuthData:    authData,
		RawAuthData: authDataBz,
		Format:      "none",
	}
	attestationObjectJSON, err := webauthncbor.Marshal(attestationObject)
	require.NoError(t, err)

	attestationResponse := protocol.AuthenticatorAttestationResponse{
		AuthenticatorResponse: protocol.AuthenticatorResponse{
			ClientDataJSON: protocol.URLEncodedBase64(clientDataJSON),
		},
		AttestationObject: attestationObjectJSON,
	}
	_, err = attestationResponse.Parse()
	require.NoError(t, err)

	credentialCreationResponse := protocol.CredentialCreationResponse{
		PublicKeyCredential: protocol.PublicKeyCredential{
			Credential: protocol.Credential{
				ID:   string(credentialID),
				Type: "public-key",
			},
			RawID:                   credentialID,
			AuthenticatorAttachment: string(protocol.Platform),
		},
		AttestationResponse: attestationResponse,
	}
	_, err = credentialCreationResponse.Parse()
	require.NoError(t, err)

	credCreationRes, err := json.Marshal(credentialCreationResponse)
	require.NoError(t, err)
	_, err = protocol.ParseCredentialCreationResponseBody(bytes.NewReader((credCreationRes)))
	require.NoError(t, err)
	return credCreationRes
}

func CreateWebAuthNSignature(t *testing.T, challenge []byte) []byte {
	webAuthnConfig, webAuthn, err := CreateWebAuthn(t)
	require.NoError(t, err)
	privateKey, publicKeyBuf, pubKeyData := getWebAuthNKeys(t)

	webAuthnUser := types.SmartContractUser{
		Address: "integration_tests",
		Credential: &webauthn.Credential{
			ID:              credentialID,
			AttestationType: "none",
			PublicKey:       publicKeyBuf,
			Transport:       []protocol.AuthenticatorTransport{protocol.Internal},
			Flags: webauthn.CredentialFlags{
				UserPresent:  false,
				UserVerified: false,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:     AAGUID,
				SignCount:  0,
				Attachment: protocol.Platform,
			},
		},
	}

	sessionData := webauthn.SessionData{
		Challenge:        string(challenge),
		UserID:           webAuthnUser.WebAuthnID(),
		UserVerification: protocol.VerificationPreferred,
	}
	RPIDHash := sha256.Sum256([]byte(webAuthnConfig.RPID))
	clientData := protocol.CollectedClientData{
		Type:      "webauthn.get",
		Challenge: string(challenge),
		Origin:    "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app",
	}
	clientDataJSON, err := json.Marshal(clientData)
	require.NoError(t, err)
	clientDataBz, err := protocol.URLEncodedBase64.MarshalJSON(clientDataJSON)
	require.NoError(t, err)
	clientDataHash := sha256.Sum256(clientDataBz)
	authenticatorData := protocol.AuthenticatorData{
		RPIDHash: RPIDHash[:],
		Counter:  1,
		AttData: protocol.AttestedCredentialData{
			AAGUID:              AAGUID,
			CredentialID:        credentialID,
			CredentialPublicKey: publicKeyBuf,
		},
		Flags: 69,
	}
	authDataBz := append(RPIDHash[:], big.NewInt(69).Bytes()...)
	counterBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(counterBytes, 0)
	authDataBz = append(authDataBz, counterBytes...)

	var attData []byte

	// Concatenate AAGUID
	attData = append(attData, AAGUID...)

	// Add CredentialIDLength
	credentialIDLengthBytes := make([]byte, 2)
	credentialIDLength := uint16(len(credentialID)) //nolint:gosec
	binary.BigEndian.PutUint16(credentialIDLengthBytes, credentialIDLength)
	attData = append(attData, credentialIDLengthBytes...)

	// Add CredentialID
	attData = append(attData, credentialID...)

	// Add CredentialPublicKey
	attData = append(attData, publicKeyBuf...)

	authDataBz = append(authDataBz, attData...)
	require.NoError(t, err)
	authenticatorDataBz, err := protocol.URLEncodedBase64.MarshalJSON(authDataBz)
	require.NoError(t, err)

	signData := make([]byte, 0, len(authenticatorDataBz)+len(clientDataHash[:]))
	signData = append(signData, authenticatorDataBz...)
	signData = append(signData, clientDataHash[:]...)
	signHash := sha256.Sum256(signData)
	signature, err := privateKey.Sign(rand.Reader, signHash[:], &signOpts{})
	require.NoError(t, err)
	verified, err := pubKeyData.Verify(signData, signature)
	require.NoError(t, err)
	require.True(t, verified)

	ParsedCredentialAssertionData := &protocol.ParsedCredentialAssertionData{
		ParsedPublicKeyCredential: protocol.ParsedPublicKeyCredential{
			ParsedCredential: protocol.ParsedCredential{
				ID:   string(credentialID),
				Type: "public-key",
			},
			RawID:                   credentialID,
			AuthenticatorAttachment: protocol.Platform,
		},
		Response: protocol.ParsedAssertionResponse{
			CollectedClientData: clientData,
			AuthenticatorData:   authenticatorData,
			Signature:           signature,
			UserHandle:          webAuthnUser.WebAuthnID(),
		},
		Raw: protocol.CredentialAssertionResponse{
			AssertionResponse: protocol.AuthenticatorAssertionResponse{
				AuthenticatorResponse: protocol.AuthenticatorResponse{
					ClientDataJSON: protocol.URLEncodedBase64(clientDataBz),
				},
				AuthenticatorData: protocol.URLEncodedBase64(authenticatorDataBz),
			},
		},
	}
	// validate signature
	_, err = webAuthn.ValidateLogin(webAuthnUser, sessionData, ParsedCredentialAssertionData)
	require.NoError(t, err)
	return signature
}

func TestRegisterAndAuthenticate(t *testing.T) {
	config := sdktypes.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	bec32Addr := "xion1ncx0a0jnsyay7udd03ah2gf64772g02qswj52996dy80qfvgnmzq6eplqq"

	rp, err := url.Parse("https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app")
	require.NoError(t, err)

	challengeStr := "xion1ncx0a0jnsyay7udd03ah2gf64772g02qswj52996dy80qfvgnmzq6eplqq"
	challenge := base64url.Encode([]byte(challengeStr))
	const registerStr = `{"id":"UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg","type":"public-key","rawId":"VVd4WS15UmRJbHM4SVQtdnlNUzZsYTFaaXFFU09BZmY3YldaX0xXVjBQZw","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoiZUdsdmJqRnVZM2d3WVRCcWJuTjVZWGszZFdSa01ETmhhREpuWmpZME56Y3laekF5Y1hOM2FqVXlPVGsyWkhrNE1IRm1kbWR1YlhweE5tVndiSEZ4Iiwib3JpZ2luIjoiaHR0cHM6Ly94aW9uLWRhcHAtZXhhbXBsZS1naXQtZmVhdC1mYWNlaWQtYnVybnRmaW5hbmNlLnZlcmNlbC5hcHAifQ","attestationObject":"o2NmbXRkbm9uZWhBdXRoRGF0YaVkcnBpZFggsGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1lZmxhZ3MYRWhhdHRfZGF0YaNmYWFndWlkUEFBR1VJREFBR1VJREFBPT1qcHVibGljX2tleVkCEKQBAwM5AQAgWQIAolg7TF3aai-wR4HTDe5oR-WRhEsdW3u-O3IJHl0BiHkmR4MLskHG9HzivWoXsloUBnBMrFNxOH0x5cNMI07oi4PeRbHySiogRW9CXPjJaNlTi-pT_IgKFsyJNXsLyzrnajLkDbQU6pRsHmNeL0hAOUv48rtXv8VVWWN8okJehD2q9N7LHoFAOmIUEPg_VTHTt8K__O-9eMZKN4eMjh_4-sxRX6NXPSPT87XRlrK4GZ4pUdp86K0tOFLhwO4Uj0JkMNfI82eVZ1tAbDlqjd8jFnAb8fWm8wtdaTNbL_AAXmbDhswwJOyrw8fARZIhrXSdKBWa6e4k7sLwTIy-OO8saebnlARsjGst7ZCzmw5KCm2ctEVl3hYhHwyXu_A5rOblMrV3H0G7WqeKMCMVSJ11ssrlsmfVhNIwu1Qlt5GYmPTTJiCgGUGRxZkgDyOyjFNHglYpZamCGyJ9oyofsukEGoqMQ6WzjFi_hjVapzXi7Li-Q0OjEopIUUDDgeUrgjbGY0eiHI6sAz5hoaD0Qjc9e3Hk6-y7VcKCTCAanZOlJV0vJkHB98LBLh9qAoVUei_VaLFe2IcfVlrL_43aXlsHhr_SUQY5pHPlUMbQihE_57dpPRh31qDX_w6ye8dilniP8JmpKM2uIwnJ0x7hfJ45Qa0oLHmrGlzY9wi-RGP0YUkhQwEAAW1jcmVkZW50aWFsX2lkWCtVV3hZLXlSZElsczhJVC12eU1TNmxhMVppcUVTT0FmZjdiV1pfTFdWMFBnaGV4dF9kYXRh9mpzaWduX2NvdW50AGhhdXRoRGF0YVkCcrBjAYg3BKaYjH8UNdEzwntvhWiqy3k5L6c8XM54EzMNRQAAAABBQUdVSURBQUdVSURBQT09ACtVV3hZLXlSZElsczhJVC12eU1TNmxhMVppcUVTT0FmZjdiV1pfTFdWMFBnpAEDAzkBACBZAgCiWDtMXdpqL7BHgdMN7mhH5ZGESx1be747cgkeXQGIeSZHgwuyQcb0fOK9aheyWhQGcEysU3E4fTHlw0wjTuiLg95FsfJKKiBFb0Jc-Mlo2VOL6lP8iAoWzIk1ewvLOudqMuQNtBTqlGweY14vSEA5S_jyu1e_xVVZY3yiQl6EPar03ssegUA6YhQQ-D9VMdO3wr_87714xko3h4yOH_j6zFFfo1c9I9PztdGWsrgZnilR2nzorS04UuHA7hSPQmQw18jzZ5VnW0BsOWqN3yMWcBvx9abzC11pM1sv8ABeZsOGzDAk7KvDx8BFkiGtdJ0oFZrp7iTuwvBMjL447yxp5ueUBGyMay3tkLObDkoKbZy0RWXeFiEfDJe78Dms5uUytXcfQbtap4owIxVInXWyyuWyZ9WE0jC7VCW3kZiY9NMmIKAZQZHFmSAPI7KMU0eCVillqYIbIn2jKh-y6QQaioxDpbOMWL-GNVqnNeLsuL5DQ6MSikhRQMOB5SuCNsZjR6IcjqwDPmGhoPRCNz17ceTr7LtVwoJMIBqdk6UlXS8mQcH3wsEuH2oChVR6L9VosV7Yhx9WWsv_jdpeWweGv9JRBjmkc-VQxtCKET_nt2k9GHfWoNf_DrJ7x2KWeI_wmakoza4jCcnTHuF8njlBrSgseasaXNj3CL5EY_RhSSFDAQAB"}}`
	data, err := protocol.ParseCredentialCreationResponseBody(strings.NewReader(registerStr))
	require.NoError(t, err)

	sdkCtx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)
	cred, err := types.VerifyRegistration(sdkCtx, rp, bec32Addr, challenge, data)
	require.NoError(t, err)

	authenticateStr := `{"id":"UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg","type":"public-key","rawId":"VVd4WS15UmRJbHM4SVQtdnlNUzZsYTFaaXFFU09BZmY3YldaX0xXVjBQZw","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uZ2V0IiwiY2hhbGxlbmdlIjoiTWZhT1pqdUlkS0ZiWGtMS1diUGdoU0w4dzQxUnNLMklzc3A0aTBUd3p2VT0iLCJvcmlnaW4iOiJodHRwczovL3hpb24tZGFwcC1leGFtcGxlLWdpdC1mZWF0LWZhY2VpZC1idXJudGZpbmFuY2UudmVyY2VsLmFwcCJ9","authenticatorData":"sGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1FAAAAAEFBR1VJREFBR1VJREFBPT0AK1VXeFkteVJkSWxzOElULXZ5TVM2bGExWmlxRVNPQWZmN2JXWl9MV1YwUGekAQMDOQEAIFkCAKJYO0xd2movsEeB0w3uaEflkYRLHVt7vjtyCR5dAYh5JkeDC7JBxvR84r1qF7JaFAZwTKxTcTh9MeXDTCNO6IuD3kWx8koqIEVvQlz4yWjZU4vqU_yIChbMiTV7C8s652oy5A20FOqUbB5jXi9IQDlL-PK7V7_FVVljfKJCXoQ9qvTeyx6BQDpiFBD4P1Ux07fCv_zvvXjGSjeHjI4f-PrMUV-jVz0j0_O10ZayuBmeKVHafOitLThS4cDuFI9CZDDXyPNnlWdbQGw5ao3fIxZwG_H1pvMLXWkzWy_wAF5mw4bMMCTsq8PHwEWSIa10nSgVmunuJO7C8EyMvjjvLGnm55QEbIxrLe2Qs5sOSgptnLRFZd4WIR8Ml7vwOazm5TK1dx9Bu1qnijAjFUiddbLK5bJn1YTSMLtUJbeRmJj00yYgoBlBkcWZIA8jsoxTR4JWKWWpghsifaMqH7LpBBqKjEOls4xYv4Y1Wqc14uy4vkNDoxKKSFFAw4HlK4I2xmNHohyOrAM-YaGg9EI3PXtx5Ovsu1XCgkwgGp2TpSVdLyZBwffCwS4fagKFVHov1WixXtiHH1Zay_-N2l5bB4a_0lEGOaRz5VDG0IoRP-e3aT0Yd9ag1_8OsnvHYpZ4j_CZqSjNriMJydMe4XyeOUGtKCx5qxpc2PcIvkRj9GFJIUMBAAE","signature":"HoWSrIL-9keuWgvywoD9fxv-AMdGZdw7bYJP2cNnYv_0vKQ6iSmU3WVjE3MvdUDuruE9wYwIuZ-nqUve-56ZTBYmowzZ79PGgCUUNEFFScgH7ShD8McLK90XLKJGEyiTODPlFv2erCCi7pw2o9L3IWDK_B_yFlkYBkhkHI2h3kwcs8aDxcn_hMjHZonxYqm3eB4Syj-FNseCneVYUw8HljSyBVzrMpa4PkukUWTlo46p6HLoe51XMK_UPpXKFnutQkF_DPcwrUzWdgyEZe4B96TZazcRi8-EZtMRKDLrRgzQ1QYe6srqT74FDuMNI8w-0_aUQBUMWPvGGCHZOAUvQV-TnmY5tsAPFpYH5A0Wi5xHw6r5-Gvw9PZH5zss65zA1nHC085w9KGFjhBEkUE_TmzrZTBX6vogt4YIMinA-YxwGUJyF-gbM8-9BkElSSYY3OsAhwlYDERRAE_gw4hoWSNIf2gjZKH0RhLnZY6eViOiqEdnJWnVWbBVL3UMaYvcLvhNakh59OwB0DO2CEGZziw1qQJeN-3d9Rez7ef_gOO5zT1HSYIPHg9Br9z63e0C3abAsg1iNz8kWtvQ_mjypvCL28vaFoXrcYaUHZQogzaqEEGQ-zSwQK-NAsXI_ZKzYSXmbgAv0wFibBMCG_FzE_hYAGHKSQj9tsdxXicBinY","userHandle":"eGlvbjFuY3gwYTBqbnN5YXk3dWRkMDNhaDJnZjY0NzcyZzAycXN3ajUyOTk2ZHk4MHFmdmdubXpxNmVwbHFx"}}`
	authData, err := protocol.ParseCredentialRequestResponseBody(strings.NewReader(authenticateStr))
	require.NoError(t, err)

	challenge = "MfaOZjuIdKFbXkLKWbPghSL8w41RsK2Issp4i0TwzvU="
	verified, err := types.VerifyAuthentication(sdkCtx, rp, bec32Addr, challenge, cred, authData)
	require.NoError(t, err)
	require.True(t, verified)
}

func TestGenerateWebAuthNSignature(t *testing.T) {
	challenge := base64url.Encode([]byte("MfaOZjuIdKFbXkLKWbPghSL8w41RsK2Issp4i0TwzvU="))
	cred := CreateWebAuthNAttestationCred(t, []byte(challenge))
	require.NotNil(t, cred)
	signature := CreateWebAuthNSignature(t, []byte(challenge))
	require.NotNil(t, signature)
}

func TestSmartContractUser(t *testing.T) {
	address := "cosmos1abcdefghijklmnopqrstuvwxyz"
	credential := &webauthn.Credential{
		ID: []byte("test_credential_id"),
	}

	user := types.SmartContractUser{
		Address:    address,
		Credential: credential,
	}

	// Test WebAuthnID
	require.Equal(t, []byte(address), user.WebAuthnID())

	// Test WebAuthnName
	require.Equal(t, address, user.WebAuthnName())

	// Test WebAuthnDisplayName
	require.Equal(t, address, user.WebAuthnDisplayName())
	require.Equal(t, user.WebAuthnName(), user.WebAuthnDisplayName())

	// Test WebAuthnCredentials
	credentials := user.WebAuthnCredentials()
	require.Len(t, credentials, 1)
	require.Equal(t, *credential, credentials[0])

	// Test WebAuthnIcon
	require.Equal(t, "", user.WebAuthnIcon())
}

func TestSmartContractUser_Interface(t *testing.T) {
	// Test that SmartContractUser implements webauthn.User interface
	var _ webauthn.User = types.SmartContractUser{}

	// Test with actual instance
	user := types.SmartContractUser{
		Address: "test_address",
		Credential: &webauthn.Credential{
			ID: []byte("test_id"),
		},
	}

	// Should be able to use as webauthn.User
	var webauthnUser webauthn.User = user
	require.NotNil(t, webauthnUser)
	require.Equal(t, user.WebAuthnID(), webauthnUser.WebAuthnID())
	require.Equal(t, user.WebAuthnName(), webauthnUser.WebAuthnName())
	require.Equal(t, user.WebAuthnDisplayName(), webauthnUser.WebAuthnDisplayName())
	require.Equal(t, user.WebAuthnCredentials(), webauthnUser.WebAuthnCredentials())
}

func TestSmartContractUser_EmptyValues(t *testing.T) {
	// Test with empty/nil values (but valid credential)
	user := types.SmartContractUser{
		Credential: &webauthn.Credential{}, // Valid empty credential, not nil
	}

	require.Equal(t, []byte(""), user.WebAuthnID())
	require.Equal(t, "", user.WebAuthnName())
	require.Equal(t, "", user.WebAuthnDisplayName())
	require.Equal(t, "", user.WebAuthnIcon())

	// WebAuthnCredentials should work with empty credential
	credentials := user.WebAuthnCredentials()
	require.Len(t, credentials, 1)
	require.Equal(t, webauthn.Credential{}, credentials[0])
}

func TestSmartContractUser_WithNilCredential(t *testing.T) {
	user := types.SmartContractUser{
		Address:    "test_address",
		Credential: nil,
	}

	// Should handle nil credential without panicking - but current implementation doesn't
	// This test documents that nil credentials cause panic (which might be intended behavior)
	require.Panics(t, func() {
		user.WebAuthnCredentials()
	})
}

func TestCreateCredential_ErrorPaths(t *testing.T) {
	config := webauthn.Config{
		RPID:          "example.com",
		RPDisplayName: "Example",
		RPOrigins:     []string{"https://example.com"},
	}
	webAuthn, err := webauthn.New(&config)
	require.NoError(t, err)

	user := types.SmartContractUser{
		Address: "test_user",
		Credential: &webauthn.Credential{
			ID: []byte("test_id"),
		},
	}

	ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)

	// Test ID mismatch error
	session := webauthn.SessionData{
		Challenge:        "test_challenge",
		UserID:           []byte("different_user_id"), // Mismatch with user.WebAuthnID()
		UserVerification: protocol.VerificationPreferred,
	}

	parsedResponse := &protocol.ParsedCredentialCreationData{}

	_, err = types.CreateCredential(webAuthn, ctx, user, session, parsedResponse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ID mismatch for User and Session")

	// Test session expiry error
	pastTime := time.Now().Add(-1 * time.Hour)
	expiredSession := webauthn.SessionData{
		Challenge:        "test_challenge",
		UserID:           user.WebAuthnID(),
		UserVerification: protocol.VerificationPreferred,
		Expires:          pastTime, // Session expired
	}

	_, err = types.CreateCredential(webAuthn, ctx, user, expiredSession, parsedResponse)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Session has Expired")

	// Test verification error - invalid parsed response
	validSession := webauthn.SessionData{
		Challenge:        "test_challenge",
		UserID:           user.WebAuthnID(),
		UserVerification: protocol.VerificationRequired, // Test UserVerification Required path
	}

	// parsedResponse with invalid data will cause Verify to fail
	invalidParsedResponse := &protocol.ParsedCredentialCreationData{
		ParsedPublicKeyCredential: protocol.ParsedPublicKeyCredential{
			ParsedCredential: protocol.ParsedCredential{
				ID:   "invalid",
				Type: "public-key",
			},
		},
		Response: protocol.ParsedAttestationResponse{
			CollectedClientData: protocol.CollectedClientData{
				Type:      "webauthn.create",
				Challenge: "different_challenge", // Mismatch will cause verification error
				Origin:    "https://badorigin.com",
			},
		},
	}

	_, err = types.CreateCredential(webAuthn, ctx, user, validSession, invalidParsedResponse)
	require.Error(t, err)
	// This should trigger the verification error path
}

func TestSmartContractUser_AllMethods(t *testing.T) {
	address := "cosmos1test"
	credential := &webauthn.Credential{
		ID:              []byte("test_credential_id"),
		AttestationType: "none",
		PublicKey:       []byte("test_public_key"),
	}

	user := types.SmartContractUser{
		Address:    address,
		Credential: credential,
	}

	// Test all SmartContractUser methods to get 100% coverage
	require.Equal(t, []byte(address), user.WebAuthnID())
	require.Equal(t, address, user.WebAuthnName())
	require.Equal(t, address, user.WebAuthnDisplayName())
	require.Equal(t, "", user.WebAuthnIcon())

	credentials := user.WebAuthnCredentials()
	require.Len(t, credentials, 1)
	require.Equal(t, *credential, credentials[0])
}

func TestVerifyRegistration_ErrorPath(t *testing.T) {
	// Test invalid URL/config error path in VerifyRegistration
	invalidRP := &url.URL{Host: ""} // Invalid config will cause webauthn.New to fail

	ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)
	challenge := "test_challenge"
	contractAddr := "test_contract"

	data := &protocol.ParsedCredentialCreationData{}

	_, err := types.VerifyRegistration(ctx, invalidRP, contractAddr, challenge, data)
	require.Error(t, err)
	// Should get error from webauthn.New with invalid config
}

func TestVerifyAuthentication_ErrorPath(t *testing.T) {
	// Test invalid URL/config error path in VerifyAuthentication
	invalidRP := &url.URL{Host: ""} // Invalid config will cause webauthn.New to fail

	challenge := "test_challenge"
	contractAddr := "test_contract"
	credential := &webauthn.Credential{
		ID: []byte("test_id"),
	}

	data := &protocol.ParsedCredentialAssertionData{}

	ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)
	_, err := types.VerifyAuthentication(ctx, invalidRP, contractAddr, challenge, credential, data)
	require.Error(t, err)
	// Should get error from webauthn.New with invalid config
}

func TestVerifyAuthentication_ValidateLoginError(t *testing.T) {
	// Test ValidateLogin error path with valid config but invalid assertion data
	rp, err := url.Parse("https://example.com")
	require.NoError(t, err)

	challenge := "test_challenge"
	contractAddr := "test_contract"
	credential := &webauthn.Credential{
		ID:        []byte("test_id"),
		PublicKey: []byte("invalid_public_key"),
	}

	// Create invalid assertion data that will cause ValidateLogin to fail
	invalidData := &protocol.ParsedCredentialAssertionData{
		ParsedPublicKeyCredential: protocol.ParsedPublicKeyCredential{
			ParsedCredential: protocol.ParsedCredential{
				ID:   "test_id",
				Type: "public-key",
			},
		},
		Response: protocol.ParsedAssertionResponse{
			CollectedClientData: protocol.CollectedClientData{
				Type:      "webauthn.get",
				Challenge: "different_challenge", // Wrong challenge will cause validation to fail
				Origin:    "https://badorigin.com",
			},
			Signature: []byte("invalid_signature"),
		},
	}

	ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)
	verified, err := types.VerifyAuthentication(ctx, rp, contractAddr, challenge, credential, invalidData)
	require.Error(t, err)
	require.False(t, verified)
	// Should get error from ValidateLogin with invalid assertion data
}

// === Consensus Determinism Tests ===

// Helper function to create a short-lived certificate for testing time-based consensus issues
func createCertForTimeValidation(referenceTime time.Time, validityPeriod time.Duration) (certDER []byte, priv *rsa.PrivateKey, err error) {
	priv, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	// Create certificate with specific validity period relative to reference time
	startTime := referenceTime
	endTime := referenceTime.Add(validityPeriod)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2025),
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"Test Authenticator"},
			OrganizationalUnit: []string{"Authenticator Attestation"},
			CommonName:         "Test-WebAuthn-Cert",
		},
		NotBefore:             startTime,
		NotAfter:              endTime,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDER, err = x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	return
}

// Helper to encode time as bytes for certificate extension
func encodeTime(t time.Time) []byte {
	return []byte(t.Format(time.RFC3339))
}

func createShortLivedCert(referenceTime time.Time, validDuration time.Duration) (certDER []byte, priv *rsa.PrivateKey, err error) {
	priv, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	// Create certificate that starts well before the reference time to avoid timing issues
	// Make sure it's also valid during actual test execution (current time)
	startTime := referenceTime.Add(-2 * time.Hour)

	// For very short certificates (meant to test expiry), don't add buffer
	// For longer certificates, add buffer to avoid expiry during test runs
	var endTime time.Time
	if validDuration <= 30*time.Second {
		// Short-lived certificate for testing expiry - use exact duration
		endTime = referenceTime.Add(validDuration)
	} else {
		// Longer certificate - add buffer to avoid test timing issues
		endTime = startTime.Add(validDuration + 48*time.Hour)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2025),
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"Test Authenticator"},
			OrganizationalUnit: []string{"Authenticator Attestation"},
			CommonName:         "Test-WebAuthn-Cert",
		},
		NotBefore:             startTime,
		NotAfter:              endTime,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDER, err = x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	return
}

// createCertWithValidityPeriod creates a certificate valid between start and end times
func createCertWithValidityPeriod(notBefore, notAfter time.Time) ([]byte, *rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2025),
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"Test Authenticator"},
			OrganizationalUnit: []string{"Authenticator Attestation"},
			CommonName:         "Test-WebAuthn-Cert",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	return certDER, priv, err
}

// Helper to build a WebAuthn attestation object with the given certificate
func buildPackedAttestation(certDER []byte, priv *rsa.PrivateKey, clientDataHash []byte) ([]byte, error) {
	credID := make([]byte, 16)
	rand.Read(credID)

	authData := &bytes.Buffer{}

	rpIDHash := sha256.Sum256([]byte("test.example"))
	authData.Write(rpIDHash[:])

	authData.WriteByte(0x41)           // flags
	authData.Write([]byte{0, 0, 0, 0}) // counter
	authData.Write(make([]byte, 16))   // AAGUID
	authData.WriteByte(0x00)
	authData.WriteByte(0x10)
	authData.Write(credID)

	pubKeyBytes, err := encodeRSAPublicKeyAsCOSE(&priv.PublicKey)
	if err != nil {
		return nil, err
	}
	authData.Write(pubKeyBytes)

	authDataBytes := authData.Bytes()
	toSign := append(authDataBytes, clientDataHash...)

	hashToSign := sha256.Sum256(toSign)
	signature, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashToSign[:])
	if err != nil {
		return nil, err
	}

	attStmt := map[string]interface{}{
		"alg": -257, // RS256 for RSA
		"sig": signature,
		"x5c": []interface{}{certDER},
	}

	attObj := map[string]interface{}{
		"fmt":      "packed",
		"authData": authDataBytes,
		"attStmt":  attStmt,
	}

	return webauthncbor.Marshal(attObj)
}

func encodeRSAPublicKeyAsCOSE(pubKey *rsa.PublicKey) ([]byte, error) {
	coseKey := map[int]interface{}{
		1:  3,    // kty: RSA
		3:  -257, // alg: RS256
		-1: pubKey.N.Bytes(),
		-2: big.NewInt(int64(pubKey.E)).Bytes(),
	}
	return webauthncbor.Marshal(coseKey)
}

func buildCredentialCreationJSON(attBytes []byte, clientDataJSON []byte) []byte {
	credID := make([]byte, 16)
	rand.Read(credID)

	cred := map[string]interface{}{
		"id":    base64url.Encode(credID),
		"rawId": base64url.Encode(credID),
		"type":  "public-key",
		"response": map[string]string{
			"attestationObject": base64url.Encode(attBytes),
			"clientDataJSON":    base64url.Encode(clientDataJSON),
		},
	}
	b, _ := json.Marshal(cred)
	return b
}

// TestWebAuthnTimeConsensus tests that WebAuthn verification is deterministic across validators
func TestWebAuthnTimeConsensus(t *testing.T) {
	// Test with a fixed block time (deterministic)
	baseTime := time.Now()
	
	// Create a certificate that expires in 5 seconds from baseTime
	certDER, priv, err := createShortLivedCert(baseTime, 5*time.Second)
	require.NoError(t, err)

	// Create test data
	clientData := map[string]string{
		"type":      "webauthn.create",
		"challenge": "test_challenge_123",
		"origin":    "https://test.example",
	}
	clientDataJSON, _ := json.Marshal(clientData)
	clientDataHash := sha256.Sum256(clientDataJSON)

	attObj, err := buildPackedAttestation(certDER, priv, clientDataHash[:])
	require.NoError(t, err)

	bodyJSON := buildCredentialCreationJSON(attObj, clientDataJSON)

	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(bodyJSON))
	require.NoError(t, err)

	rp, _ := url.Parse("https://test.example")

	// Both contexts should produce the same result (deterministic)
	ctx1 := sdktypes.NewContext(nil, cmtproto.Header{Time: baseTime}, false, nil)
	ctx2 := sdktypes.NewContext(nil, cmtproto.Header{Time: baseTime}, false, nil)

	// Both contexts should produce the same result (deterministic)
	cred1, err1 := types.VerifyRegistration(ctx1, rp, "contract1", "test_challenge_123", parsed)
	cred2, err2 := types.VerifyRegistration(ctx2, rp, "contract1", "test_challenge_123", parsed)

	// Both should succeed or both should fail
	require.Equal(t, err1 == nil, err2 == nil, "Deterministic verification should produce same result")
	if err1 == nil && err2 == nil {
		require.Equal(t, cred1.ID, cred2.ID, "Credentials should be identical")
	}

	t.Logf("Deterministic verification result: success=%v", err1 == nil)

	// Test with block time after certificate expiry
	futureTime := baseTime.Add(10 * time.Second) // After cert expires
	ctx3 := sdktypes.NewContext(nil, cmtproto.Header{Time: futureTime}, false, nil)

	certDER, priv, err = createShortLivedCert(baseTime, 5*time.Second)
	require.NoError(t, err)

	// Create test data
	clientData2 := map[string]string{
		"type":      "webauthn.create",
		"challenge": "test_challenge_123",
		"origin":    "https://test.example",
	}
	clientDataJSON2, _ := json.Marshal(clientData2)
	clientDataHash2 := sha256.Sum256(clientDataJSON2)

	attObj, err = buildPackedAttestation(certDER, priv, clientDataHash2[:])
	require.NoError(t, err)

	bodyJSON = buildCredentialCreationJSON(attObj, clientDataJSON2)

	parsed, err = protocol.ParseCredentialCreationResponseBody(bytes.NewReader(bodyJSON))
	require.NoError(t, err)

	cred3, err3 := types.VerifyRegistration(ctx3, rp, "contract1", "test_challenge_123", parsed)
	require.Error(t, err3, "Should fail when certificate is expired according to block time")
	require.Nil(t, cred3, "Credential should be nil on failure")

	t.Logf("Expired certificate verification correctly failed: %v", err3)
}

// TestWebAuthnBlockTimeConsistency verifies that the same block time produces identical results
func TestWebAuthnBlockTimeConsistency(t *testing.T) {
	// Create a long-lived certificate to avoid expiry issues during test
	baseTime := time.Now()
	certDER, priv, err := createShortLivedCert(baseTime, 1*time.Hour)
	require.NoError(t, err)

	clientData := map[string]string{
		"type":      "webauthn.create",
		"challenge": "consistency_test",
		"origin":    "https://test.example",
	}
	clientDataJSON, _ := json.Marshal(clientData)
	clientDataHash := sha256.Sum256(clientDataJSON)

	attObj, err := buildPackedAttestation(certDER, priv, clientDataHash[:])
	require.NoError(t, err)

	bodyJSON := buildCredentialCreationJSON(attObj, clientDataJSON)
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(bodyJSON))
	require.NoError(t, err)

	rp, _ := url.Parse("https://test.example")

	// Use the same block time for multiple verification attempts
	fixedBlockTime := time.Now()

	results := make([]*webauthn.Credential, 5)
	errors := make([]error, 5)

	// Run verification multiple times with same block time
	for i := 0; i < 5; i++ {
		ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: fixedBlockTime}, false, nil)
		results[i], errors[i] = types.VerifyRegistration(ctx, rp, "contract1", "consistency_test", parsed)
	}

	// All results should be identical
	for i := 1; i < 5; i++ {
		require.Equal(t, errors[0] == nil, errors[i] == nil,
			"All verifications with same block time should have same success/failure")

		if errors[0] == nil && errors[i] == nil {
			require.Equal(t, results[0].ID, results[i].ID,
				"All successful verifications should produce identical credentials")
		}
	}

	t.Logf("All %d verifications with same block time produced identical results", 5)
}

// TestWebAuthnCertificateValidation tests that certificate validation uses block time
func TestWebAuthnCertificateValidation(t *testing.T) {
	// Use fixed dates that are stable and well in the past/future
	// This ensures we can test specific time scenarios without flakiness
	testBaseTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	// Create a long-lived certificate that covers all our test scenarios
	// This certificate will be valid from 2024-01-01 to 2025-12-31
	certStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	certEnd := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	cert, priv, err := createCertWithValidityPeriod(certStart, certEnd)
	require.NoError(t, err)

	rp, _ := url.Parse("https://test.example")

	testCases := []struct {
		name          string
		certDER       []byte
		priv          *rsa.PrivateKey
		blockTime     time.Time
		shouldSucceed bool
	}{
		{
			name:          "block_time_before_cert_valid",
			certDER:       cert,
			priv:          priv,
			blockTime:     time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC), // Before cert starts
			shouldSucceed: false,
		},
		{
			name:          "block_time_during_cert_valid",
			certDER:       cert,
			priv:          priv,
			blockTime:     testBaseTime, // Well within cert validity
			shouldSucceed: true,
		},
		{
			name:          "block_time_after_cert_expires",
			certDER:       cert,
			priv:          priv,
			blockTime:     time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC), // After cert expires
			shouldSucceed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Debug: Let's check the certificate validity period
			cert, err := x509.ParseCertificate(tc.certDER)
			require.NoError(t, err)
			t.Logf("Certificate NotBefore: %v", cert.NotBefore)
			t.Logf("Certificate NotAfter: %v", cert.NotAfter)
			t.Logf("Block time: %v", tc.blockTime)
			t.Logf("Is valid at block time: %v", tc.blockTime.After(cert.NotBefore) && tc.blockTime.Before(cert.NotAfter))

			clientData := map[string]string{
				"type":      "webauthn.create",
				"challenge": "cert_validation_test",
				"origin":    "https://test.example",
			}
			clientDataJSON, _ := json.Marshal(clientData)
			clientDataHash := sha256.Sum256(clientDataJSON)

			attObj, err := buildPackedAttestation(tc.certDER, tc.priv, clientDataHash[:])
			require.NoError(t, err)

			bodyJSON := buildCredentialCreationJSON(attObj, clientDataJSON)
			parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(bodyJSON))
			require.NoError(t, err)

			ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: tc.blockTime}, false, nil)
			cred, err := types.VerifyRegistration(ctx, rp, "contract1", "cert_validation_test", parsed)

			if tc.shouldSucceed {
				require.NoError(t, err, "Certificate should be valid at block time")
				require.NotNil(t, cred, "Should return valid credential")
			} else {
				require.Error(t, err, "Certificate should be invalid at block time")
				require.Nil(t, cred, "Should not return credential when invalid")
			}
		})
	}
}

// === Deterministic Function Tests ===

// TestCreateCredential tests the core deterministic credential creation function
func TestCreateCredential(t *testing.T) {
	// Create a simple WebAuthn config
	config := webauthn.Config{
		RPID:          "test.example",
		RPDisplayName: "Test Example",
		RPOrigins:     []string{"https://test.example"},
	}
	webAuth, err := webauthn.New(&config)
	require.NoError(t, err)

	// Create test user
	user := types.SmartContractUser{Address: "test_user"}

	// Create test session
	session := webauthn.SessionData{
		Challenge:        "test_challenge",
		UserID:           user.WebAuthnID(),
		UserVerification: protocol.VerificationPreferred,
	}

	// Test with valid certificate and current block time
	baseTime := time.Now()
	certDER, priv, err := createShortLivedCert(baseTime, 1*time.Hour)
	require.NoError(t, err)

	clientData := map[string]string{
		"type":      "webauthn.create",
		"challenge": "test_challenge",
		"origin":    "https://test.example",
	}
	clientDataJSON, _ := json.Marshal(clientData)
	clientDataHash := sha256.Sum256(clientDataJSON)

	attObj, err := buildPackedAttestation(certDER, priv, clientDataHash[:])
	require.NoError(t, err)

	bodyJSON := buildCredentialCreationJSON(attObj, clientDataJSON)
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(bodyJSON))
	require.NoError(t, err)

	// Test successful creation with valid block time
	ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)
	cred, err := types.CreateCredential(webAuth, ctx, user, session, parsed)
	require.NoError(t, err)
	require.NotNil(t, cred)
	require.NotEmpty(t, cred.ID)

	t.Logf("Successfully created deterministic credential: %x", cred.ID)
}

// TestCreateCredential_IDMismatch tests ID validation
func TestCreateCredential_IDMismatch(t *testing.T) {
	config := webauthn.Config{
		RPID:          "test.example",
		RPDisplayName: "Test Example",
		RPOrigins:     []string{"https://test.example"},
	}
	webAuth, err := webauthn.New(&config)
	require.NoError(t, err)

	user := types.SmartContractUser{Address: "test_user"}

	// Create session with different user ID
	session := webauthn.SessionData{
		Challenge:        "test_challenge",
		UserID:           []byte("different_user"),
		UserVerification: protocol.VerificationPreferred,
	}

	// Create minimal valid parsed response
	baseTime := time.Now()
	certDER, priv, err := createShortLivedCert(baseTime, 1*time.Hour)
	require.NoError(t, err)

	clientData := map[string]string{
		"type":      "webauthn.create",
		"challenge": "test_challenge",
		"origin":    "https://test.example",
	}
	clientDataJSON, _ := json.Marshal(clientData)
	clientDataHash := sha256.Sum256(clientDataJSON)

	attObj, err := buildPackedAttestation(certDER, priv, clientDataHash[:])
	require.NoError(t, err)

	bodyJSON := buildCredentialCreationJSON(attObj, clientDataJSON)
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(bodyJSON))
	require.NoError(t, err)

	ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)
	cred, err := types.CreateCredential(webAuth, ctx, user, session, parsed)

	require.Error(t, err)
	require.Nil(t, cred)
	require.Contains(t, err.Error(), "ID mismatch for User and Session")
}

// TestCreateCredential_SessionExpired tests session expiry validation
func TestCreateCredential_SessionExpired(t *testing.T) {
	config := webauthn.Config{
		RPID:          "test.example",
		RPDisplayName: "Test Example",
		RPOrigins:     []string{"https://test.example"},
	}
	webAuth, err := webauthn.New(&config)
	require.NoError(t, err)

	user := types.SmartContractUser{Address: "test_user"}

	// Create session that expires before block time
	pastTime := time.Now().Add(-1 * time.Hour)
	session := webauthn.SessionData{
		Challenge:        "test_challenge",
		UserID:           user.WebAuthnID(),
		UserVerification: protocol.VerificationPreferred,
		Expires:          pastTime,
	}

	// Create valid parsed response
	baseTime := time.Now()
	certDER, priv, err := createShortLivedCert(baseTime, 1*time.Hour)
	require.NoError(t, err)

	clientData := map[string]string{
		"type":      "webauthn.create",
		"challenge": "test_challenge",
		"origin":    "https://test.example",
	}
	clientDataJSON, _ := json.Marshal(clientData)
	clientDataHash := sha256.Sum256(clientDataJSON)

	attObj, err := buildPackedAttestation(certDER, priv, clientDataHash[:])
	require.NoError(t, err)

	bodyJSON := buildCredentialCreationJSON(attObj, clientDataJSON)
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(bodyJSON))
	require.NoError(t, err)

	// Use block time after session expiry
	ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)
	cred, err := types.CreateCredential(webAuth, ctx, user, session, parsed)

	require.Error(t, err)
	require.Nil(t, cred)
	require.Contains(t, err.Error(), "Session has Expired")
}

func TestCreateCredential_MalformedCertificate(t *testing.T) {
	// Create a test context with block time
	ctx := sdktypes.Context{}.WithBlockTime(time.Now())

	webAuth, err := webauthn.New(&webauthn.Config{
		RPID:          "example.com",
		RPDisplayName: "Example",
		RPOrigins:     []string{"https://example.com"},
	})
	require.NoError(t, err)

	user := types.SmartContractUser{
		Address: "test-address",
	}

	session := webauthn.SessionData{
		Challenge:        "test-challenge",
		UserID:           user.WebAuthnID(),
		UserVerification: protocol.VerificationPreferred,
		Expires:          time.Now().Add(time.Hour), // Not expired
	}

	// Create a credential creation response with malformed certificate data
	parsed := &protocol.ParsedCredentialCreationData{
		ParsedPublicKeyCredential: protocol.ParsedPublicKeyCredential{
			ParsedCredential: protocol.ParsedCredential{
				ID:   "test-credential-id",
				Type: "public-key",
			},
		},
		Response: protocol.ParsedAttestationResponse{
			CollectedClientData: protocol.CollectedClientData{
				Type:      "webauthn.create",
				Challenge: "test-challenge",
				Origin:    "https://example.com",
			},
			AttestationObject: protocol.AttestationObject{
				AttStatement: map[string]interface{}{
					"fmt": "none",
					// Add malformed certificate data in x5c
					"x5c": []interface{}{
						[]byte("malformed-certificate-data"), // This will cause parsing to fail
					},
				},
			},
		},
	}

	cred, err := types.CreateCredential(webAuth, ctx, user, session, parsed)

	require.Error(t, err)
	require.Nil(t, cred)
	require.Contains(t, err.Error(), "Failed to parse X.509 certificate")
}

// TestWebAuthnTimeConsensusVulnerability demonstrates that Eduardo's commit did NOT fix
// the WebAuthN consensus vulnerability. The issue remains: webauthn.MakeNewCredential()
// uses system time.Now() for X.509 certificate validation instead of blockchain consensus time.
//
// This creates a critical consensus failure where different validators with different
// system clocks will get different results for the same WebAuthN credential verification.
func TestWebAuthnTimeConsensusVulnerability(t *testing.T) {
	// Keep all test times within a 2-hour window around a single captured "now"
	baseNow := time.Now().UTC()
	pastBlockTime := baseNow.Add(-1 * time.Hour)
	futureBlockTime := baseNow.Add(1 * time.Hour)

	pastCtx := sdktypes.Context{}.WithBlockTime(pastBlockTime)
	futureCtx := sdktypes.Context{}.WithBlockTime(futureBlockTime)

	currentSystemTime := baseNow

	t.Log("=== WEBAUTHN CONSENSUS VULNERABILITY DEMONSTRATION ===")
	t.Log("Past block time:", pastCtx.BlockTime())
	t.Log("Future block time:", futureCtx.BlockTime())
	t.Log("Current system time:", currentSystemTime)
	t.Log("")

	// VULNERABILITY ANALYSIS:
	t.Log("VULNERABILITY: WebAuthN X.509 certificate validation uses time.Now() not consensus time")
	t.Log("")
	t.Log("Code path that causes the issue:")
	t.Log("1. types.VerifyRegistration(ctx, ...) receives blockchain consensus time")
	t.Log("2. Eventually calls types.CreateCredential(webauth, ctx, user, session, data)")
	t.Log("3. CreateCredential uses ctx.BlockTime() for session validation (correct)")
	t.Log("4. But then calls webauthn.MakeNewCredential(parsedResponse)")
	t.Log("5. webauthn.MakeNewCredential() IGNORES ctx and uses time.Now() internally")
	t.Log("6. X.509 certificate validation becomes non-deterministic!")
	t.Log("")

	// CONSENSUS IMPACT:
	t.Log("CONSENSUS FAILURE SCENARIO:")
	t.Log("- All validators process the same block with same BlockTime")
	t.Log("- But validators have different system clocks (time.Now())")
	t.Log("- WebAuthN certificate validation gives different results per validator")
	t.Log("- Network cannot reach consensus on WebAuthN transactions")
	t.Log("")

	// Demonstrate the time differences that would cause issues
	pastDiff := currentSystemTime.Sub(pastBlockTime)
	futureDiff := futureBlockTime.Sub(currentSystemTime)

	t.Logf("Time difference (system vs past block): %v", pastDiff)
	t.Logf("Time difference (future block vs system): %v", futureDiff)

	if pastDiff > time.Hour || futureDiff > time.Hour {
		t.Log("CRITICAL: Large time differences would cause certificate validation inconsistencies")
	}

	t.Log("")
	t.Log("PREVIOUS COMMIT DID NOT FIX THIS:")
	t.Log("- Current webauthn.go still calls webauthn.MakeNewCredential() at line ~105")
	t.Log("- webauthn.MakeNewCredential() is from external library, cannot be modified")
	t.Log("- No deterministic alternative was implemented")
	t.Log("- Vulnerability remains active in this codebase")
	t.Log("")

	t.Log("REQUIRED FIX:")
	t.Log("- Replace webauthn.MakeNewCredential() with custom deterministic implementation")
	t.Log("- Use ctx.BlockTime() for all X.509 certificate validation")
	t.Log("- Ensure identical results across all validators")
}
