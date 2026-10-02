package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

type msgServer struct {
	k Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{k}
}

// ------------------------------- UpdateParams --------------------------------

func (ms msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.Sender != ms.k.authority {
		return nil, sdkerrors.ErrUnauthorized.Wrapf("sender is not authority: expect %s, found %s", ms.k.authority, req.Sender)
	}

	// Defense in depth alongside ValidateBasic: Params is dereferenced below.
	if req.Params == nil {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("params cannot be nil")
	}

	currentParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if currentParams.RegistrationConfigured() &&
		!bytes.Equal(req.Params.AddressDerivationHash, currentParams.AddressDerivationHash) {
		return nil, types.ErrImmutableAddressHash.Wrapf(
			"expected %s, found %s",
			hex.EncodeToString(currentParams.AddressDerivationHash),
			hex.EncodeToString(req.Params.AddressDerivationHash),
		)
	}
	if err := req.Params.Validate(); err != nil {
		return nil, err
	}

	if err := ms.k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// ------------------------------ RegisterAccount ------------------------------

func (ms msgServer) ensureAccountNamespaceAvailable(
	ctx sdk.Context,
	sender sdk.AccAddress,
	salt []byte,
) (sdk.AccAddress, error) {
	if address, found := ms.k.GetAccountAddress(ctx, sender, salt); found {
		return nil, types.ErrAccountAlreadyRegistered.Wrap(address.String())
	}

	predicted, err := ms.k.PredictAccountAddress(ctx, sender, salt)
	if err != nil {
		return nil, err
	}
	if ms.k.IsAbstractAccount(ctx, predicted) {
		return nil, types.ErrAccountAlreadyRegistered.Wrap(predicted.String())
	}

	return predicted, nil
}

func (ms msgServer) instantiateAccount(
	ctx sdk.Context,
	params *types.Params,
	sender, predicted sdk.AccAddress,
	req *types.MsgRegisterAccount,
) (sdk.AccAddress, []byte, error) {
	contractAddr, data, err := ms.k.ck.Instantiate2WithAddressHash(
		ctx,
		req.CodeID,
		params.AddressDerivationHash,
		sender,
		sender,
		req.Msg,
		fmt.Sprintf("%s/%d", types.ModuleName, ms.k.GetAndIncrementNextAccountID(ctx)),
		req.Funds,
		req.Salt,
	)
	if err != nil {
		return nil, nil, err
	}
	if !contractAddr.Equals(predicted) {
		return nil, nil, fmt.Errorf("instantiated account address %s does not match predicted address %s", contractAddr, predicted)
	}

	return contractAddr, data, nil
}

func (ms msgServer) validateRegistrationRequest(
	ctx sdk.Context,
	params *types.Params,
	req *types.MsgRegisterAccount,
) error {
	if !params.RegistrationEnabled {
		return types.ErrRegistrationDisabled
	}
	if !params.IsAllowed(req.CodeID) {
		return types.ErrNotAllowedCodeID.Wrapf("registration implementation code ID %d", req.CodeID)
	}
	if ms.k.vk.GetCodeInfo(ctx, req.CodeID) == nil {
		return types.ErrCodeIDNotFound.Wrapf("implementation code ID %d", req.CodeID)
	}

	return nil
}

func (ms msgServer) RegisterAccount(goCtx context.Context, req *types.MsgRegisterAccount) (*types.MsgRegisterAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	if err := ms.validateRegistrationRequest(ctx, params, req); err != nil {
		return nil, err
	}

	senderAddr, err := sdk.AccAddressFromBech32(req.Sender)
	if err != nil {
		return nil, err
	}
	predictedAddr, err := ms.ensureAccountNamespaceAvailable(ctx, senderAddr, req.Salt)
	if err != nil {
		return nil, err
	}

	contractAddr, data, err := ms.instantiateAccount(ctx, params, senderAddr, predictedAddr, req)
	if err != nil {
		return nil, err
	}

	// set the contract's admin to itself
	if err = ms.k.ck.UpdateContractAdmin(ctx, contractAddr, senderAddr, contractAddr); err != nil {
		return nil, err
	}

	// the contract instantiation should have created a BaseAccount
	acc := ms.k.ak.GetAccount(ctx, contractAddr)
	if _, ok := acc.(*authtypes.BaseAccount); !ok {
		return nil, types.ErrNotBaseAccount
	}

	// we overwrite this BaseAccount with our AbstractAccount
	ms.k.ak.SetAccount(ctx, types.NewAbstractAccountFromAccount(acc))
	ms.k.SetAccountAddress(ctx, senderAddr, req.Salt, contractAddr)

	ms.k.Logger(ctx).Info(
		"account registered",
		types.AttributeKeyCreator, req.Sender,
		types.AttributeKeyAddressDerivationHash, hex.EncodeToString(params.AddressDerivationHash),
		types.AttributeKeyCodeID, req.CodeID,
		types.AttributeKeyContractAddr, contractAddr.String(),
		types.AttributeKeyAccountNumber, acc.GetAccountNumber(),
	)

	if err = ctx.EventManager().EmitTypedEvent(&types.EventAccountRegistered{
		Creator:               req.Sender,
		CodeID:                req.CodeID,
		AddressDerivationHash: params.AddressDerivationHash,
		ContractAddr:          contractAddr.String(),
		AccountNumber:         acc.GetAccountNumber(),
	}); err != nil {
		return nil, err
	}

	return &types.MsgRegisterAccountResponse{Address: contractAddr.String(), Data: data}, nil
}
