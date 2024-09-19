package keeper

import (
	"context"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/errors"
	dkimv1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
	"github.com/burnt-labs/xion/x/dkim/types"
)

type msgServer struct {
	k Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}

	return nil, ms.k.Params.Set(ctx, msg.Params)
}

// AddDkimPubKey implements types.MsgServer.
func (ms msgServer) AddDkimPubKey(ctx context.Context, msg *types.MsgAddDkimPubKey) (*types.MsgAddDkimPubKeyResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}
	for _, dkimPubKey := range msg.DkimPubkeys {
		if err := ms.k.OrmDB.DkimPubKeyTable().Save(ctx, &dkimv1.DkimPubKey{
			Domain:   dkimPubKey.Domain,
			PubKey:   dkimPubKey.PubKey,
			Selector: dkimPubKey.Selector,
		}); err != nil {
			return nil, err
		}
	}
	return &types.MsgAddDkimPubKeyResponse{}, nil
}

// RemoveDkimPubKey implements types.MsgServer.
func (ms msgServer) RemoveDkimPubKey(ctx context.Context, msg *types.MsgRemoveDkimPubKey) (*types.MsgRemoveDkimPubKeyResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}
	dkimPubKey, err := ms.k.OrmDB.DkimPubKeyTable().Get(ctx, msg.Selector, msg.Domain)
	if err != nil {
		return nil, err
	}
	if err := ms.k.OrmDB.DkimPubKeyTable().Delete(ctx, &dkimv1.DkimPubKey{
		Domain:   dkimPubKey.Domain,
		PubKey:   dkimPubKey.PubKey,
		Selector: dkimPubKey.Selector,
	}); err != nil {
		return nil, err
	}

	return &types.MsgRemoveDkimPubKeyResponse{}, nil
}
