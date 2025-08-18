package feegrant

import (
	"context"

	"sync"

	"cosmossdk.io/core/address"
	feegrant "cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	"github.com/burnt-labs/xion/x/genextensions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	FeeGrantExportItemTypeAllowance = 1
)

type FeeGrantGenesisExtension struct {
	keeper    feegrantkeeper.Keeper
	cdc       codec.BinaryCodec
	addrCodec address.Codec
	grantPool *sync.Pool
}

func NewFeeGrantGenesisExtension(keeper feegrantkeeper.Keeper, cdc codec.BinaryCodec, addrCodec address.Codec) *FeeGrantGenesisExtension {
	return &FeeGrantGenesisExtension{
		keeper:    keeper,
		cdc:       cdc,
		addrCodec: addrCodec,
		grantPool: &sync.Pool{
			New: func() any {
				return &feegrant.Grant{}
			},
		},
	}
}

func (e *FeeGrantGenesisExtension) Export(ctx context.Context, export func(types.ExportItem) error) error {

	var err error
	e.keeper.IterateAllFeeAllowances(ctx, func(grant feegrant.Grant) bool {
		bz := e.cdc.MustMarshal(&grant)
		err = export(types.ExportItem{
			Type:  FeeGrantExportItemTypeAllowance,
			Value: bz,
		})
		// if error, stop iterating
		return err != nil

	})
	return err
}

func (e *FeeGrantGenesisExtension) Import(ctx sdk.Context, item *types.ExportItem) error {
	switch item.Type {
	case FeeGrantExportItemTypeAllowance:
		grant := e.grantPool.Get().(*feegrant.Grant)
		defer e.grantPool.Put(grant)
		// reset the grant object from the pool
		*grant = feegrant.Grant{}
		e.cdc.MustUnmarshal(item.Value, grant)
		return e.importAllowance(ctx, grant)
	}
	return nil
}

// ImportGenesis imports the authz genesis state from a given context.
// This is a copy of x/authz/keeper/genesis.go:InitGenesis but used for stream import.
func (e *FeeGrantGenesisExtension) importAllowance(ctx sdk.Context, entry *feegrant.Grant) error {

	granter, err := e.addrCodec.StringToBytes(entry.Granter)
	if err != nil {
		return err
	}
	grantee, err := e.addrCodec.StringToBytes(entry.Grantee)
	if err != nil {
		return err
	}
	grant, err := entry.GetGrant()
	if err != nil {
		return err
	}
	exp, err := grant.ExpiresAt()
	if err != nil {
		return err
	}
	// if the grant expired before the genesis block then just skip it
	if exp != nil && exp.Before(ctx.BlockTime()) {
		return nil
	}
	err = e.keeper.GrantAllowance(ctx, granter, grantee, grant)
	if err != nil {
		return err
	}

	return nil
}
