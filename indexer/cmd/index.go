package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"cosmossdk.io/log"
	"cosmossdk.io/x/feegrant"
	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/indexer"
	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

const FlagAppDBBackend = "app-db-backend"

func ReIndex(appCreator servertypes.AppCreator, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reindex",
		Short: "re-index",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			homeDir, _ := cmd.Flags().GetString(flags.FlagHome)
			config.SetRoot(homeDir)

			if _, err := os.Stat(config.GenesisFile()); os.IsNotExist(err) {
				return err
			}

			db, err := openDB(config.RootDir, server.GetAppDBBackend(serverCtx.Viper))
			if err != nil {
				return err
			}

			// in our test, it's important to close db explicitly for pebbledb to write to disk.
			defer db.Close()

			logger := log.NewLogger(cmd.OutOrStdout(), log.LevelOption(zerolog.ErrorLevel))

			app := appCreator(logger, db, nil, serverCtx.Viper)
			wasmApp := app.(*xionapp.WasmApp)

			authzHandler := wasmApp.IndexerService().AuthzHandler()

			totalAuthz := 0
			wasmApp.AuthzKeeper.IterateGrants(wasmApp.NewContext(true),
				func(granterAddr sdk.AccAddress, granteeAddr sdk.AccAddress, grant authz.Grant) bool {
					authorization, err := grant.GetAuthorization()
					if err != nil {
						logger.Error("error unpacking authorization", "error", err)
						return true
					}
					msgType := authorization.MsgTypeURL()
					err = authzHandler.SetGrant(wasmApp.NewContext(true), granterAddr, granteeAddr, msgType, grant)
					if err != nil {
						logger.Error("error setting grant", "error", err)
						return true
					}
					totalAuthz++
					if totalAuthz%100000 == 0 {
						slog.Info("granter", "granter", granterAddr.String(), "grantee", granteeAddr.String())

						slog.Info("authz grants imported", "total", totalAuthz)
					}
					if granteeAddr.String() == "xion1t4qcjjseqstaqtsnjxrvahqejcrgyuxp9tf9hv" {
						slog.Info("granter", "msgType", msgType, "granter", granterAddr.String(), "grantee", granteeAddr.String())
					}
					return false
				})

			feeGrantHandler := wasmApp.IndexerService().FeeGrantHandler()
			totalFeeGrant := 0
			addressCodec := wasmApp.AccountKeeper.AddressCodec()
			wasmApp.FeeGrantKeeper.IterateAllFeeAllowances(wasmApp.NewContext(true), func(grant feegrant.Grant) bool {
				granter, err := addressCodec.StringToBytes(grant.Granter)
				if err != nil {
					logger.Error("error parsing granter", "error", err)
					return true
				}
				grantee, err := addressCodec.StringToBytes(grant.Grantee)
				if err != nil {
					logger.Error("error parsing grantee", "error", err)
					return true
				}

				err = feeGrantHandler.SetGrant(wasmApp.NewContext(true), granter, grantee, grant)
				if err != nil {
					logger.Error("error setting grant", "error", err)
					return true
				}
				totalFeeGrant++
				if totalFeeGrant%100000 == 0 {
					slog.Info("fee grants imported", "total", totalFeeGrant)
				}
				return false
			})
			slog.Info("totals", "authz grants", totalAuthz, "fee grants", totalFeeGrant)
			err = wasmApp.Close()
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.Flags().String(FlagAppDBBackend, "", "The type of database for application and snapshots databases")
	return cmd
}

func Query(appCreator servertypes.AppCreator, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reindex",
		Short: "re-index",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			homeDir, _ := cmd.Flags().GetString(flags.FlagHome)
			config.SetRoot(homeDir)

			if _, err := os.Stat(config.GenesisFile()); os.IsNotExist(err) {
				return err
			}

			db, err := openDB(config.RootDir, server.GetAppDBBackend(serverCtx.Viper))
			if err != nil {
				return err
			}

			// in our test, it's important to close db explicitly for pebbledb to write to disk.
			defer db.Close()

			logger := log.NewLogger(cmd.OutOrStdout(), log.LevelOption(zerolog.ErrorLevel))

			app := appCreator(logger, db, nil, serverCtx.Viper)
			wasmApp := app.(*xionapp.WasmApp)

			authzHandler := wasmApp.IndexerService().AuthzHandler()
			querier := indexer.NewAuthzQuerier(authzHandler, wasmApp.AppCodec(), wasmApp.AccountKeeper.AddressCodec())
			querier.GranteeGrants(wasmApp.NewContext(true), &indexerauthz.QueryGranteeGrantsRequest{
				Grantee: "xion1t4qcjjseqstaqtsnjxrvahqejcrgyuxp9tf9hv",
				Pagination: &query.PageRequest{
					Key:        []byte{},
					Limit:      1000,
					CountTotal: true,
				},
			})
			err = wasmApp.Close()
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.Flags().String(FlagAppDBBackend, "", "The type of database for application and snapshots databases")
	return cmd
}

func openDB(rootDir string, backendType dbm.BackendType) (dbm.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return dbm.NewDB("application", backendType, dataDir)
}
