package types

import (
	"context"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// TODO: Revisit this once we have proper gas fee framework.
// Tracking issues https://github.com/cosmos/cosmos-sdk/issues/9054, https://github.com/cosmos/cosmos-sdk/discussions/9072
const (
	gasCostPerIteration = uint64(10)
)

var (
	_ feegrant.FeeAllowanceI        = (*AuthzAllowance)(nil)
	_ feegrant.FeeAllowanceI        = (*ContractsAllowance)(nil)
	_ feegrant.FeeAllowanceI        = (*MultiAnyAllowance)(nil)
	_ types.UnpackInterfacesMessage = (*AuthzAllowance)(nil)
	_ types.UnpackInterfacesMessage = (*ContractsAllowance)(nil)
	_ types.UnpackInterfacesMessage = (*MultiAnyAllowance)(nil)
)

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (a *AuthzAllowance) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	var allowance feegrant.FeeAllowanceI
	return unpacker.UnpackAny(a.Allowance, &allowance)
}

func NewAuthzAllowance(allowance feegrant.FeeAllowanceI, authzGrantee sdk.AccAddress) (*AuthzAllowance, error) {
	msg, ok := allowance.(proto.Message)
	if !ok {
		return nil, errorsmod.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", msg)
	}
	anyAllowance, err := types.NewAnyWithValue(msg)
	if err != nil {
		return nil, err
	}

	return &AuthzAllowance{
		Allowance:    anyAllowance,
		AuthzGrantee: authzGrantee.String(),
	}, nil
}

// GetAllowance returns allowed fee allowance.
func (a *AuthzAllowance) GetAllowance() (feegrant.FeeAllowanceI, error) {
	allowance, ok := a.Allowance.GetCachedValue().(feegrant.FeeAllowanceI)
	if !ok {
		return nil, errorsmod.Wrap(feegrant.ErrNoAllowance, "failed to get allowance")
	}

	return allowance, nil
}

// SetAllowance sets allowed fee allowance.
func (a *AuthzAllowance) SetAllowance(allowance feegrant.FeeAllowanceI) error {
	var err error
	a.Allowance, err = types.NewAnyWithValue(allowance.(proto.Message))
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", allowance)
	}

	return nil
}

func (a *AuthzAllowance) Accept(ctx context.Context, fee sdk.Coins, msgs []sdk.Msg) (bool, error) {
	subMsgs, ok := a.allMsgTypesAuthz(ctx, msgs)
	if !ok {
		return false, errorsmod.Wrap(feegrant.ErrMessageNotAllowed, "messages are not authz")
	}

	allowance, err := a.GetAllowance()
	if err != nil {
		return false, err
	}

	remove, err := allowance.Accept(ctx, fee, subMsgs)
	if err == nil && !remove {
		if err = a.SetAllowance(allowance); err != nil {
			return false, err
		}
	}
	return remove, err
}

func (a *AuthzAllowance) allMsgTypesAuthz(ctx context.Context, msgs []sdk.Msg) ([]sdk.Msg, bool) {
	var subMsgs []sdk.Msg

	for _, msg := range msgs {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.GasMeter().ConsumeGas(gasCostPerIteration, "check msg")

		authzMsg, ok := msg.(*authz.MsgExec)
		if !ok {
			return nil, false
		}
		if authzMsg.Grantee != a.AuthzGrantee {
			return nil, false
		}

		msgMsgs, err := authzMsg.GetMessages()
		if err != nil {
			return nil, false
		}
		subMsgs = append(subMsgs, msgMsgs...)
	}

	return subMsgs, true
}

func (a *AuthzAllowance) ValidateBasic() error {
	if a.Allowance == nil {
		return errorsmod.Wrap(feegrant.ErrNoAllowance, "allowance should not be empty")
	}

	if _, err := sdk.AccAddressFromBech32(a.AuthzGrantee); err != nil {
		return err
	}

	allowance, err := a.GetAllowance()
	if err != nil {
		return err
	}

	return allowance.ValidateBasic()
}

func (a *AuthzAllowance) ExpiresAt() (*time.Time, error) {
	allowance, err := a.GetAllowance()
	if err != nil {
		return nil, err
	}
	return allowance.ExpiresAt()
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (a *ContractsAllowance) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	var allowance feegrant.FeeAllowanceI
	return unpacker.UnpackAny(a.Allowance, &allowance)
}

func NewContractsAllowance(allowance feegrant.FeeAllowanceI, allowedContractAddrs []sdk.AccAddress) (*ContractsAllowance, error) {
	msg, ok := allowance.(proto.Message)
	if !ok {
		return nil, errorsmod.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", msg)
	}
	anyAllowance, err := types.NewAnyWithValue(msg)
	if err != nil {
		return nil, err
	}

	allowedAddrStrings := make([]string, len(allowedContractAddrs))
	for i, addr := range allowedContractAddrs {
		allowedAddrStrings[i] = addr.String()
	}

	return &ContractsAllowance{
		Allowance:         anyAllowance,
		ContractAddresses: allowedAddrStrings,
	}, nil
}

// GetAllowance returns allowed fee allowance.
func (a *ContractsAllowance) GetAllowance() (feegrant.FeeAllowanceI, error) {
	allowance, ok := a.Allowance.GetCachedValue().(feegrant.FeeAllowanceI)
	if !ok {
		return nil, errorsmod.Wrap(feegrant.ErrNoAllowance, "failed to get allowance")
	}

	return allowance, nil
}

// SetAllowance sets allowed fee allowance.
func (a *ContractsAllowance) SetAllowance(allowance feegrant.FeeAllowanceI) error {
	var err error
	a.Allowance, err = types.NewAnyWithValue(allowance.(proto.Message))
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", allowance)
	}

	return nil
}

func (a *ContractsAllowance) Accept(ctx context.Context, fee sdk.Coins, msgs []sdk.Msg) (bool, error) {
	if !a.allMsgsValidWasmExecs(ctx, msgs) {
		return false, errorsmod.Wrap(feegrant.ErrMessageNotAllowed, "messages are not for specific contracts")
	}

	allowance, err := a.GetAllowance()
	if err != nil {
		return false, err
	}

	remove, err := allowance.Accept(ctx, fee, msgs)
	if err == nil && !remove {
		if err = a.SetAllowance(allowance); err != nil {
			return false, err
		}
	}
	return remove, err
}

func (a *ContractsAllowance) allowedContractsToMap(ctx sdk.Context) map[string]bool {
	addrsMap := make(map[string]bool, len(a.ContractAddresses))
	for _, addr := range a.ContractAddresses {
		ctx.GasMeter().ConsumeGas(gasCostPerIteration, "check msg")
		addrsMap[addr] = true
	}

	return addrsMap
}

func (a *ContractsAllowance) allMsgsValidWasmExecs(ctx context.Context, msgs []sdk.Msg) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	addrsMap := a.allowedContractsToMap(sdkCtx)

	for _, msg := range msgs {
		sdkCtx.GasMeter().ConsumeGas(gasCostPerIteration, "check msg")

		wasmMsg, ok := msg.(*wasmtypes.MsgExecuteContract)
		if !ok {
			return false
		}
		if !addrsMap[wasmMsg.Contract] {
			return false
		}
	}

	return true
}

func (a *ContractsAllowance) ValidateBasic() error {
	if a.Allowance == nil {
		return errorsmod.Wrap(feegrant.ErrNoAllowance, "allowance should not be empty")
	}

	if len(a.ContractAddresses) < 1 {
		return errorsmod.Wrap(ErrNoAllowedContracts, "must set contracts for feegrant")
	}

	for _, addr := range a.ContractAddresses {
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return err
		}
	}

	allowance, err := a.GetAllowance()
	if err != nil {
		return err
	}

	return allowance.ValidateBasic()
}

func (a *ContractsAllowance) ExpiresAt() (*time.Time, error) {
	allowance, err := a.GetAllowance()
	if err != nil {
		return nil, err
	}
	return allowance.ExpiresAt()
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (a *MultiAnyAllowance) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	var allowance feegrant.FeeAllowanceI
	for _, innerAllowance := range a.Allowances {
		if err := unpacker.UnpackAny(innerAllowance, &allowance); err != nil {
			return err
		}
	}

	return nil
}

func NewMultiAnyAllowance(allowances []feegrant.FeeAllowanceI) (*MultiAnyAllowance, error) {
	var anyAllowances []*types.Any
	for _, allowance := range allowances {
		msg, ok := allowance.(proto.Message)
		if !ok {
			return nil, errorsmod.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", msg)
		}
		anyAllowance, err := types.NewAnyWithValue(msg)
		if err != nil {
			return nil, err
		}
		anyAllowances = append(anyAllowances, anyAllowance)
	}

	return &MultiAnyAllowance{
		Allowances: anyAllowances,
	}, nil
}

// GetAllowance returns allowed fee allowance.
func (a *MultiAnyAllowance) GetAllowance(index int) (feegrant.FeeAllowanceI, error) {
	allowance, ok := a.Allowances[index].GetCachedValue().(feegrant.FeeAllowanceI)
	if !ok {
		return nil, errorsmod.Wrap(feegrant.ErrNoAllowance, "failed to get allowance")
	}

	return allowance, nil
}

// SetAllowance sets allowed fee allowance.
func (a *MultiAnyAllowance) SetAllowance(index int, allowance feegrant.FeeAllowanceI) error {
	var err error
	a.Allowances[index], err = types.NewAnyWithValue(allowance.(proto.Message))
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", allowance)
	}

	return nil
}

func (a *MultiAnyAllowance) Accept(ctx context.Context, fee sdk.Coins, msgs []sdk.Msg) (bool, error) {
	// accept and charge first allowance that doesn't error
	accepted := false
	for i := range a.Allowances {
		sdk.UnwrapSDKContext(ctx).GasMeter().ConsumeGas(gasCostPerIteration, "check allowance")
		allowance, err := a.GetAllowance(i)
		if err != nil {
			return false, err
		}

		remove, err := allowance.Accept(ctx, fee, msgs)
		if err != nil {
			// the allowance errored, try the next
			continue
		}
		// the allowance was accepted
		accepted = true

		if !remove {
			// update the allowance state
			if err = a.SetAllowance(i, allowance); err != nil {
				return false, err
			}
		} else {
			// if the allowance is complete, remove it from the allowed list
			a.Allowances = append(a.Allowances[:i], a.Allowances[i+1:]...)
		}
		break
	}

	// if no allowances accepted, the allowance doesn't accept
	if !accepted {
		return false, errorsmod.Wrapf(ErrNoValidAllowances, "all allowances errored")
	}

	// if all the allowances have been removed, remove this allowance as well
	return len(a.Allowances) == 0, nil
}

func (a *MultiAnyAllowance) ValidateBasic() error {
	if len(a.Allowances) == 0 {
		return errorsmod.Wrap(feegrant.ErrNoAllowance, "allowance list should contain at least one")
	}

	for i := range a.Allowances {
		allowance, err := a.GetAllowance(i)
		if err != nil {
			return err
		}
		if err := allowance.ValidateBasic(); err != nil {
			return err
		}
	}

	if _, err := a.ExpiresAt(); err != nil {
		return err
	}

	return nil
}

func (a *MultiAnyAllowance) ExpiresAt() (*time.Time, error) {
	// all allowances must expire at the same time
	var expiration *time.Time
	set := false
	for i := range a.Allowances {
		allowance, err := a.GetAllowance(i)
		if err != nil {
			return nil, err
		}
		newExpiration, err := allowance.ExpiresAt()
		if err != nil {
			return nil, err
		}
		if set {
			if !EqTime(expiration, newExpiration) {
				return nil, errorsmod.Wrapf(ErrInconsistentExpiry, "allowance 0 had expiration %v while allowance %d had expiration %v", expiration, i, newExpiration)
			}
		} else {
			set = true
			expiration = newExpiration
		}
	}
	return expiration, nil
}

func EqTime(a, b *time.Time) bool {
	if a != nil && b != nil {
		return a.Equal(*b)
	}
	if a == nil && b == nil {
		return true
	}
	return false
}
