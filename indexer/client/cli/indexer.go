package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/burnt-labs/xion/x/authz"

	xionapp "github.com/burnt-labs/xion/app"
	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
	indexerfeegrant "github.com/burnt-labs/xion/indexer/feegrant"
)

const FlagAppDBBackend = "app-db-backend"

func Indexer(appCreator servertypes.AppCreator, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "indexer",
		Short: "indexer",
		Long: `
		Indexer support for xion. Indexing Authz and FeeGrant grants and allowances with streaming support.
		`,

		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(ReIndex(appCreator, defaultNodeHome))
	cmd.AddCommand(QueryGrantsByGrantee())
	cmd.AddCommand(QueryGrantsByGranter())
	cmd.AddCommand(QueryAllowancesByGrantee())
	cmd.AddCommand(QueryAllowancesByGranter())
	return cmd
}

func ReIndex(appCreator servertypes.AppCreator, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "re-index",
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
					return false
				})

			feeGrantHandler := wasmApp.IndexerService().FeeGrantHandler()
			totalFeeGrant := 0
			addressCodec := wasmApp.AccountKeeper.AddressCodec()
			if err := wasmApp.FeeGrantKeeper.IterateAllFeeAllowances(wasmApp.NewContext(true), func(grant feegrant.Grant) bool {
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
				return false
			}); err != nil {
				logger.Error("error iterating fee allowances", "error", err)
				return err
			}
			slog.Info("totals", "authz grants", totalAuthz, "fee grants", totalFeeGrant)
			// close to flush the db
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

func QueryGrantsByGrantee() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-grants-by-grantee [grantee]",
		Short: "query authz grants by grantee",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grantee := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := indexerauthz.NewQueryClient(clientCtx)
			res, err := queryClient.GranteeGrants(cmd.Context(), &indexerauthz.QueryGranteeGrantsRequest{
				Grantee: grantee,
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func QueryGrantsByGranter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-grants-by-granter [granter]",
		Short: "query grants by granter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			granter := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := indexerauthz.NewQueryClient(clientCtx)
			res, err := queryClient.GranterGrants(cmd.Context(), &indexerauthz.QueryGranterGrantsRequest{
				Granter: granter,
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func QueryAllowancesByGrantee() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-allowances-by-grantee [grantee]",
		Short: "query fee grant allowances by grantee",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grantee := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := indexerfeegrant.NewQueryClient(clientCtx)
			res, err := queryClient.Allowances(cmd.Context(), &indexerfeegrant.QueryAllowancesRequest{
				Grantee: grantee,
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func QueryAllowancesByGranter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-allowances-by-granter [granter]",
		Short: "query allowances by granter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			granter := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := indexerfeegrant.NewQueryClient(clientCtx)
			res, err := queryClient.AllowancesByGranter(cmd.Context(), &indexerfeegrant.QueryAllowancesByGranterRequest{
				Granter: granter,
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func openDB(rootDir string, backendType dbm.BackendType) (dbm.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return dbm.NewDB("application", backendType, dataDir)
}
