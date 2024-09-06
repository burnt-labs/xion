package app

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v10"

func (app *WasmApp) RegisterUpgradeHandlers() {
	app.WrapSetUpgradeHandler(UpgradeName)
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}
	app.Logger().Info("upgrade info", "name", upgradeInfo.Name, "height", upgradeInfo.Height)

	if upgradeInfo.Name == UpgradeName {
		if !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
			app.Logger().Info("setting upgrade store loaders")
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storetypes.StoreUpgrades{}))
		}
	}
}

func (app *WasmApp) WrapSetUpgradeHandler(upgradeName string) {
	// ensure legacy params exist here -- query legacy paramspace
	// ensure self-managed params exist -- query module self-managed
	// direct params manipulation -- set default params in module until migration runs?

	app.Logger().Info("setting upgrade handler", "name", upgradeName)
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName, func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		sdkCtx.Logger().Info("running module migrations", "name", plan.Name)
		vm, err = app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
		if err != nil {
			return vm, err
		}

		return vm, err
	})
}

/*
> only panic on unset params and not on empty params
https://github.com/cosmos/ibc-go/blob/4cea99ec8eea14f6db81b5386e2ff0b2561163e0/modules/core/02-client/keeper/keeper.go#L431
*/

/*
panic: client params are not set in store

goroutine 1 [running]:
github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper.Keeper.GetParams({{_, _}, {_, _}, {_, _}, {_, _}, {_, _}, ...}, ...)
	github.com/cosmos/ibc-go/v8@v8.4.0/modules/core/02-client/keeper/keeper.go:347 +0x188
github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper.Keeper.GetClientStatus({{_, _}, {_, _}, {_, _}, {_, _}, {_, _}, ...}, ...)
	github.com/cosmos/ibc-go/v8@v8.4.0/modules/core/02-client/keeper/keeper.go:336 +0xb8
github.com/cosmos/ibc-go/v8/modules/core/02-client.BeginBlocker({{0x4eb5738, 0x7284c60}, {0x4ed3910, 0x4001d79600}, {{0x0, 0x0}, {0x4001d4a904, 0x6}, 0x48, {0x71ca54f, ...}, ...}, ...}, ...)
	github.com/cosmos/ibc-go/v8@v8.4.0/modules/core/02-client/abci.go:38 +0x588
github.com/cosmos/ibc-go/v8/modules/core.AppModule.BeginBlock({{}, 0x358?}, {0x4eb5770?, 0x4001bda388?})
	github.com/cosmos/ibc-go/v8@v8.4.0/modules/core/module.go:191 +0xd8
github.com/cosmos/cosmos-sdk/types/module.(*Manager).BeginBlock(_, {{0x4eb5738, 0x7284c60}, {0x4ed3910, 0x4001d79600}, {{0x0, 0x0}, {0x4001d4a904, 0x6}, 0x48, ...}, ...})
	github.com/cosmos/cosmos-sdk@v0.50.8/types/module/module.go:779 +0x14c
github.com/burnt-labs/xion/app.(*WasmApp).BeginBlocker(...)
	github.com/burnt-labs/xion/app/app.go:1107
github.com/cosmos/cosmos-sdk/baseapp.(*BaseApp).beginBlock(0x4001df8488, 0x7284c60?)
	github.com/cosmos/cosmos-sdk@v0.50.8/baseapp/baseapp.go:734 +0xa4
github.com/cosmos/cosmos-sdk/baseapp.(*BaseApp).internalFinalizeBlock(0x4001df8488, {0x4eb5738, 0x7284c60}, 0x4000951c80)
	github.com/cosmos/cosmos-sdk@v0.50.8/baseapp/abci.go:760 +0x9cc
github.com/cosmos/cosmos-sdk/baseapp.(*BaseApp).FinalizeBlock(0x4001df8488, 0x4000951c80)
	github.com/cosmos/cosmos-sdk@v0.50.8/baseapp/abci.go:884 +0x120
github.com/cosmos/cosmos-sdk/server.cometABCIWrapper.FinalizeBlock(...)
	github.com/cosmos/cosmos-sdk@v0.50.8/server/cmt_abci.go:44
github.com/cometbft/cometbft/abci/client.(*localClient).FinalizeBlock(0x4ed6980?, {0x4eb6228?, 0x7284c60?}, 0xffff5206a1a0?)
	github.com/cometbft/cometbft@v0.38.9/abci/client/local_client.go:185 +0xe8
github.com/cometbft/cometbft/proxy.(*appConnConsensus).FinalizeBlock(0x4003a1c7f8, {0x4eb6228, 0x7284c60}, 0x4000951c80)
	github.com/cometbft/cometbft@v0.38.9/proxy/app_conn.go:104 +0x130
github.com/cometbft/cometbft/state.(*BlockExecutor).applyBlock(_, {{{0xb, 0x0}, {0x4003a5a460, 0x6}}, {0x4003a5a466, 0x6}, 0x1, 0x47, {{0x4001bcc320, ...}, ...}, ...}, ...)
	github.com/cometbft/cometbft@v0.38.9/state/execution.go:224 +0x410
github.com/cometbft/cometbft/state.(*BlockExecutor).ApplyBlock(_, {{{0xb, 0x0}, {0x4003a5a460, 0x6}}, {0x4003a5a466, 0x6}, 0x1, 0x47, {{0x4001bcc320, ...}, ...}, ...}, ...)
	github.com/cometbft/cometbft@v0.38.9/state/execution.go:219 +0x140
github.com/cometbft/cometbft/consensus.(*Handshaker).replayBlock(_, {{{0xb, 0x0}, {0x4003a5a460, 0x6}}, {0x4003a5a466, 0x6}, 0x1, 0x47, {{0x4001bcc320, ...}, ...}, ...}, ...)
	github.com/cometbft/cometbft@v0.38.9/consensus/replay.go:534 +0x1a8
github.com/cometbft/cometbft/consensus.(*Handshaker).ReplayBlocksWithContext(_, {_, _}, {{{0xb, 0x0}, {0x4003a5a460, 0x6}}, {0x4003a5a466, 0x6}, 0x1, ...}, ...)
	github.com/cometbft/cometbft@v0.38.9/consensus/replay.go:433 +0x5dc
github.com/cometbft/cometbft/consensus.(*Handshaker).HandshakeWithContext(0x4001bfb390, {0x4eb6180, 0x4003a7ca50}, {0x4ed8070, 0x40036e40e0})
	github.com/cometbft/cometbft@v0.38.9/consensus/replay.go:274 +0x370
github.com/cometbft/cometbft/node.doHandshake({_, _}, {_, _}, {{{0xb, 0x0}, {0x4003a5a460, 0x6}}, {0x4003a5a466, 0x6}, ...}, ...)
	github.com/cometbft/cometbft@v0.38.9/node/setup.go:182 +0x12c
github.com/cometbft/cometbft/node.NewNodeWithContext({0x4eb6180, 0x4003a7ca50}, 0x400158eb40, {0x4e92c10, 0x4001b5f9a0}, 0x4003c82100, {0x4e76a20, 0x4003c84318}, 0x4001bfc210, 0x4748208, ...)
	github.com/cometbft/cometbft@v0.38.9/node/node.go:359 +0x42c
github.com/cosmos/cosmos-sdk/server.startCmtNode({0x4eb6180, 0x4003a7ca50}, 0x400158eb40, {0x4efc8c8, 0x4000304c88}, 0x4001ad5b20)
	github.com/cosmos/cosmos-sdk@v0.50.8/server/start.go:377 +0x308
github.com/cosmos/cosmos-sdk/server.startInProcess(_, {{{0x4001c078b8, 0x8}, 0x0, {0x4001c07ea0, 0x7}, {0x3e7ca15, 0x1}, {0x3e7ca15, 0x1}, ...}, ...}, ...)
	github.com/cosmos/cosmos-sdk@v0.50.8/server/start.go:323 +0x11c
github.com/cosmos/cosmos-sdk/server.start(_, {{0x0, 0x0, 0x0}, {0x4edc2d8, 0x4001b45080}, 0x0, {0x0, 0x0}, {0x4efc990, ...}, ...}, ...)
	github.com/cosmos/cosmos-sdk@v0.50.8/server/start.go:240 +0x1d8
github.com/cosmos/cosmos-sdk/server.StartCmdWithOptions.func2.1()
	github.com/cosmos/cosmos-sdk@v0.50.8/server/start.go:198 +0x68
github.com/cosmos/cosmos-sdk/server.wrapCPUProfile(0x4001ad5b20, 0x40015bf9c8)
	github.com/cosmos/cosmos-sdk@v0.50.8/server/start.go:570 +0x16c
github.com/cosmos/cosmos-sdk/server.StartCmdWithOptions.func2(0x4001cea308, {0x4001b448a0?, 0x0?, 0x3?})
	github.com/cosmos/cosmos-sdk@v0.50.8/server/start.go:197 +0x184
github.com/spf13/cobra.(*Command).execute(0x4001cea308, {0x4001b44810, 0x3, 0x3})
	github.com/spf13/cobra@v1.8.0/command.go:983 +0x840
github.com/spf13/cobra.(*Command).ExecuteC(0x400142a908)
	github.com/spf13/cobra@v1.8.0/command.go:1115 +0x344
github.com/spf13/cobra.(*Command).Execute(...)
	github.com/spf13/cobra@v1.8.0/command.go:1039
github.com/spf13/cobra.(*Command).ExecuteContext(...)
	github.com/spf13/cobra@v1.8.0/command.go:1032
github.com/cosmos/cosmos-sdk/server/cmd.Execute(0x400142a908, {0x0, 0x0}, {0x4001209488, 0x17})
	github.com/cosmos/cosmos-sdk@v0.50.8/server/cmd/execute.go:34 +0x154
main.main()
	github.com/burnt-labs/xion/cmd/xiond/main.go:16 +0x3c
*/
