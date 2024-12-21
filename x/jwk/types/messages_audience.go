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

func (msg *MsgCreateAudience) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
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

	return nil
}

var _ sdk.Msg = &MsgUpdateAudience{}

func NewMsgUpdateAudience(
	admin string,
	newAdmin string,
	aud string,
	key string,
) *MsgUpdateAudience {
	return &MsgUpdateAudience{
		NewAdmin: newAdmin,
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

	return nil
}
