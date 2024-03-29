package types

import (
	"net/url"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type SmartContractUser struct {
	address    string
	credential *webauthn.Credential
}

func (s SmartContractUser) WebAuthnID() []byte {

	return []byte(s.address)
}

func (s SmartContractUser) WebAuthnName() string {
	return s.address
}

func (s SmartContractUser) WebAuthnDisplayName() string {
	return s.WebAuthnName()
}

func (s SmartContractUser) WebAuthnCredentials() []webauthn.Credential {
	return []webauthn.Credential{*s.credential}
}

func (s SmartContractUser) WebAuthnIcon() string {
	return ""
}

var _ webauthn.User = SmartContractUser{}

func VerifyRegistration(rp *url.URL, contractAddr string, challenge string, credentialCreationData *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
	config := webauthn.Config{
		RPID:                   rp.Host,
		RPDisplayName:          rp.String(),
		RPOrigins:              []string{rp.String()},
		AttestationPreference:  "",
		AuthenticatorSelection: protocol.AuthenticatorSelection{},
	}
	webAuthn, err := webauthn.New(&config)
	if err != nil {
		return nil, err
	}

	var smartContractUser = SmartContractUser{address: contractAddr}
	var session = webauthn.SessionData{
		Challenge:        challenge,
		UserID:           smartContractUser.WebAuthnID(),
		UserVerification: protocol.VerificationPreferred,
	}

	return webAuthn.CreateCredential(smartContractUser, session, credentialCreationData)
}

func VerifyAuthentication(rp *url.URL, contractAddr string, challenge string, credential *webauthn.Credential, credentialAssertionData *protocol.ParsedCredentialAssertionData) (bool, error) {
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

	var smartContractUser = SmartContractUser{
		address:    contractAddr,
		credential: credential,
	}
	var session = webauthn.SessionData{
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
