package testlib

import (
	"encoding/json"
	"strconv"

	"github.com/icza/dyno"

	"github.com/cosmos/interchaintest/v11/chain/cosmos"
	"github.com/cosmos/interchaintest/v11/ibc"
)

// ZKFixtureAAAddress is the abstract account every checked-in ZK proof fixture is bound to.
const ZKFixtureAAAddress = "xion12y3uudw8h2a693fnfyrzpnwv674gxhw68p6wqsysu85uc9c6mpsq2u27hz"

// WithZKFixtureAccount reserves accountNumber for ZKFixtureAAAddress in genesis, after modifyGenesis runs.
// The proofs embed SignDocs with a fixed account number, but SDK v0.55 assigns new accounts pseudorandom
// numbers; wasm instantiation keeps an empty BaseAccount, and AA registration keeps its number.
func WithZKFixtureAccount(modifyGenesis func(ibc.ChainConfig, []byte) ([]byte, error), accountNumber uint64) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(cfg ibc.ChainConfig, genesis []byte) ([]byte, error) {
		genesis, err := modifyGenesis(cfg, genesis)
		if err != nil {
			return nil, err
		}
		var state map[string]any
		if err := json.Unmarshal(genesis, &state); err != nil {
			return nil, err
		}
		accounts, err := dyno.GetSlice(state, "app_state", "auth", "accounts")
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, map[string]any{
			"@type":          "/cosmos.auth.v1beta1.BaseAccount",
			"address":        ZKFixtureAAAddress,
			"pub_key":        nil,
			"account_number": strconv.FormatUint(accountNumber, 10),
			"sequence":       "0",
		})
		return cosmos.ModifyGenesis([]cosmos.GenesisKV{
			cosmos.NewGenesisKV("app_state.auth.accounts", accounts),
		})(cfg, genesis)
	}
}
