package types_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

func TestVerifyRejectsRelyingPartyWithUnparsableHost(t *testing.T) {
	// A percent sign that does not start a valid escape makes the host, and so
	// the derived RPID, fail URL validation inside the webauthn library.
	rp := &url.URL{Scheme: "https", Host: "%zz"}
	ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)

	t.Run("registration", func(t *testing.T) {
		_, err := types.VerifyRegistration(ctx, rp, testContractAddr, testChallenge, &protocol.ParsedCredentialCreationData{})
		require.ErrorContains(t, err, "RPID")
	})

	t.Run("authentication", func(t *testing.T) {
		credential := &webauthn.Credential{ID: []byte("id")}
		_, err := types.VerifyAuthentication(ctx, rp, testContractAddr, testChallenge, credential, &protocol.ParsedCredentialAssertionData{})
		require.ErrorContains(t, err, "RPID")
	})
}

func attestationWithGarbageCertificate(format string) *protocol.ParsedCredentialCreationData {
	return &protocol.ParsedCredentialCreationData{
		ParsedPublicKeyCredential: protocol.ParsedPublicKeyCredential{
			ParsedCredential: protocol.ParsedCredential{ID: "credential", Type: "public-key"},
		},
		Response: protocol.ParsedAttestationResponse{
			CollectedClientData: protocol.CollectedClientData{
				Type:      "webauthn.create",
				Challenge: testChallenge,
				Origin:    "https://example.com",
			},
			AttestationObject: protocol.AttestationObject{
				Format: format,
				AttStatement: map[string]any{
					"x5c": []any{[]byte("not a certificate")},
				},
			},
		},
	}
}

func TestCreateCredentialCertificateHandlingByFormat(t *testing.T) {
	webAuth, err := webauthn.New(&webauthn.Config{
		RPID:          "example.com",
		RPDisplayName: "Example",
		RPOrigins:     []string{"https://example.com"},
	})
	require.NoError(t, err)

	ctx := sdktypes.NewContext(nil, cmtproto.Header{Time: time.Now()}, false, nil)
	user := types.SmartContractUser{Address: testContractAddr}
	session := webauthn.SessionData{
		Challenge:        testChallenge,
		UserID:           user.WebAuthnID(),
		UserVerification: protocol.VerificationPreferred,
	}

	t.Run("none format skips the certificate chain", func(t *testing.T) {
		_, err := types.CreateCredential(webAuth, ctx, user, session, attestationWithGarbageCertificate("none"))
		require.Error(t, err)
		require.NotContains(t, err.Error(), "Failed to parse X.509 certificate")
	})

	t.Run("other formats parse the certificate chain", func(t *testing.T) {
		_, err := types.CreateCredential(webAuth, ctx, user, session, attestationWithGarbageCertificate("packed"))
		require.ErrorContains(t, err, "Failed to parse X.509 certificate")
	})
}
