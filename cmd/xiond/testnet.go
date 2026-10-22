package main

import (
	"fmt"
	"io"

	cmtcrypto "github.com/cometbft/cometbft/crypto"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"

	"github.com/burnt-labs/xion/app"
)

// newTestnetApp creates the normal application, then applies the state changes
// requested exclusively by the in-place-testnet command.
func newTestnetApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	wasmApp, ok := newApp(logger, db, traceStore, appOpts).(*app.WasmApp)
	if !ok {
		panic("newApp did not return *app.WasmApp")
	}

	pubKey, ok := appOpts.Get(server.KeyUserPubKey).(cmtcrypto.PubKey)
	if !ok || pubKey == nil {
		panic(fmt.Sprintf("testnet app option %q must be a CometBFT public key", server.KeyUserPubKey))
	}

	operator, ok := appOpts.Get(server.KeyNewOpAddr).(string)
	if !ok {
		panic(fmt.Sprintf("testnet app option %q must be a string", server.KeyNewOpAddr))
	}

	chainID, ok := appOpts.Get(server.KeyNewChainID).(string)
	if !ok {
		panic(fmt.Sprintf("testnet app option %q must be a string", server.KeyNewChainID))
	}

	upgradeName, ok := appOpts.Get(server.KeyTriggerTestnetUpgrade).(string)
	if !ok {
		panic(fmt.Sprintf("testnet app option %q must be a string", server.KeyTriggerTestnetUpgrade))
	}

	if err := wasmApp.InitForTestnet(pubKey, operator, chainID, upgradeName); err != nil {
		panic(fmt.Errorf("initialize application for in-place testnet: %w", err))
	}

	return wasmApp
}
