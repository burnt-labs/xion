package app

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtcrypto "github.com/cometbft/cometbft/crypto"
	cmttypes "github.com/cometbft/cometbft/types"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/server"
	srvconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// The random helpers hand every caller a fresh identity; a fixed return would
// silently make tests share accounts, so distinctness is the invariant worth
// pinning.

func TestRandomAccAddress(t *testing.T) {
	addr := RandomAccAddress()
	require.NoError(t, sdk.VerifyAddressFormat(addr))
	require.NotEqual(t, addr, RandomAccAddress())
}

func TestRandomPubKey(t *testing.T) {
	pubKey := RandomPubKey()
	require.IsType(t, &secp256k1.PubKey{}, pubKey)
	require.Len(t, pubKey.Bytes(), secp256k1.PubKeySize)
	require.NoError(t, sdk.VerifyAddressFormat(pubKey.Address().Bytes()))
	require.NotEqual(t, pubKey.Bytes(), RandomPubKey().Bytes())
}

func TestSetupWithBalancesFundsGenesisAccounts(t *testing.T) {
	addr := RandomAccAddress()
	coins := sdk.NewCoins(sdk.NewCoin("uxion", sdkmath.NewInt(42)))

	gapp := SetupWithBalances(t, []banktypes.Balance{{Address: addr.String(), Coins: coins}})
	ctx := gapp.NewContext(false)

	require.NotNil(t, gapp.AccountKeeper.GetAccount(ctx, addr))
	require.Equal(t, coins, gapp.BankKeeper.GetAllBalances(ctx, addr))
}

func TestInitAccountWithCoinsPanicsOnKeeperFailure(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false)

	t.Run("mint rejects invalid coins", func(t *testing.T) {
		invalid := sdk.Coins{sdk.Coin{Denom: "", Amount: sdkmath.OneInt()}}
		require.Panics(t, func() { initAccountWithCoins(gapp, ctx, RandomAccAddress(), invalid) })
	})

	t.Run("send to a blocked address is refused", func(t *testing.T) {
		coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.OneInt()))
		blocked := authtypes.NewModuleAddress(distrtypes.ModuleName)
		require.True(t, gapp.BankKeeper.BlockedAddr(blocked))
		require.Panics(t, func() { initAccountWithCoins(gapp, ctx, blocked, coins) })
	})
}

type fixtureValidator struct {
	ctx *server.Context
	cfg *srvconfig.Config
}

func (v fixtureValidator) GetCtx() *server.Context         { return v.ctx }
func (v fixtureValidator) GetAppConfig() *srvconfig.Config { return v.cfg }

func TestNewTestNetworkFixtureBuildsValidatorApps(t *testing.T) {
	fixture := NewTestNetworkFixture()

	cmtCfg := cmtcfg.DefaultConfig()
	cmtCfg.SetRoot(t.TempDir())
	v := viper.New()
	v.Set(flags.FlagChainID, "fixture-chain")
	validator := fixtureValidator{
		ctx: server.NewContext(v, cmtCfg, log.NewNopLogger()),
		cfg: srvconfig.DefaultConfig(),
	}

	application := fixture.AppConstructor(validator)
	wasmApp, ok := application.(*WasmApp)
	require.True(t, ok)
	require.Equal(t, "fixture-chain", wasmApp.ChainID())
}

// unsupportedPubKey is a consensus key type the SDK cannot convert.
type unsupportedPubKey struct{}

func (unsupportedPubKey) Address() cmtcrypto.Address {
	return cmtcrypto.AddressHash([]byte("unsupported"))
}
func (unsupportedPubKey) Bytes() []byte                    { return []byte("unsupported") }
func (unsupportedPubKey) VerifySignature(_, _ []byte) bool { return false }
func (unsupportedPubKey) Equals(other cmtcrypto.PubKey) bool {
	return bytes.Equal(other.Bytes(), []byte("unsupported"))
}
func (unsupportedPubKey) Type() string { return "unsupported" }

func TestGenesisStateWithValSetRejectsUnsupportedPubKey(t *testing.T) {
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{cmttypes.NewValidator(unsupportedPubKey{}, 1)})

	_, err := GenesisStateWithValSet(MakeEncodingConfig(t).Codec, GenesisState{}, valSet, nil)
	require.ErrorContains(t, err, "failed to convert pubkey")
}
