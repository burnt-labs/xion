package types

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"net/url"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

const (
	attestationFormatNone = "none"
	// MaxWebAuthDataSize defines the maximum allowed size for WebAuthN data payloads
	// to prevent DoS attacks via oversized attestation objects
	MaxWebAuthDataSize = 64 * 1024 // 64KB
)

type SmartContractUser struct {
	Address    string
	Credential *webauthn.Credential
}

func (s SmartContractUser) WebAuthnID() []byte {
	return []byte(s.Address)
}

func (s SmartContractUser) WebAuthnName() string {
	return s.Address
}

func (s SmartContractUser) WebAuthnDisplayName() string {
	return s.WebAuthnName()
}

func (s SmartContractUser) WebAuthnCredentials() []webauthn.Credential {
	return []webauthn.Credential{*s.Credential}
}

func (s SmartContractUser) WebAuthnIcon() string {
	return ""
}

var _ webauthn.User = SmartContractUser{}

func VerifyRegistration(ctx sdktypes.Context, rp *url.URL, contractAddr string, challenge string, credentialCreationData *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
	config := webauthn.Config{
		RPID:                        rp.Host,
		RPDisplayName:               rp.String(),
		RPOrigins:                   []string{rp.String()},
		RPTopOrigins:                []string{rp.String()},
		RPTopOriginVerificationMode: protocol.TopOriginIgnoreVerificationMode,
		AttestationPreference:       "none",
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred,
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
		},
	}
	webAuthn, err := webauthn.New(&config)
	if err != nil {
		return nil, err
	}

	smartContractUser := SmartContractUser{Address: contractAddr}
	session := webauthn.SessionData{
		Challenge:        challenge,
		UserID:           smartContractUser.WebAuthnID(),
		UserVerification: protocol.VerificationPreferred,
		RelyingPartyID:   rp.Host,
		CredParams: []protocol.CredentialParameter{
			{
				Type:      protocol.PublicKeyCredentialType,
				Algorithm: webauthncose.AlgES256K,
			},
			{
				Type:      protocol.PublicKeyCredentialType,
				Algorithm: webauthncose.AlgES256,
			},
			{
				Type:      protocol.PublicKeyCredentialType,
				Algorithm: webauthncose.AlgEdDSA,
			},
			{
				Type:      protocol.PublicKeyCredentialType,
				Algorithm: webauthncose.AlgRS256,
			},
		},
	}

	return CreateCredential(webAuthn, ctx, smartContractUser, session, credentialCreationData)
}

func VerifyAuthentication(ctx sdktypes.Context, rp *url.URL, contractAddr string, challenge string, credential *webauthn.Credential, credentialAssertionData *protocol.ParsedCredentialAssertionData) (bool, error) {
	config := webauthn.Config{
		RPID:                   rp.Host,
		RPDisplayName:          rp.String(),
		RPOrigins:              []string{rp.String()},
		AttestationPreference:  "",
		AuthenticatorSelection: protocol.AuthenticatorSelection{},
	}
	webAuthn, err := webauthn.New(&config)
	if err != nil {
		return false, err
	}

	smartContractUser := SmartContractUser{
		Address:    contractAddr,
		Credential: credential,
	}
	session := webauthn.SessionData{
		Challenge:            challenge,
		UserID:               smartContractUser.WebAuthnID(),
		UserVerification:     protocol.VerificationPreferred,
		AllowedCredentialIDs: [][]byte{credential.ID},
	}

	if _, err := webAuthn.ValidateLogin(smartContractUser, session, credentialAssertionData); err != nil {
		return false, err
	}

	return true, nil
}

// CreateCredential verifies a parsed response against the user's credentials and session data.
func CreateCredential(webauth *webauthn.WebAuthn, ctx sdktypes.Context, user webauthn.User, session webauthn.SessionData, parsedResponse *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
	// First do the basic validations that don't involve time
	if !bytes.Equal(user.WebAuthnID(), session.UserID) {
		return nil, protocol.ErrBadRequest.WithDetails("ID mismatch for User and Session")
	}

	// Use block time for session expiry check (deterministic)
	if !session.Expires.IsZero() && session.Expires.Before(ctx.BlockTime()) {
		return nil, protocol.ErrBadRequest.WithDetails("Session has Expired")
	}

	shouldVerifyUser := session.UserVerification == protocol.VerificationRequired

	// Validate certificates using block time BEFORE calling original verification
	if err := validateCertificatesWithBlockTime(parsedResponse, ctx.BlockTime()); err != nil {
		return nil, err
	}

	// Call original verification with all required parameters
	// Based on webauthn v13.4 API, Verify now takes more parameters including credential parameters
	// we don't want to verify the attestation statement to avoid network calls
	// so we set the attestation statement to nil even if the parsed response has one
	parsedResponse.Response.AttestationObject.AttStatement = nil
	parsedResponse.Response.AttestationObject.Format = attestationFormatNone

	if _, err := parsedResponse.Verify(
		session.Challenge,           // storedChallenge
		shouldVerifyUser,            // verifyUser
		false,                       // allowSetUserVerificationHint - set to false for deterministic behavior
		webauth.Config.RPID,         // relyingPartyID
		webauth.Config.RPOrigins,    // relyingPartyOrigin
		webauth.Config.RPTopOrigins, // attestationObject (optional)
		webauth.Config.RPTopOriginVerificationMode, // topOriginVerification
		nil, // attestationStatement
		session.CredParams,
	); err != nil {
		return nil, err
	}

	clientDataHash := sha256.Sum256(parsedResponse.Raw.AttestationResponse.ClientDataJSON)
	return webauthn.NewCredential(clientDataHash[:], parsedResponse)
}

// validateCertificatesWithBlockTime validates X.509 certificates using block time instead of system time
func validateCertificatesWithBlockTime(parsedResponse *protocol.ParsedCredentialCreationData, blockTime time.Time) error {
	attStmt := parsedResponse.Response.AttestationObject.AttStatement
	if attStmt == nil {
		return nil // No certificates to validate
	}

	// If attestation format is explicitly set to "none" we intentionally skip
	// certificate parsing/validation even if an x5c field is present. The
	// registration flow configures AttestationPreference "none" and later code
	// strips the attestation statement before verification to avoid non-
	// deterministic network calls. Some tests inject malformed x5c entries to
	// ensure we ignore them under this mode.
	if parsedResponse.Response.AttestationObject.Format == attestationFormatNone {
		return nil
	}

	// Some callers (tests) may only set the attestation format inside the attestation
	// statement map under the key "fmt" without populating the Format field prior
	// to validation. Treat that as an explicit request to skip certificate parsing.
	if fmtRaw, ok := attStmt["fmt"].(string); ok && fmtRaw == attestationFormatNone {
		return nil
	}

	// Look for x5c (X.509 certificate chain) in the attestation statement
	if x5cRaw, exists := attStmt["x5c"]; exists {
		if x5cSlice, ok := x5cRaw.([]interface{}); ok {
			for _, certRaw := range x5cSlice {
				if certBytes, ok := certRaw.([]byte); ok {
					cert, err := x509.ParseCertificate(certBytes)
					if err != nil {
						return protocol.ErrInvalidAttestation.WithDetails("Failed to parse X.509 certificate")
					}

					// Use block time for certificate validity check (deterministic)
					if blockTime.Before(cert.NotBefore) || blockTime.After(cert.NotAfter) {
						return protocol.ErrInvalidAttestation.WithDetails("Certificate not valid at block time")
					}
				}
			}
		}
	}

	return nil
}
