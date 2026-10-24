package cli_test

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/client/cli"
	"github.com/burnt-labs/xion/x/xion/types"
)

func TestQueryCommandsPrintNodeResponses(t *testing.T) {
	cases := []struct {
		name     string
		newCmd   func() *cobra.Command
		path     string
		response proto.Message
		args     []string
		wantOut  string
	}{
		{
			name:     "platform-percentage",
			newCmd:   cli.CmdPlatformPercentage,
			path:     "/xion.v1.Query/PlatformPercentage",
			response: &types.QueryPlatformPercentageResponse{PlatformPercentage: 1234},
			wantOut:  `"platform_percentage":"1234"`,
		},
		{
			name:     "platform-minimum",
			newCmd:   cli.CmdPlatformMinimum,
			path:     "/xion.v1.Query/PlatformMinimum",
			response: &types.QueryPlatformMinimumResponse{Minimums: sdk.NewCoins(sdk.NewInt64Coin("uxion", 25))},
			wantOut:  `"denom":"uxion"`,
		},
		{
			name:     "webauthn-register",
			newCmd:   cli.CmdWebAuthNVerifyRegister,
			path:     "/xion.v1.Query/WebAuthNVerifyRegister",
			response: &types.QueryWebAuthNVerifyRegisterResponse{Credential: []byte("credential")},
			args:     []string{"addr", "challenge", "https://rp.example", "data"},
			wantOut:  `"credential"`,
		},
		{
			name:     "webauthn-authenticate",
			newCmd:   cli.CmdWebAuthNVerifyAuthenticate,
			path:     "/xion.v1.Query/WebAuthNVerifyAuthenticate",
			response: &types.QueryWebAuthNVerifyAuthenticateResponse{},
			args:     []string{"addr", "challenge", "https://rp.example", "credential", "data"},
			wantOut:  "{}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newCLITestEnv(t)
			rpc := routedCometRPC{responses: map[string]abci.ResponseQuery{tc.path: queryResponse(t, tc.response)}}
			clientCtx := env.clientContext(rpc).WithOutputFormat(flags.OutputFormatJSON)

			err := runCmd(t, tc.newCmd(), clientCtx, tc.args...)
			require.NoError(t, err)
			require.Contains(t, env.out.String(), tc.wantOut)
		})
	}
}

func TestQueryCommandsRejectMalformedNodeFlag(t *testing.T) {
	cases := []struct {
		name   string
		newCmd func() *cobra.Command
		args   []string
	}{
		{"platform-percentage", cli.CmdPlatformPercentage, nil},
		{"platform-minimum", cli.CmdPlatformMinimum, nil},
		{"webauthn-register", cli.CmdWebAuthNVerifyRegister, []string{"addr", "challenge", "rp", "data"}},
		{"webauthn-authenticate", cli.CmdWebAuthNVerifyAuthenticate, []string{"addr", "challenge", "rp", "credential", "data"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newCLITestEnv(t)
			args := append(tc.args, fmt.Sprintf("--%s=://no-scheme", flags.FlagNode))
			err := runCmd(t, tc.newCmd(), env.clientContext(routedCometRPC{}), args...)
			require.ErrorContains(t, err, "missing protocol scheme")
		})
	}
}
