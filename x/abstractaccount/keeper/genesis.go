package keeper

import (
	"bytes"

	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) []abci.ValidatorUpdate {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(err)
	}

	k.SetNextAccountID(ctx, gs.NextAccountId)
	for _, entry := range gs.AccountAddresses {
		sender, err := sdk.AccAddressFromBech32(entry.Sender)
		if err != nil {
			panic(err)
		}
		address, err := sdk.AccAddressFromBech32(entry.Address)
		if err != nil {
			panic(err)
		}
		if !k.IsAbstractAccount(ctx, address) {
			panic(types.ErrInvalidAccountAddressRegistry.Wrapf("genesis account address %s", entry.Address))
		}
		predicted, err := k.PredictAccountAddress(ctx, sender, entry.Salt)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(address, predicted) {
			panic(types.ErrInvalidAccountAddressRegistry.Wrapf(
				"genesis account address %s does not match derived address %s",
				entry.Address,
				predicted.String(),
			))
		}
		if !k.vk.HasContractInfo(ctx, address) {
			panic(types.ErrInvalidAccountAddressRegistry.Wrapf(
				"genesis account address %s has no Wasm contract",
				entry.Address,
			))
		}
		k.SetAccountAddress(ctx, sender, entry.Salt, address)
	}

	return []abci.ValidatorUpdate{}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	var accountAddresses []*types.AccountAddress
	if err := k.IterateAccountAddresses(ctx, func(sender sdk.AccAddress, salt []byte, address sdk.AccAddress) bool {
		accountAddresses = append(accountAddresses, &types.AccountAddress{
			Sender:  sender.String(),
			Salt:    salt,
			Address: address.String(),
		})
		return false
	}); err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Params:           params,
		NextAccountId:    k.GetNextAccountID(ctx),
		AccountAddresses: accountAddresses,
	}
}
