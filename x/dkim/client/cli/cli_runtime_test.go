package cli_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/dkim/client/cli"
	"github.com/burnt-labs/xion/x/dkim/types"
)

// unparseableNode is rejected while the client context is being built, before
// any request is made.
const unparseableNode = "tcp://[::1"

const validRSAPubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"

func newQueryCtx(t *testing.T, resp abci.ResponseQuery) client.Context {
	t.Helper()
	encCfg := app.MakeEncodingConfig(t)
	return client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithKeyring(keyring.NewInMemory(encCfg.Codec)).
		WithChainID("test-chain").
		WithOutputFormat(flags.OutputFormatJSON).
		WithClient(clitestutil.NewMockCometRPC(resp))
}

func TestQueryCommandsRejectUnparseableNode(t *testing.T) {
	ctx := client.Context{}.WithClient(clitestutil.MockCometRPC{Client: rpcclientmock.Client{}})

	cases := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"dkim-pubkey", cli.GetDkimPublicKey(), []string{"example.com", "s1"}},
		{"dkim-pubkeys", cli.GetDkimPublicKeys(), []string{}},
		{"generate-dkim-pubkey", cli.GenerateDkimPublicKey(), []string{"example.com", "s1"}},
		{"params", cli.GetParams(), []string{}},
		{"decode", cli.NewTxDecodeCmd(), []string{"AA=="}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := clitestutil.ExecTestCLICmd(ctx, tc.cmd, append(tc.args, "--node", unparseableNode))
			require.Error(t, err)
			require.Contains(t, err.Error(), "missing ']' in host")
		})
	}
}

func TestQueryCommandsPrintResponses(t *testing.T) {
	t.Run("dkim-pubkey", func(t *testing.T) {
		resp := &types.QueryDkimPubKeyResponse{DkimPubKey: &types.DkimPubKey{Domain: "example.com", Selector: "s1", PubKey: "cHVi"}}
		bz, err := resp.Marshal()
		require.NoError(t, err)

		out, err := clitestutil.ExecTestCLICmd(newQueryCtx(t, abci.ResponseQuery{Value: bz}), cli.GetDkimPublicKey(), []string{"example.com", "s1"})
		require.NoError(t, err)
		require.Contains(t, out.String(), `"domain":"example.com"`)
	})

	t.Run("dkim-pubkeys", func(t *testing.T) {
		resp := &types.QueryDkimPubKeysResponse{DkimPubKeys: []*types.DkimPubKey{{Domain: "example.com", Selector: "s1"}}}
		bz, err := resp.Marshal()
		require.NoError(t, err)

		out, err := clitestutil.ExecTestCLICmd(newQueryCtx(t, abci.ResponseQuery{Value: bz}), cli.GetDkimPublicKeys(), []string{"--domain", "example.com"})
		require.NoError(t, err)
		require.Contains(t, out.String(), `"selector":"s1"`)
	})

	t.Run("params", func(t *testing.T) {
		p := types.DefaultParams()
		resp := &types.QueryParamsResponse{Params: &p}
		bz, err := resp.Marshal()
		require.NoError(t, err)

		out, err := clitestutil.ExecTestCLICmd(newQueryCtx(t, abci.ResponseQuery{Value: bz}), cli.GetParams(), []string{})
		require.NoError(t, err)
		require.Contains(t, out.String(), `"vkey_identifier":"1"`)
	})

	t.Run("query failure is returned", func(t *testing.T) {
		_, err := clitestutil.ExecTestCLICmd(newQueryCtx(t, abci.ResponseQuery{Code: 1, Log: "boom"}), cli.GetParams(), []string{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "boom")
	})
}

func TestRevokeDkimCmdErrors(t *testing.T) {
	ctx := newQueryCtx(t, abci.ResponseQuery{})

	t.Run("tx context cannot be built", func(t *testing.T) {
		_, err := clitestutil.ExecTestCLICmd(ctx, cli.MsgRevokeDkimPubKey(), []string{"example.com", "not-a-key", "--node", unparseableNode})
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing ']' in host")
	})

	t.Run("private key is not parseable", func(t *testing.T) {
		_, err := clitestutil.ExecTestCLICmd(ctx, cli.MsgRevokeDkimPubKey(), []string{"example.com", "not-a-key"})
		require.ErrorIs(t, err, types.ErrParsingPrivKey)
	})
}

func TestDecodeTxSignDocErrors(t *testing.T) {
	encCfg := app.MakeEncodingConfig(t)
	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	validBody, err := (&txtypes.TxBody{Memo: "sign doc"}).Marshal()
	require.NoError(t, err)
	validAuthInfo, err := (&txtypes.AuthInfo{Fee: &txtypes.Fee{GasLimit: 1}}).Marshal()
	require.NoError(t, err)

	encode := func(t *testing.T, doc *txtypes.SignDoc) string {
		t.Helper()
		bz, err := doc.Marshal()
		require.NoError(t, err)
		return base64.StdEncoding.EncodeToString(bz)
	}

	t.Run("body bytes are not a tx body", func(t *testing.T) {
		err := cli.DecodeTx(clientCtx, encode(t, &txtypes.SignDoc{
			BodyBytes:     []byte{0xff, 0xff},
			AuthInfoBytes: validAuthInfo,
			ChainId:       "test-chain",
		}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode sign doc body")
	})

	t.Run("auth info bytes are not auth info", func(t *testing.T) {
		err := cli.DecodeTx(clientCtx, encode(t, &txtypes.SignDoc{
			BodyBytes:     validBody,
			AuthInfoBytes: []byte{0xff, 0xff},
			ChainId:       "test-chain",
		}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode sign doc auth info")
	})

	t.Run("body message type cannot be resolved", func(t *testing.T) {
		body, err := (&txtypes.TxBody{Messages: []*codectypes.Any{{TypeUrl: "/unknown.Msg", Value: []byte{0x08, 0x01}}}}).Marshal()
		require.NoError(t, err)
		err = cli.DecodeTx(clientCtx, encode(t, &txtypes.SignDoc{
			BodyBytes:     body,
			AuthInfoBytes: validAuthInfo,
			ChainId:       "test-chain",
		}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to marshal body to JSON")
	})

	t.Run("signer public key type cannot be resolved", func(t *testing.T) {
		authInfo, err := (&txtypes.AuthInfo{
			SignerInfos: []*txtypes.SignerInfo{{PublicKey: &codectypes.Any{TypeUrl: "/unknown.PubKey", Value: []byte{0x0a, 0x00}}}},
			Fee:         &txtypes.Fee{GasLimit: 1},
		}).Marshal()
		require.NoError(t, err)
		err = cli.DecodeTx(clientCtx, encode(t, &txtypes.SignDoc{
			BodyBytes:     validBody,
			AuthInfoBytes: authInfo,
			ChainId:       "test-chain",
		}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to marshal auth info to JSON")
	})
}

// serveTXT answers every DNS question on conn with the given TXT records,
// speaking the length-prefixed stream framing the Go resolver uses for
// non-packet connections.
func serveTXT(conn net.Conn, records []string) {
	defer conn.Close()
	for {
		var lenBuf [2]byte
		if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
			return
		}
		query := make([]byte, binary.BigEndian.Uint16(lenBuf[:]))
		if _, err := io.ReadFull(conn, query); err != nil {
			return
		}
		if len(query) < 12 {
			return
		}
		// copy the question section verbatim: labels until the root, then type and class
		qEnd := 12
		for qEnd < len(query) && query[qEnd] != 0 {
			qEnd += int(query[qEnd]) + 1
		}
		qEnd += 1 + 4
		if qEnd > len(query) {
			return
		}

		resp := make([]byte, 0, 512)
		resp = append(resp, query[0], query[1]) // transaction id
		resp = append(resp, 0x81, 0x80)         // standard response, recursion available
		resp = binary.BigEndian.AppendUint16(resp, 1)
		resp = binary.BigEndian.AppendUint16(resp, uint16(len(records)))
		resp = binary.BigEndian.AppendUint16(resp, 0)
		resp = binary.BigEndian.AppendUint16(resp, 0)
		resp = append(resp, query[12:qEnd]...)
		for _, record := range records {
			var rdata []byte
			for rest := []byte(record); len(rest) > 0; {
				n := min(len(rest), 255)
				rdata = append(rdata, byte(n))
				rdata = append(rdata, rest[:n]...)
				rest = rest[n:]
			}
			resp = append(resp, 0xc0, 0x0c)                // pointer to the question name
			resp = binary.BigEndian.AppendUint16(resp, 16) // TXT
			resp = binary.BigEndian.AppendUint16(resp, 1)  // IN
			resp = binary.BigEndian.AppendUint32(resp, 60)
			resp = binary.BigEndian.AppendUint16(resp, uint16(len(rdata)))
			resp = append(resp, rdata...)
		}

		out := binary.BigEndian.AppendUint16(nil, uint16(len(resp)))
		out = append(out, resp...)
		if _, err := conn.Write(out); err != nil {
			return
		}
	}
}

// withFakeTXTRecords routes DNS lookups for the duration of the test to an
// in-process server returning records.
func withFakeTXTRecords(t *testing.T, records []string) {
	t.Helper()
	orig := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
			clientConn, serverConn := net.Pipe()
			go serveTXT(serverConn, records)
			return clientConn, nil
		},
	}
	t.Cleanup(func() { net.DefaultResolver = orig })
}

func TestGetDKIMPublicKeyFromTXTRecords(t *testing.T) {
	t.Run("record without a p tag is not a key", func(t *testing.T) {
		withFakeTXTRecords(t, []string{"v=spf1 -all", "v=DKIM1; k=rsa"})
		_, err := cli.GetDKIMPublicKey("s1", "example.com")
		require.Error(t, err)
		require.Contains(t, err.Error(), "DKIM public key not found for selector s1")
	})

	t.Run("p tag is extracted", func(t *testing.T) {
		withFakeTXTRecords(t, []string{"v=DKIM1; k=rsa; p=" + validRSAPubKey + ";"})
		key, err := cli.GetDKIMPublicKey("s1", "example.com")
		require.NoError(t, err)
		require.Equal(t, validRSAPubKey, key)
	})
}

func TestGenerateDkimPubKeyMsgFromTXTRecords(t *testing.T) {
	withFakeTXTRecords(t, []string{"v=DKIM1; k=rsa; p=" + validRSAPubKey})

	expectedHash, err := types.ComputePoseidonHash(validRSAPubKey)
	require.NoError(t, err)

	msg, err := cli.GenerateDkimPubKeyMsg("example.com", "s1")
	require.NoError(t, err)
	require.Equal(t, "example.com", msg.Domain)
	require.Equal(t, "s1", msg.Selector)
	require.Equal(t, validRSAPubKey, msg.PubKey)
	require.Equal(t, expectedHash.Bytes(), msg.PoseidonHash)
	require.Equal(t, types.Version_VERSION_DKIM1_UNSPECIFIED, msg.Version)
	require.Equal(t, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, msg.KeyType)

	out, err := clitestutil.ExecTestCLICmd(newQueryCtx(t, abci.ResponseQuery{}), cli.GenerateDkimPublicKey(), []string{"example.com", "s1"})
	require.NoError(t, err)
	require.Contains(t, out.String(), `"domain":"example.com"`)
	require.Contains(t, out.String(), base64.StdEncoding.EncodeToString(expectedHash.Bytes()))
}
