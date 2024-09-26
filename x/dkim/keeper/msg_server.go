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

func SaveDkimPubKey(ctx context.Context, dkimKey *types.DkimPubKey, store dkimv1.StateStore) (bool, error) {
	if err := store.DkimPubKeyTable().Save(ctx, &dkimv1.DkimPubKey{
		Domain:   dkimKey.Domain,
		PubKey:   dkimKey.PubKey,
		Selector: dkimKey.Selector,
		Version:  dkimv1.Version(dkimKey.Version),
		KeyType:  dkimv1.KeyType(dkimKey.KeyType),
	}); err != nil {
		return false, err
	}

	return true, nil
}

func SaveDkimPubKeys(ctx context.Context, dkimKeys []types.DkimPubKey, store dkimv1.StateStore) (bool, error) {
	for _, dkimKey := range dkimKeys {
		if isSaved, err := SaveDkimPubKey(ctx, &dkimKey, store); !isSaved {
			return false, err
		}
	}
	return true, nil
}

// AddDkimPubKey implements types.MsgServer.
func (ms msgServer) AddDkimPubKey(ctx context.Context, msg *types.MsgAddDkimPubKey) (*types.MsgAddDkimPubKeyResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}
	for _, dkimKey := range msg.DkimPubkeys {
		if err := dkimKey.Validate(); err != nil {
			return nil, err
		}
	}
	SaveDkimPubKeys(ctx, msg.DkimPubkeys, ms.k.OrmDB)
	return &types.MsgAddDkimPubKeyResponse{}, nil
}

// RemoveDkimPubKey implements types.MsgServer.
func (ms msgServer) RemoveDkimPubKey(ctx context.Context, msg *types.MsgRemoveDkimPubKey) (*types.MsgRemoveDkimPubKeyResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}
	dkimPubKey, err := ms.k.OrmDB.DkimPubKeyTable().Get(ctx, msg.Domain, msg.Selector)
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
