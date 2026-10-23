package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	aatypes "github.com/burnt-labs/xion/x/abstractaccount/types"
	"github.com/burnt-labs/xion/x/xion/client/cli"
)

const (
	accountAddressQueryPath = "/abstractaccount.v1.Query/AccountAddress"
	authAccountQueryPath    = "/cosmos.auth.v1beta1.Query/Account"
)

var skipConfirm = fmt.Sprintf("--%s", flags.FlagSkipConfirmation)

func TestSendTxCmdRejectsBadInput(t *testing.T) {
	env := newCLITestEnv(t)
	clientCtx := env.clientContext(routedCometRPC{})
	to := env.account.Address.String()

	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"from key not in keyring", []string{"nobody", to, "10stake"}, "key not found"},
		{"malformed amount", []string{env.account.Name, to, "ten stake"}, "invalid decimal coin expression"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCmd(t, cli.NewSendTxCmd(), clientCtx, append(tc.args, skipConfirm)...)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestMultiSendTxCmdSplitAndBadAmount(t *testing.T) {
	env := newCLITestEnv(t)
	clientCtx := env.clientContext(routedCometRPC{})
	to := env.account.Address.String()

	t.Run("malformed amount", func(t *testing.T) {
		err := runCmd(t, cli.NewMultiSendTxCmd(), clientCtx, env.account.Name, to, to, "ten stake", skipConfirm)
		require.ErrorContains(t, err, "invalid decimal coin expression")
	})

	t.Run("split amount across recipients", func(t *testing.T) {
		err := runCmd(t, cli.NewMultiSendTxCmd(), clientCtx,
			env.account.Name, to, to, "10stake", fmt.Sprintf("--%s", cli.FlagSplit), skipConfirm)
		require.NoError(t, err)
	})
}

func TestRegisterCmdWithPredictedAddress(t *testing.T) {
	env := newCLITestEnv(t)
	predicted := sdk.AccAddress(bytes.Repeat([]byte{0x07}, 32)).String()
	rpc := routedCometRPC{responses: map[string]abci.ResponseQuery{
		accountAddressQueryPath: queryResponse(t, &aatypes.QueryAccountAddressResponse{Address: predicted}),
	}}
	clientCtx := env.clientContext(rpc)

	t.Run("secp256k1 authenticator broadcasts", func(t *testing.T) {
		err := runCmd(t, cli.NewRegisterCmd(), clientCtx,
			"7", env.account.Name,
			"--salt=my-salt", "--authenticator=Secp256K1", "--authenticator-id=1", "--funds=10stake", skipConfirm)
		require.NoError(t, err)
	})

	t.Run("jwt authenticator broadcasts", func(t *testing.T) {
		err := runCmd(t, cli.NewRegisterCmd(), clientCtx,
			"7", env.account.Name,
			"--salt=my-salt", "--authenticator=Jwt", "--sub=user-1", "--aud=app", "--token=header.payload.sig", skipConfirm)
		require.NoError(t, err)
	})

	t.Run("zero code id fails message validation", func(t *testing.T) {
		err := runCmd(t, cli.NewRegisterCmd(), clientCtx,
			"0", env.account.Name, "--salt=my-salt", "--authenticator=Secp256K1", skipConfirm)
		require.ErrorContains(t, err, "code id cannot be zero")
	})

	t.Run("address outside the keyring cannot sign", func(t *testing.T) {
		stranger := sdk.AccAddress(bytes.Repeat([]byte{0x09}, 20)).String()
		err := runCmd(t, cli.NewRegisterCmd(), clientCtx,
			"7", stranger, "--salt=my-salt", "--authenticator=Secp256K1", fmt.Sprintf("--%s", flags.FlagGenerateOnly))
		require.ErrorContains(t, err, "error signing predicted address")
	})
}

// writeSignableTx stores an unsigned bank send as the JSON the sign command reads.
func writeSignableTx(t *testing.T, env cliTestEnv) string {
	t.Helper()
	builder := env.encCfg.TxConfig.NewTxBuilder()
	msg := banktypes.NewMsgSend(env.account.Address, env.account.Address, sdk.NewCoins(sdk.NewInt64Coin("stake", 1)))
	require.NoError(t, builder.SetMsgs(msg))
	bz, err := env.encCfg.TxConfig.TxJSONEncoder()(builder.GetTx())
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "tx.json")
	require.NoError(t, os.WriteFile(path, bz, 0o600))
	return path
}

func abstractAccountResponse(t *testing.T, addr sdk.AccAddress) abci.ResponseQuery {
	t.Helper()
	acc := aatypes.NewAbstractAccount(addr.String(), 3, 5)
	packed, err := codectypes.NewAnyWithValue(acc)
	require.NoError(t, err)
	return queryResponse(t, &authtypes.QueryAccountResponse{Account: packed})
}

func TestSignCmdSignsForAbstractAccount(t *testing.T) {
	env := newCLITestEnv(t)
	signer := sdk.AccAddress(bytes.Repeat([]byte{0x0a}, 32))
	responses := map[string]abci.ResponseQuery{
		authAccountQueryPath: abstractAccountResponse(t, signer),
	}
	txFile := writeSignableTx(t, env)

	t.Run("signs and broadcasts", func(t *testing.T) {
		clientCtx := env.clientContext(routedCometRPC{responses: responses})
		err := runCmd(t, cli.NewSignCmd(), clientCtx, env.account.Name, signer.String(), txFile, "--authenticator-id=1")
		require.NoError(t, err)
		require.Contains(t, env.out.String(), "txhash")
	})

	t.Run("broadcast failure panics", func(t *testing.T) {
		clientCtx := env.clientContext(routedCometRPC{responses: responses, broadcastErr: errors.New("node unavailable")})
		err := capturePanic(t, func() {
			_ = runCmd(t, cli.NewSignCmd(), clientCtx, env.account.Name, signer.String(), txFile)
		})
		require.ErrorContains(t, err, "node unavailable")
	})

	t.Run("missing signing key panics", func(t *testing.T) {
		// --generate-only resolves a bech32 address without consulting the
		// keyring, so the later keyring signature has no key to use.
		stranger := sdk.AccAddress(bytes.Repeat([]byte{0x0b}, 20)).String()
		clientCtx := env.clientContext(routedCometRPC{responses: responses})
		err := capturePanic(t, func() {
			_ = runCmd(t, cli.NewSignCmd(), clientCtx, stranger, signer.String(), txFile, fmt.Sprintf("--%s", flags.FlagGenerateOnly))
		})
		require.ErrorContains(t, err, "no key name or address provided")
	})
}

func TestAddAuthenticatorCmdRejectsBadContract(t *testing.T) {
	env := newCLITestEnv(t)
	clientCtx := env.clientContext(routedCometRPC{})
	from := fmt.Sprintf("--%s=%s", flags.FlagFrom, env.account.Name)

	err := runCmd(t, cli.NewAddAuthenticatorCmd(), clientCtx, "not-a-contract", from, "--authenticator-id=1", skipConfirm)
	require.ErrorContains(t, err, "sender")
}

func TestEmitArbitraryDataCmdPaths(t *testing.T) {
	env := newCLITestEnv(t)
	clientCtx := env.clientContext(routedCometRPC{})
	contract := env.account.Address.String()

	t.Run("from key not in keyring", func(t *testing.T) {
		err := runCmd(t, cli.NewEmitArbitraryDataCmd(), clientCtx, "hello", contract, fmt.Sprintf("--%s=nobody", flags.FlagFrom))
		require.ErrorContains(t, err, "key not found")
	})

	t.Run("broadcasts emit message", func(t *testing.T) {
		err := runCmd(t, cli.NewEmitArbitraryDataCmd(), clientCtx,
			"hello", contract, fmt.Sprintf("--%s=%s", flags.FlagFrom, env.account.Name), skipConfirm)
		require.NoError(t, err)
	})
}

func TestUpdateConfigsCmdSourceErrors(t *testing.T) {
	env := newCLITestEnv(t)
	clientCtx := env.clientContext(routedCometRPC{})
	contract := env.account.Address.String()
	from := fmt.Sprintf("--%s=%s", flags.FlagFrom, env.account.Name)

	writeConfig := func(t *testing.T, name, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		return path
	}
	serve := func(t *testing.T, handler http.HandlerFunc) string {
		t.Helper()
		srv := httptest.NewServer(handler)
		t.Cleanup(srv.Close)
		return srv.URL + "/config.json"
	}

	validFee := `"fee_config": {"description": "fee", "allowance": {"@type": "/cosmos.feegrant.v1beta1.BasicAllowance"}}`
	grantWith := func(authorization string) string {
		return `{"grant_config": [{"msg_type_url": "/cosmos.bank.v1beta1.MsgSend", "grant_config": {"description": "d", "authorization": ` + authorization + `}}], ` + validFee + `}`
	}

	cases := []struct {
		name    string
		source  func(t *testing.T) string
		local   bool
		args    []string
		wantErr string
	}{
		{
			name:    "from key not in keyring",
			source:  func(*testing.T) string { return "config.json" },
			local:   true,
			args:    []string{fmt.Sprintf("--%s=nobody", flags.FlagFrom)},
			wantErr: "key not found",
		},
		{
			name:    "local file missing",
			source:  func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.json") },
			local:   true,
			wantErr: "failed to read configuration file",
		},
		{
			name:    "local file is not JSON",
			source:  func(t *testing.T) string { return writeConfig(t, "bad.json", "{not json") },
			local:   true,
			wantErr: "failed to unmarshal local configuration file",
		},
		{
			name: "server drops the connection",
			source: func(t *testing.T) string {
				return serve(t, func(w http.ResponseWriter, _ *http.Request) {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						_ = conn.Close()
					}
				})
			},
			wantErr: "failed to fetch configuration from URI",
		},
		{
			name: "server answers non-200",
			source: func(t *testing.T) string {
				return serve(t, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: "received non-200 response code: 500",
		},
		{
			name: "server body is not JSON",
			source: func(t *testing.T) string {
				return serve(t, func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte("<html>"))
				})
			},
			wantErr: "failed to decode JSON response",
		},
		{
			name:    "authorization is not an object",
			source:  func(t *testing.T) string { return writeConfig(t, "cfg.json", grantWith(`"generic"`)) },
			local:   true,
			wantErr: "failed to parse authorization from grant config",
		},
		{
			name: "authorization type unknown",
			source: func(t *testing.T) string {
				return writeConfig(t, "cfg.json", grantWith(`{"@type": "/nope.Authorization"}`))
			},
			local:   true,
			wantErr: "failed to convert grant config to Any",
		},
		{
			name: "fee allowance type unknown",
			source: func(t *testing.T) string {
				return writeConfig(t, "cfg.json", `{"grant_config": [], "fee_config": {"allowance": {"@type": "/nope.Allowance"}}}`)
			},
			local:   true,
			wantErr: "failed to convert fee config to Any",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{contract, tc.source(t), from, skipConfirm}
			if tc.local {
				args = append(args, "--local")
			}
			args = append(args, tc.args...)
			err := runCmd(t, cli.NewUpdateConfigsCmd(), clientCtx, args...)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestUpdateParamsCmdRejectsUnknownKey(t *testing.T) {
	env := newCLITestEnv(t)
	clientCtx := env.clientContext(routedCometRPC{})

	err := runCmd(t, cli.NewUpdateParamsCmd(), clientCtx,
		env.account.Address.String(), "https://example.com/redirect", "https://example.com/icon.png",
		fmt.Sprintf("--%s=nobody", flags.FlagFrom))
	require.ErrorContains(t, err, "key not found")
}
