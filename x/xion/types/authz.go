package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ authztypes.Authorization         = &CodeIdExecutionAuthorization{}
	_ cdctypes.UnpackInterfacesMessage = &CodeIdExecutionAuthorization{}
)

// ContractAuthzFactory factory to create an updated Authorization object
type CodeIdAuthzFactory interface {
	NewAuthz([]CodeIdGrant) authztypes.Authorization
}

// NewCodeIdExecutionAuthorization constructor
func NewCodeIdExecutionAuthorization(grants ...CodeIdGrant) *CodeIdExecutionAuthorization {
	return &CodeIdExecutionAuthorization{
		Grants: grants,
	}
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (a CodeIdExecutionAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&wasmtypes.MsgExecuteContract{})
}

// NewAuthz factory method to create an Authorization with updated grants
func (a CodeIdExecutionAuthorization) NewAuthz(g []CodeIdGrant) authztypes.Authorization {
	return NewCodeIdExecutionAuthorization(g...)
}

// Accept implements Authorization.Accept.
func (a *CodeIdExecutionAuthorization) Accept(ctx sdk.Context, msg sdk.Msg) (authztypes.AcceptResponse, error) {
	return AcceptGrantedMessage[*wasmtypes.MsgExecuteContract](ctx, a.Grants, msg, a)
}

// ValidateBasic implements Authorization.ValidateBasic.
func (a CodeIdExecutionAuthorization) ValidateBasic() error {
	return validateGrants(a.Grants)
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (a CodeIdExecutionAuthorization) UnpackInterfaces(unpacker cdctypes.AnyUnpacker) error {
	for _, g := range a.Grants {
		if err := g.UnpackInterfaces(unpacker); err != nil {
			return err
		}
	}
	return nil
}

// validateGrants validates the grants
func validateGrants(g []CodeIdGrant) error {
	if len(g) == 0 {
		return wasmtypes.ErrEmpty.Wrap("grants")
	}
	for i, v := range g {
		if err := v.ValidateBasic(); err != nil {
			return errorsmod.Wrapf(err, "position %d", i)
		}
	}
	// allow multiple grants for a contract:
	// contractA:doThis:1,doThat:*  has with different counters for different methods
	return nil
}

// AcceptGrantedMessage determines whether this grant permits the provided sdk.Msg to be performed,
// and if so provides an upgraded authorization instance.
func AcceptGrantedMessage[T wasmtypes.AuthzableWasmMsg](ctx sdk.Context, grants []CodeIdGrant, msg sdk.Msg, factory CodeIdAuthzFactory) (authztypes.AcceptResponse, error) {
	exec, ok := msg.(T)
	if !ok {
		return authztypes.AcceptResponse{}, sdkerrors.ErrInvalidType.Wrap("type mismatch")
	}
	if exec.GetMsg() == nil {
		return authztypes.AcceptResponse{}, sdkerrors.ErrInvalidType.Wrap("empty message")
	}
	if err := exec.ValidateBasic(); err != nil {
		return authztypes.AcceptResponse{}, err
	}

	// todo: is this the best way to access code id?
	store := ctx.KVStore(sdk.NewKVStoreKey(wasmtypes.StoreKey))
	var contract wasmtypes.ContractInfo
	contractAddr := sdk.MustAccAddressFromBech32(exec.GetContract())
	contractBz := store.Get(wasmtypes.GetContractAddressKey(contractAddr))
	if contractBz == nil {
		return authztypes.AcceptResponse{}, sdkerrors.ErrNotFound.Wrap("contract not found")
	}
	if err := contract.Unmarshal(contractBz); err != nil {
		return authztypes.AcceptResponse{}, err
	}

	// iterate though all grants
	for i, g := range grants {
		contractGrant := wasmtypes.ContractGrant{
			Contract: "",
			Limit:    g.Limit,
			Filter:   g.Filter,
		}
		// TODO: Make sure the contract is instantiated from the code id
		if contract.CodeID != g.CodeId {
			continue
		}

		// first check limits
		result, err := contractGrant.GetLimit().Accept(ctx, exec)
		switch {
		case err != nil:
			return authztypes.AcceptResponse{}, errorsmod.Wrap(err, "limit")
		case result == nil: // sanity check
			return authztypes.AcceptResponse{}, sdkerrors.ErrInvalidType.Wrap("limit result must not be nil")
		case !result.Accepted:
			// not applicable, continue with next grant
			continue
		}

		// then check permission set
		ok, err := contractGrant.GetFilter().Accept(ctx, exec.GetMsg())
		switch {
		case err != nil:
			return authztypes.AcceptResponse{}, errorsmod.Wrap(err, "filter")
		case !ok:
			// no limit update and continue with next grant
			continue
		}

		// finally do limit state updates in result
		switch {
		case result.DeleteLimit:
			updatedGrants := append(grants[0:i], grants[i+1:]...)
			if len(updatedGrants) == 0 { // remove when empty
				return authztypes.AcceptResponse{Accept: true, Delete: true}, nil
			}
			newAuthz := factory.NewAuthz(updatedGrants)
			if err := newAuthz.ValidateBasic(); err != nil { // sanity check
				return authztypes.AcceptResponse{}, wasmtypes.ErrInvalid.Wrapf("new grant state: %s", err)
			}
			return authztypes.AcceptResponse{Accept: true, Updated: newAuthz}, nil
		case result.UpdateLimit != nil:
			obj, err := g.WithNewLimits(result.UpdateLimit)
			if err != nil {
				return authztypes.AcceptResponse{}, err
			}
			newAuthz := factory.NewAuthz(append(append(grants[0:i], *obj), grants[i+1:]...))
			if err := newAuthz.ValidateBasic(); err != nil { // sanity check
				return authztypes.AcceptResponse{}, wasmtypes.ErrInvalid.Wrapf("new grant state: %s", err)
			}
			return authztypes.AcceptResponse{Accept: true, Updated: newAuthz}, nil
		default: // accepted without a limit state update
			return authztypes.AcceptResponse{Accept: true}, nil
		}
	}
	return authztypes.AcceptResponse{Accept: false}, nil
}
