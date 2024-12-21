package types_test

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"net/url"
	"strings"
	"testing"

	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"

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

	signData := append(authenticatorDataBz, clientDataHash[:]...)
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

	cred, err := types.VerifyRegistration(rp, bec32Addr, challenge, data)
	require.NoError(t, err)

	authenticateStr := `{"id":"UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg","type":"public-key","rawId":"VVd4WS15UmRJbHM4SVQtdnlNUzZsYTFaaXFFU09BZmY3YldaX0xXVjBQZw","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uZ2V0IiwiY2hhbGxlbmdlIjoiTWZhT1pqdUlkS0ZiWGtMS1diUGdoU0w4dzQxUnNLMklzc3A0aTBUd3p2VT0iLCJvcmlnaW4iOiJodHRwczovL3hpb24tZGFwcC1leGFtcGxlLWdpdC1mZWF0LWZhY2VpZC1idXJudGZpbmFuY2UudmVyY2VsLmFwcCJ9","authenticatorData":"sGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1FAAAAAEFBR1VJREFBR1VJREFBPT0AK1VXeFkteVJkSWxzOElULXZ5TVM2bGExWmlxRVNPQWZmN2JXWl9MV1YwUGekAQMDOQEAIFkCAKJYO0xd2movsEeB0w3uaEflkYRLHVt7vjtyCR5dAYh5JkeDC7JBxvR84r1qF7JaFAZwTKxTcTh9MeXDTCNO6IuD3kWx8koqIEVvQlz4yWjZU4vqU_yIChbMiTV7C8s652oy5A20FOqUbB5jXi9IQDlL-PK7V7_FVVljfKJCXoQ9qvTeyx6BQDpiFBD4P1Ux07fCv_zvvXjGSjeHjI4f-PrMUV-jVz0j0_O10ZayuBmeKVHafOitLThS4cDuFI9CZDDXyPNnlWdbQGw5ao3fIxZwG_H1pvMLXWkzWy_wAF5mw4bMMCTsq8PHwEWSIa10nSgVmunuJO7C8EyMvjjvLGnm55QEbIxrLe2Qs5sOSgptnLRFZd4WIR8Ml7vwOazm5TK1dx9Bu1qnijAjFUiddbLK5bJn1YTSMLtUJbeRmJj00yYgoBlBkcWZIA8jsoxTR4JWKWWpghsifaMqH7LpBBqKjEOls4xYv4Y1Wqc14uy4vkNDoxKKSFFAw4HlK4I2xmNHohyOrAM-YaGg9EI3PXtx5Ovsu1XCgkwgGp2TpSVdLyZBwffCwS4fagKFVHov1WixXtiHH1Zay_-N2l5bB4a_0lEGOaRz5VDG0IoRP-e3aT0Yd9ag1_8OsnvHYpZ4j_CZqSjNriMJydMe4XyeOUGtKCx5qxpc2PcIvkRj9GFJIUMBAAE","signature":"HoWSrIL-9keuWgvywoD9fxv-AMdGZdw7bYJP2cNnYv_0vKQ6iSmU3WVjE3MvdUDuruE9wYwIuZ-nqUve-56ZTBYmowzZ79PGgCUUNEFFScgH7ShD8McLK90XLKJGEyiTODPlFv2erCCi7pw2o9L3IWDK_B_yFlkYBkhkHI2h3kwcs8aDxcn_hMjHZonxYqm3eB4Syj-FNseCneVYUw8HljSyBVzrMpa4PkukUWTlo46p6HLoe51XMK_UPpXKFnutQkF_DPcwrUzWdgyEZe4B96TZazcRi8-EZtMRKDLrRgzQ1QYe6srqT74FDuMNI8w-0_aUQBUMWPvGGCHZOAUvQV-TnmY5tsAPFpYH5A0Wi5xHw6r5-Gvw9PZH5zss65zA1nHC085w9KGFjhBEkUE_TmzrZTBX6vogt4YIMinA-YxwGUJyF-gbM8-9BkElSSYY3OsAhwlYDERRAE_gw4hoWSNIf2gjZKH0RhLnZY6eViOiqEdnJWnVWbBVL3UMaYvcLvhNakh59OwB0DO2CEGZziw1qQJeN-3d9Rez7ef_gOO5zT1HSYIPHg9Br9z63e0C3abAsg1iNz8kWtvQ_mjypvCL28vaFoXrcYaUHZQogzaqEEGQ-zSwQK-NAsXI_ZKzYSXmbgAv0wFibBMCG_FzE_hYAGHKSQj9tsdxXicBinY","userHandle":"eGlvbjFuY3gwYTBqbnN5YXk3dWRkMDNhaDJnZjY0NzcyZzAycXN3ajUyOTk2ZHk4MHFmdmdubXpxNmVwbHFx"}}`
	authData, err := protocol.ParseCredentialRequestResponseBody(strings.NewReader(authenticateStr))
	require.NoError(t, err)

	challenge = "MfaOZjuIdKFbXkLKWbPghSL8w41RsK2Issp4i0TwzvU="
	verified, err := types.VerifyAuthentication(rp, bec32Addr, challenge, cred, authData)
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
