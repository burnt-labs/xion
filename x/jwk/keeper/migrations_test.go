package keeper_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestMigrate3To4(t *testing.T) {
	tests := []struct {
		name   string
		legacy map[string]string
		want   types.Params
		err    string
	}{
		{
			name: "mainnet params",
			legacy: map[string]string{
				"jwk/TimeOffset":    `"30000000000"`,
				"jwk/DeploymentGas": `"10000"`,
			},
			want: types.NewParams(30_000_000_000, 10_000),
		},
		{
			name: "custom params",
			legacy: map[string]string{
				"jwk/TimeOffset":    `"45000000000"`,
				"jwk/DeploymentGas": `"20000"`,
			},
			want: types.NewParams(45_000_000_000, 20_000),
		},
		{
			name: "missing keys",
			want: types.DefaultParams(),
		},
		{
			name:   "missing time offset",
			legacy: map[string]string{"jwk/DeploymentGas": `"20000"`},
			want:   types.NewParams(types.DefaultParams().TimeOffset, 20_000),
		},
		{
			name:   "missing deployment gas",
			legacy: map[string]string{"jwk/TimeOffset": `"45000000000"`},
			want:   types.NewParams(45_000_000_000, types.DefaultParams().DeploymentGas),
		},
		{
			name:   "malformed time offset",
			legacy: map[string]string{"jwk/TimeOffset": `"invalid"`},
			err:    "decode jwk/TimeOffset",
		},
		{
			name: "malformed deployment gas",
			legacy: map[string]string{
				"jwk/TimeOffset":    `"45000000000"`,
				"jwk/DeploymentGas": `invalid`,
			},
			err: "decode jwk/DeploymentGas",
		},
		{
			name:   "invalid deployment gas",
			legacy: map[string]string{"jwk/DeploymentGas": `"0"`},
			err:    "deployment gas must be positive",
		},
		{
			name:   "invalid time offset",
			legacy: map[string]string{"jwk/TimeOffset": `"0"`},
			err:    "time offset must be positive",
		},
		{
			name:   "time offset overflows int64",
			legacy: map[string]string{"jwk/TimeOffset": `"9223372036854775808"`},
			err:    "time offset exceeds max int64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storeKey := storetypes.NewKVStoreKey(types.StoreKey)
			legacyKey := storetypes.NewKVStoreKey("params")
			ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{
				types.StoreKey: storeKey,
				"params":       legacyKey,
			}, nil, nil)
			cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
			k := keeper.NewKeeper(cdc, storeKey, authtypes.NewModuleAddress(govtypes.ModuleName).String())
			before := types.NewParams(math.MaxInt64, 123)
			require.NoError(t, k.SetParams(ctx, before))
			for key, value := range tt.legacy {
				ctx.KVStore(legacyKey).Set([]byte(key), []byte(value))
			}

			m := keeper.NewMigrator(k, runtime.NewKVStoreService(legacyKey), codec.NewLegacyAmino())
			err := m.Migrate3To4(ctx)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				require.Equal(t, before, k.GetParams(ctx))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, k.GetParams(ctx))
		})
	}
}
