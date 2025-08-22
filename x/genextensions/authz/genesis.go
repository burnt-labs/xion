package authz

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/core/address"
	"github.com/burnt-labs/xion/x/genextensions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
)

const (
	AuthzExportItemTypeAuthorization = 1
)

type AuthzGenesisExtension struct {
	keeper    authzkeeper.Keeper
	cdc       codec.BinaryCodec
	addrCodec address.Codec
	grantPool *sync.Pool
}

func NewAuthzGenesisExtension(keeper authzkeeper.Keeper, cdc codec.BinaryCodec, addrCodec address.Codec) *AuthzGenesisExtension {
	return &AuthzGenesisExtension{
		keeper:    keeper,
		cdc:       cdc,
		addrCodec: addrCodec,
		grantPool: &sync.Pool{
			New: func() any {
				return &authz.GrantAuthorization{}
			},
		},
	}
}

func (e *AuthzGenesisExtension) Export(ctx context.Context, export func(types.ExportItem) error) error {
	var err error
	e.keeper.IterateGrants(ctx, func(granter, grantee sdk.AccAddress, grant authz.Grant) bool {
		ga := authz.GrantAuthorization{
			Granter:       granter.String(),
			Grantee:       grantee.String(),
			Expiration:    grant.Expiration,
			Authorization: grant.Authorization,
		}

		bz := e.cdc.MustMarshal(&ga)
		err = export(types.ExportItem{
			Type:  AuthzExportItemTypeAuthorization,
			Value: bz,
		})
		return err != nil
	})
	return err
}

func (e *AuthzGenesisExtension) Import(ctx sdk.Context, item *types.ExportItem) error {
	switch item.Type {
	case AuthzExportItemTypeAuthorization:
		ga := e.grantPool.Get().(*authz.GrantAuthorization)
		defer e.grantPool.Put(ga)
		// reset the grant object from the pool
		*ga = authz.GrantAuthorization{}
		e.cdc.MustUnmarshal(item.Value, ga)
		return e.importGrant(ctx, ga)
	}
	return nil
}

// ImportGenesis imports the authz genesis state from a given context.
// This is a copy of x/authz/keeper/genesis.go:InitGenesis but used for stream import.
func (e *AuthzGenesisExtension) importGrant(ctx sdk.Context, entry *authz.GrantAuthorization) error {
	now := ctx.BlockTime()
	// ignore expired authorizations
	if entry.Expiration != nil && entry.Expiration.Before(now) {
		return nil
	}

	grantee, err := e.addrCodec.StringToBytes(entry.Grantee)
	if err != nil {
		panic(err)
	}
	granter, err := e.addrCodec.StringToBytes(entry.Granter)
	if err != nil {
		panic(err)
	}

	a, ok := entry.Authorization.GetCachedValue().(authz.Authorization)
	if !ok {
		return fmt.Errorf("expected authorization")
	}

	err = e.keeper.SaveGrant(ctx, grantee, granter, a, entry.Expiration)
	if err != nil {
		return err
	}

	return nil
}
