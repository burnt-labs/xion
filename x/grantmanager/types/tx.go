package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (m *MsgRevokeAuthzGrants) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Granter); err != nil {
		return err
	}
	return nil
}

func (m *MsgRevokeFeegrantAllowances) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Granter); err != nil {
		return err
	}
	return nil
}
