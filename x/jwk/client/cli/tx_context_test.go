package cli_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/jwk/client/cli"
)

// unparseableNode fails while the client context is being read from flags.
const unparseableNode = "tcp://[::1"

func rsaPublicJWK(t *testing.T) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	key, err := jwk.FromRaw(&priv.PublicKey)
	require.NoError(t, err)
	require.NoError(t, key.Set(jwk.AlgorithmKey, jwa.RS256))
	bz, err := json.Marshal(key)
	require.NoError(t, err)
	return string(bz)
}

func writeTempPEM(t *testing.T) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o600))
	return path
}

func TestCommandsRejectUnparseableNode(t *testing.T) {
	ctx := newMockCtx(t)
	key := rsaPublicJWK(t)

	cases := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"create-audience-claim", cli.CmdCreateAudienceClaim(), []string{"aud"}},
		{"create-audience", cli.CmdCreateAudience(), []string{"aud", key}},
		{"update-audience", cli.CmdUpdateAudience(), []string{"aud", key}},
		{"delete-audience", cli.CmdDeleteAudience(), []string{"aud"}},
		{"delete-audience-claim", cli.CmdDeleteAudienceClaim(), []string{"aud"}},
		{"list-audience", cli.CmdListAudience(), []string{}},
		{"show-audience", cli.CmdShowAudience(), []string{"aud"}},
		{"show-audience-claim", cli.CmdShowAudienceClaim(), []string{"AQID"}},
		{"params", cli.CmdQueryParams(), []string{}},
		{"validate-jwt", cli.CmdValidateJWT(), []string{"aud", "sub", "a.b.c"}},
		{"decode-jwt", cli.CmdDecodeJWT(), []string{"aud", "sub", "a.b.c"}},
		{"verify-jws", cli.CmdVerifyJWS(), []string{"aud", "a.b.c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := clitestutil.ExecTestCLICmd(ctx, tc.cmd, append(tc.args, "--node", unparseableNode))
			require.Error(t, err)
			require.Contains(t, err.Error(), "missing ']' in host")
		})
	}

	t.Run("convert-pem", func(t *testing.T) {
		pemFile := writeTempPEM(t)
		_, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdConvertPemToJSON(), []string{pemFile, "--node", unparseableNode})
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing ']' in host")
	})
}

func TestTxCommandsGenerateUnsignedTx(t *testing.T) {
	ctx := newMockCtx(t)
	from := sdk.AccAddress("test_from_address_____").String()
	key := rsaPublicJWK(t)

	cases := []struct {
		name    string
		cmd     *cobra.Command
		args    []string
		typeURL string
	}{
		{"create-audience-claim", cli.CmdCreateAudienceClaim(), []string{"aud"}, "MsgCreateAudienceClaim"},
		{"create-audience", cli.CmdCreateAudience(), []string{"aud", key}, "MsgCreateAudience"},
		{"create-audience with explicit admin", cli.CmdCreateAudience(), []string{"aud", key, from}, "MsgCreateAudience"},
		{"update-audience", cli.CmdUpdateAudience(), []string{"aud", key, "--" + cli.FlagNewAud, "aud2"}, "MsgUpdateAudience"},
		{"delete-audience", cli.CmdDeleteAudience(), []string{"aud"}, "MsgDeleteAudience"},
		{"delete-audience-claim", cli.CmdDeleteAudienceClaim(), []string{"aud"}, "MsgDeleteAudienceClaim"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := clitestutil.ExecTestCLICmd(ctx, tc.cmd, append(tc.args,
				"--"+flags.FlagFrom, from,
				"--"+flags.FlagGenerateOnly,
			))
			require.NoError(t, err)
			require.Contains(t, out.String(), tc.typeURL)
			require.Contains(t, out.String(), from)
		})
	}
}

func TestListAudienceRejectsPageWithOffset(t *testing.T) {
	_, err := clitestutil.ExecTestCLICmd(newMockCtx(t), cli.CmdListAudience(), []string{"--page", "2", "--offset", "1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "page and offset cannot be used together")
}

func TestQueryCommandsSurfaceQueryErrors(t *testing.T) {
	ctx := newMockCtx(t).WithClient(clitestutil.NewMockCometRPC(abci.ResponseQuery{Code: 1, Log: "boom"}))

	cases := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"list-audience", cli.CmdListAudience(), []string{}},
		{"show-audience", cli.CmdShowAudience(), []string{"aud"}},
		{"show-audience-claim", cli.CmdShowAudienceClaim(), []string{"AQID"}},
		{"params", cli.CmdQueryParams(), []string{}},
		{"validate-jwt", cli.CmdValidateJWT(), []string{"aud", "sub", "a.b.c"}},
		{"decode-jwt", cli.CmdDecodeJWT(), []string{"aud", "sub", "a.b.c"}},
		{"verify-jws", cli.CmdVerifyJWS(), []string{"aud", "a.b.c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := clitestutil.ExecTestCLICmd(ctx, tc.cmd, tc.args)
			require.Error(t, err)
			require.Contains(t, err.Error(), "boom")
		})
	}
}
