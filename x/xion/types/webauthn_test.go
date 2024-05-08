package types_test

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/dvsekhvalnov/jose2go/base64url"

	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
	"github.com/burnt-labs/xion/x/xion/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
)

type signOpts struct{}

func (*signOpts) HashFunc() crypto.Hash {
	return crypto.SHA256
}

func CreateWebAuthNSignature(t *testing.T, challenge []byte) []byte {
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

	privateKey, publicKey, err := wasmbinding.SetupPublicKeys("../../../wasmbindings/keys/jwtRS256.key")
	require.NoError(t, err)
	publicKeyJson, err := json.Marshal(publicKey)
	require.NoError(t, err)

	credentialID := []byte("UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg")
	AAGUID := []byte("rc4AAjW8xgpkiwsl8fBVAw==")

	webAuthnUser := types.SmartContractUser{
		Address: "integration_tests",
		Credential: &webauthn.Credential{
			ID:              credentialID,
			AttestationType: "none",
			PublicKey:       publicKeyJson,
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
			CredentialPublicKey: publicKeyJson,
		},
		Flags: 69,
	}
	authenticatorDataJSON, err := json.Marshal(authenticatorData)
	require.NoError(t, err)
	authenticatorDataBz, err := protocol.URLEncodedBase64.MarshalJSON(authenticatorDataJSON)
	require.NoError(t, err)
	authenticatorHash := sha256.Sum256(authenticatorDataBz)

	signHash := sha256.Sum256(append(authenticatorHash[:], clientDataHash[:]...))
	signature, err := privateKey.Sign(rand.Reader, signHash[:], &signOpts{})
	require.NoError(t, err)

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

	bec32Addr := "xion1cyyld62ly828e2xnp0c0ckpyz68wwfs26tjpscmqlaum2jcj8zdstlxvya"

	rp, err := url.Parse("https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app")
	require.NoError(t, err)

	challengeStr := "test-challenge"
	challenge := base64url.Encode([]byte(challengeStr))
	const registerStr = `{"type":"public-key","id":"6BnpSHlIXwOndHhxfPw4l3SylupnZIvTVP9Vp_aK34w","rawId":"6BnpSHlIXwOndHhxfPw4l3SylupnZIvTVP9Vp_aK34w","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoiZEdWemRDMWphR0ZzYkdWdVoyVSIsIm9yaWdpbiI6Imh0dHBzOi8veGlvbi1kYXBwLWV4YW1wbGUtZ2l0LWZlYXQtZmFjZWlkLWJ1cm50ZmluYW5jZS52ZXJjZWwuYXBwIiwiY3Jvc3NPcmlnaW4iOmZhbHNlfQ","attestationObject":"o2NmbXRkbm9uZWdhdHRTdG10oGhhdXRoRGF0YViksGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1BAAAAAK3OAAI1vMYKZIsLJfHwVQMAIOgZ6Uh5SF8Dp3R4cXz8OJd0spbqZ2SL01T_Vaf2it-MpQECAyYgASFYINnBKEMfG6wkb9W1grSXgNAQ8lx6H7j6EcMyTSbZ91-XIlggdk2OOxV_bISxCsqFac6ZE8-gEurV4xQd7kFFYdfMqtE","transports":["internal"]},"clientExtensionResults":{}}`

	data, err := protocol.ParseCredentialCreationResponseBody(strings.NewReader(registerStr))
	require.NoError(t, err)

	cred, err := types.VerifyRegistration(rp, bec32Addr, challenge, data)
	require.NoError(t, err)

	t.Logf("credential: %v", cred)

	authenticateStr := `{"type":"public-key","id":"6BnpSHlIXwOndHhxfPw4l3SylupnZIvTVP9Vp_aK34w","rawId":"6BnpSHlIXwOndHhxfPw4l3SylupnZIvTVP9Vp_aK34w","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uZ2V0IiwiY2hhbGxlbmdlIjoiZEdWemRDMWphR0ZzYkdWdVoyVSIsIm9yaWdpbiI6Imh0dHBzOi8veGlvbi1kYXBwLWV4YW1wbGUtZ2l0LWZlYXQtZmFjZWlkLWJ1cm50ZmluYW5jZS52ZXJjZWwuYXBwIiwiY3Jvc3NPcmlnaW4iOmZhbHNlfQ","authenticatorData":"sGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw0BAAAAAA","signature":"MEQCIF1Fm_XjFV5FjBRYXNN1WcDm0V4xbPn3sQ85gC34_FGmAiBzLYGsat3HwDcn4jh50gTW4mgGcmYqkvT2g1bfdFxElA","userHandle":null},"clientExtensionResults":{}}`

	authData, err := protocol.ParseCredentialRequestResponseBody(strings.NewReader(authenticateStr))
	require.NoError(t, err)

	verified, err := types.VerifyAuthentication(rp, bec32Addr, challenge, cred, authData)
	require.NoError(t, err)
	require.True(t, verified)
}

func TestGenerateWebAuthNSignature(t *testing.T) {
	challenge := base64url.Encode([]byte("test-challenge"))
	signature := CreateWebAuthNSignature(t, []byte(challenge))
	require.NotNil(t, signature)
}
