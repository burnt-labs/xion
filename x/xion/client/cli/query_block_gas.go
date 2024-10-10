package cli

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strconv"
)

func CmdQueryBlockGasUsage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-usage [height]",
		Short: "gas usage",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			var height int64
			heightStr := ""
			if len(args) > 0 {
				heightStr = args[0]
			}

			if heightStr == "" {
				cmd.Println("Falling back to latest block height:")
				height, err = rpc.GetChainHeight(clientCtx)
				if err != nil {
					return fmt.Errorf("failed to get chain height: %w", err)
				}
			} else {
				height, err = strconv.ParseInt(heightStr, 10, 64)
				if err != nil {
					return fmt.Errorf("failed to parse block height: %w", err)
				}
			}

			totalGasUsed := int64(0)
			totalGasWanted := int64(0)
			totalTXs := uint8(0)
			totalRoundTXs := uint8(0)
			fmt.Printf("block: %d\n", height)
			p := message.NewPrinter(language.English)

			node, err := clientCtx.GetNode()
			if err != nil {
				return err
			}
			block, err := node.Block(context.Background(), &height)
			for _, tx := range block.Block.Txs {
				resTx, err := node.Tx(context.Background(), tx.Hash(), false)
				if err != nil {
					return err
				}
				totalGasUsed += resTx.TxResult.GasUsed
				totalGasWanted += resTx.TxResult.GasWanted
				totalTXs += 1
				if resTx.TxResult.GasWanted%1000 == 0 {
					totalRoundTXs += 1
					cmd.Printf("https://explorer.burnt.com/xion-testnet-1/tx/%s %d\n", resTx.Hash.String(), resTx.TxResult.GasWanted)
				}
			}
			gasUseRatio := float32(totalGasUsed) / float32(totalGasWanted)
			output := p.Sprintf("total txs: %d, un-simulated txs: %d\n", totalTXs, totalRoundTXs)
			cmd.Printf(output)
			output = p.Sprintf("total used: %d, total wanted: %d, ratio: %f\n", totalGasUsed, totalGasWanted, gasUseRatio)
			cmd.Printf(output)

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
