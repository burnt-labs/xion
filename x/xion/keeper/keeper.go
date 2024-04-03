package keeper

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/burnt-labs/xion/x/xion/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	paramSpace    paramtypes.Subspace
	bankKeeper    types.BankKeeper
	accountKeeper types.AccountKeeper
	wasmKeeper    types.WasmKeeper
	authzKeeper   types.AuthzKeeper

	// the address capable of executing a MsgSetPlatformPercentage message.
	// Typically, this should be the x/gov module account
	authority string
}

func NewKeeper(cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	paramSpace paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	wasmKeeper types.WasmKeeper,
	authzKeeper types.AuthzKeeper,
	authority string) Keeper {

	return Keeper{
		storeKey:      key,
		cdc:           cdc,
		paramSpace:    paramSpace,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
		wasmKeeper:    wasmKeeper,
		authzKeeper:   authzKeeper,
		authority:     authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdktypes.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// Platform Percentage

func (k Keeper) GetPlatformPercentage(ctx sdktypes.Context) math.Int {
	bz := ctx.KVStore(k.storeKey).Get(types.PlatformPercentageKey)
	percentage := sdktypes.BigEndianToUint64(bz)
	return math.NewIntFromUint64(percentage)
}

func (k Keeper) OverwritePlatformPercentage(ctx sdktypes.Context, percentage uint32) {
	ctx.KVStore(k.storeKey).Set(types.PlatformPercentageKey, sdktypes.Uint64ToBigEndian(uint64(percentage)))
}

// Authority

// GetAuthority returns the x/xion module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// DispatchActions attempts to execute the provided messages via authorization
// grants from the message signer to the grantee.
func (k Keeper) DispatchActions(ctx sdk.Context, grantee sdk.AccAddress, msgs []sdk.Msg) ([][]byte, error) {

	now := ctx.BlockTime()
	for _, msg := range msgs {
		execMsg, ok := msg.(wasmtypes.AuthzableWasmMsg)
		if !ok {
			return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "unsupported message type: %T", msg)
		}

		signers := msg.GetSigners()
		if len(signers) != 1 {
			return nil, authztypes.ErrAuthorizationNumOfSigners
		}

		granter := signers[0]

		if !granter.Equals(grantee) {
			// authz doesn't expose a method to get a grant from the store so use the query method instead
			grants, err := k.authzKeeper.Grants(ctx, &authztypes.QueryGrantsRequest{
				Granter:    granter.String(),
				Grantee:    grantee.String(),
				MsgTypeUrl: sdk.MsgTypeURL(msg),
			})
			if err != nil {
				return nil, err
			}

			grant := grants.Grants[0]
			if grant.Expiration != nil && grant.Expiration.Before(now) {
				return nil, authztypes.ErrAuthorizationExpired
			}

			authorization, err := grant.GetAuthorization()
			if err != nil {
				return nil, err
			}
			xionAuth, ok := authorization.(*types.CodeIdExecutionAuthorization)
			if ok {
				isAuthorized := false
				// check contract is instantiated from the grant code id
				contractInfo := k.wasmKeeper.GetContractInfo(ctx, sdktypes.AccAddress(execMsg.GetContract()))

				for _, g := range xionAuth.Grants {
					if g.CodeId == contractInfo.CodeID {
						// contract is instantiated from the grant code id
						isAuthorized = true
						break
					}
				}
				if !isAuthorized {
					return nil, authztypes.ErrNoAuthorizationFound
				}
			}
		}
	}
	return k.authzKeeper.DispatchActions(ctx, grantee, msgs)
}
