package testing

import (
	"encoding/json"
	"os"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/abstractaccount/simapp"
)

const DefaultBondDenom = "utoken"

var DefaultConsensusParams = &tmproto.ConsensusParams{
	Block: &tmproto.BlockParams{
		MaxBytes: 200000,
		MaxGas:   10000000,
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

func MakeSimpleMockApp() *simapp.SimApp {
	app, _ := MakeMockAppWithCleanup([]banktypes.Balance{})
	return app
}

func MakeMockApp(balances []banktypes.Balance) *simapp.SimApp {
	app, _ := MakeMockAppWithCleanup(balances)
	return app
}

// MakeMockAppWithCleanup creates a mock app and returns a cleanup function
// Use this in tests where you want to properly clean up the temporary directory
func MakeMockAppWithCleanup(balances []banktypes.Balance) (*simapp.SimApp, func()) {
	encCfg := simapp.MakeEncodingConfig()

	// Create a unique temporary directory for each test to avoid WASM exclusive.lock conflicts
	tempDir, err := os.MkdirTemp("", "abstract-account-test-*")
	if err != nil {
		panic(err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	app := simapp.NewSimApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		NewTestAppOptions(tempDir),
		[]wasmkeeper.Option{},
	)

	gs := MakeMockGenesisState(encCfg.Codec, balances)
	gsBytes, err := json.Marshal(gs)
	if err != nil {
		cleanup()
		panic(err)
	}

	//nolint: errcheck // (validators are empty)
	app.InitChain(
		&abci.RequestInitChain{
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: DefaultConsensusParams,
			AppStateBytes:   gsBytes,
		},
	)

	return app, cleanup
}

func MakeMockGenesisState(cdc codec.JSONCodec, balances []banktypes.Balance) simapp.GenesisState {
	gs := simapp.DefaultGenesisState(cdc)

	// prepare genesis accounts
	genAccts := []authtypes.GenesisAccount{}
	for _, balance := range balances {
		addr, err := sdk.AccAddressFromBech32(balance.Address)
		if err != nil {
			panic(err)
		}

		genAccts = append(genAccts, authtypes.NewBaseAccountWithAddress(addr))
	}

	// compute total supply of tokens
	supply := sdk.NewCoins()
	for _, balance := range balances {
		supply = supply.Add(balance.Coins...)
	}

	gs[authtypes.ModuleName] = cdc.MustMarshalJSON(authtypes.NewGenesisState(
		authtypes.DefaultParams(),
		genAccts,
	))

	gs[banktypes.ModuleName] = cdc.MustMarshalJSON(banktypes.NewGenesisState(
		banktypes.DefaultParams(),
		balances,
		supply,
		[]banktypes.Metadata{},
		[]banktypes.SendEnabled{},
	))

	return gs
}

// ----------------------------------- Keys ------------------------------------

func MakeRandomAddress() sdk.AccAddress {
	return sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
}

func MakeRandomPubKey() cryptotypes.PubKey {
	return secp256k1.GenPrivKey().PubKey()
}

// -------------------------------- AppOptions ---------------------------------

type TestAppOptions struct {
	homeDir string
}

func NewTestAppOptions(homeDir string) *TestAppOptions {
	return &TestAppOptions{homeDir: homeDir}
}

func (opts *TestAppOptions) Get(key string) interface{} {
	switch key {
	case flags.FlagHome:
		return opts.homeDir
	default:
		return nil
	}
}

type EmptyAppOptions struct{}

func (opts EmptyAppOptions) Get(_ string) interface{} {
	return nil
}
