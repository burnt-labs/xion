package types

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	MaxJWKKeySize = 8192 // 8 KB
	MaxAudSize    = 512

	TypeMsgCreateAudience = "create_audience"
	TypeMsgUpdateAudience = "update_audience"
	TypeMsgDeleteAudience = "delete_audience"

	TypeMsgCreateAudienceClaim = "create_audience_claim"
	TypeMsgDeleteAudienceClaim = "delete_audience_claim"
)

var _ sdk.Msg = &MsgCreateAudience{}

func NewMsgCreateAudience(
	admin string,
	aud string,
	key string,
) *MsgCreateAudience {
	return &MsgCreateAudience{
		Admin: admin,
		Aud:   aud,
		Key:   key,
	}
}

func (msg *MsgCreateAudience) Route() string {
	return RouterKey
}

func (msg *MsgCreateAudience) Type() string {
	return TypeMsgCreateAudience
}

func (msg *MsgCreateAudience) GetSigners() []sdk.AccAddress {
	admin, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{admin}
}

func (msg *MsgCreateAudience) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// validateJWKKeyTypeAlgConsistency checks that the key type (kty) is
// consistent with the signature algorithm (alg).  A mismatch (e.g.
// kty=oct combined with alg=RS256) would be stored successfully but
// would permanently break JWT verification for that audience because
// the verifier would attempt to use the wrong key material.
func validateJWKKeyTypeAlgConsistency(key jwk.Key, sigAlg jwa.SignatureAlgorithm) error {
	kty := key.KeyType()

	switch sigAlg {
	// RSA algorithms require kty=RSA
	case jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512:
		if kty != jwa.RSA {
			return fmt.Errorf("algorithm %s requires kty=RSA, got kty=%s", sigAlg, kty)
		}
	// ECDSA algorithms require kty=EC
	case jwa.ES256, jwa.ES384, jwa.ES512:
		if kty != jwa.EC {
			return fmt.Errorf("algorithm %s requires kty=EC, got kty=%s", sigAlg, kty)
		}
	// EdDSA requires kty=OKP
	case jwa.EdDSA:
		if kty != jwa.OKP {
			return fmt.Errorf("algorithm %s requires kty=OKP, got kty=%s", sigAlg, kty)
		}
	}

	return nil
}

func (msg *MsgCreateAudience) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}

	if len(msg.Key) > MaxJWKKeySize {
		return errorsmod.Wrapf(ErrInvalidJWK, "key size %d exceeds maximum %d bytes", len(msg.Key), MaxJWKKeySize)
	}
	if len(msg.Aud) > MaxAudSize {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "aud length %d exceeds maximum %d", len(msg.Aud), MaxAudSize)
	}

	key, err := jwk.ParseKey([]byte(msg.Key))
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidJWK, "invalid jwk format (%s)", err)
	}

	var sigAlg jwa.SignatureAlgorithm
	if err := sigAlg.Accept(key.Algorithm().String()); err != nil {
		return err
	}

	switch sigAlg {
	case jwa.HS256, jwa.HS384, jwa.HS512, jwa.NoSignature:
		return fmt.Errorf("invalid algorithm: %s", sigAlg.String())
	}

	if err := validateJWKKeyTypeAlgConsistency(key, sigAlg); err != nil {
		return errorsmod.Wrapf(ErrInvalidJWK, "%s", err)
	}

	return nil
}

var _ sdk.Msg = &MsgUpdateAudience{}

func NewMsgUpdateAudience(
	admin string,
	newAdmin string,
	aud string,
	newAud string,
	key string,
) *MsgUpdateAudience {
	return &MsgUpdateAudience{
		NewAdmin: newAdmin,
		NewAud:   newAud,
		Admin:    admin,
		Aud:      aud,
		Key:      key,
	}
}

func (msg *MsgUpdateAudience) Route() string {
	return RouterKey
}

func (msg *MsgUpdateAudience) Type() string {
	return TypeMsgUpdateAudience
}

func (msg *MsgUpdateAudience) GetSigners() []sdk.AccAddress {
	admin, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{admin}
}

func (msg *MsgUpdateAudience) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUpdateAudience) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.NewAdmin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}

	if len(msg.Key) > MaxJWKKeySize {
		return errorsmod.Wrapf(ErrInvalidJWK, "key size %d exceeds maximum %d bytes", len(msg.Key), MaxJWKKeySize)
	}
	if len(msg.Aud) > MaxAudSize {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "aud length %d exceeds maximum %d", len(msg.Aud), MaxAudSize)
	}
	if len(msg.NewAud) > MaxAudSize {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "new_aud length %d exceeds maximum %d", len(msg.NewAud), MaxAudSize)
	}

	key, err := jwk.ParseKey([]byte(msg.Key))
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidJWK, "invalid jwk format (%s)", err)
	}

	var sigAlg jwa.SignatureAlgorithm
	if err := sigAlg.Accept(key.Algorithm().String()); err != nil {
		return err
	}

	switch sigAlg {
	case jwa.HS256, jwa.HS384, jwa.HS512, jwa.NoSignature:
		return fmt.Errorf("invalid algorithm: %s", sigAlg.String())
	}

	if err := validateJWKKeyTypeAlgConsistency(key, sigAlg); err != nil {
		return errorsmod.Wrapf(ErrInvalidJWK, "%s", err)
	}

	return nil
}

var _ sdk.Msg = &MsgDeleteAudience{}

func NewMsgDeleteAudience(
	admin string,
	aud string,
) *MsgDeleteAudience {
	return &MsgDeleteAudience{
		Admin: admin,
		Aud:   aud,
	}
}

func (msg *MsgDeleteAudience) Route() string {
	return RouterKey
}

func (msg *MsgDeleteAudience) Type() string {
	return TypeMsgDeleteAudience
}

func (msg *MsgDeleteAudience) GetSigners() []sdk.AccAddress {
	admin, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{admin}
}

func (msg *MsgDeleteAudience) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDeleteAudience) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}
	return nil
}

var _ sdk.Msg = &MsgCreateAudienceClaim{}

func NewMsgCreateAudienceClaim(
	admin sdk.AccAddress,
	hash []byte,
) *MsgCreateAudienceClaim {
	return &MsgCreateAudienceClaim{
		Admin:   admin.String(),
		AudHash: hash,
	}
}

func (msg *MsgCreateAudienceClaim) Route() string {
	return RouterKey
}

func (msg *MsgCreateAudienceClaim) Type() string {
	return TypeMsgCreateAudienceClaim
}

func (msg *MsgCreateAudienceClaim) GetSigners() []sdk.AccAddress {
	admin, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{admin}
}

func (msg *MsgCreateAudienceClaim) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreateAudienceClaim) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}

	if len(msg.AudHash) != 32 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "hash must be 32 byte sha256")
	}

	return nil
}

var _ sdk.Msg = &MsgDeleteAudienceClaim{}

func NewMsgDeleteAudienceClaim(
	admin sdk.AccAddress,
	hash []byte,
) *MsgDeleteAudienceClaim {
	return &MsgDeleteAudienceClaim{
		Admin:   admin.String(),
		AudHash: hash,
	}
}

func (msg *MsgDeleteAudienceClaim) Route() string {
	return RouterKey
}

func (msg *MsgDeleteAudienceClaim) Type() string {
	return TypeMsgDeleteAudienceClaim
}

func (msg *MsgDeleteAudienceClaim) GetSigners() []sdk.AccAddress {
	admin, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{admin}
}

func (msg *MsgDeleteAudienceClaim) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDeleteAudienceClaim) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}

	if len(msg.AudHash) != 32 {
		return errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"audience hash must be 32-byte SHA-256 (got %d bytes)",
			len(msg.AudHash),
		)
	}

	return nil
}
