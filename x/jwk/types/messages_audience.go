package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

const (
	TypeMsgCreateAudience = "create_audience"
	TypeMsgUpdateAudience = "update_audience"
	TypeMsgDeleteAudience = "delete_audience"
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
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}

	_, err = jwk.ParseKey([]byte(msg.Key))
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidJWK, "invalid jwk format (%s)", err)
	}

	return nil
}

var _ sdk.Msg = &MsgUpdateAudience{}

func NewMsgUpdateAudience(
	signer string,
	admin string,
	aud string,
	key string,

) *MsgUpdateAudience {
	return &MsgUpdateAudience{
		Signer: signer,
		Admin:  admin,
		Aud:    aud,
		Key:    key,
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
	_, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
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
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}
	return nil
}
