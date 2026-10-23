package app

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// TestLegacyIBCWasmGovQueries stores proposals whose messages are the retired
// 08-wasm Msg types, the way governance history on xion-testnet-2 holds them,
// and checks that the gov queries decode and render them while no Msg handler
// exists for any of them.
//
// The fixtures are synthetic protobuf bytes. "gov" is not a valid signer and
// 00 61 73 6d is not an executable module: a historical decoder must keep the
// bytes as they are without validating or executing them. The checksum is the
// byte sequence 00 through 1f.
func TestLegacyIBCWasmGovQueries(t *testing.T) {
	fixtures := []struct {
		typeURL string
		wireHex string
		json    string
	}{
		{
			"/ibc.lightclients.wasm.v1.MsgStoreCode",
			"0a03676f7612040061736d",
			`{"@type":"/ibc.lightclients.wasm.v1.MsgStoreCode","signer":"gov","wasm_byte_code":"AGFzbQ=="}`,
		},
		{
			"/ibc.lightclients.wasm.v1.MsgRemoveChecksum",
			"0a03676f761220000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
			`{"@type":"/ibc.lightclients.wasm.v1.MsgRemoveChecksum","signer":"gov","checksum":"AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="}`,
		},
		{
			"/ibc.lightclients.wasm.v1.MsgMigrateContract",
			"0a03676f76120a30382d7761736d2d31351a20000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f22027b7d",
			`{"@type":"/ibc.lightclients.wasm.v1.MsgMigrateContract","signer":"gov","client_id":"08-wasm-15","checksum":"AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=","msg":"e30="}`,
		},
	}

	gapp := Setup(t)
	ctx := gapp.NewContext(false)
	server := govkeeper.NewQueryServer(&gapp.GovKeeper)

	empty, err := server.Proposals(ctx, &govv1.QueryProposalsRequest{})
	require.NoError(t, err)
	require.Empty(t, empty.Proposals)

	for i, fixture := range fixtures {
		wire, err := hex.DecodeString(fixture.wireHex)
		require.NoError(t, err)
		id := uint64(i + 1)
		// An Any built from raw bytes, as read back from the store. Never use
		// NewAnyWithValue here: its cached value would skip the decode under test.
		proposal := govv1.Proposal{
			Id:       id,
			Status:   govv1.StatusPassed,
			Messages: []*codectypes.Any{{TypeUrl: fixture.typeURL, Value: wire}},
		}
		require.NoError(t, gapp.GovKeeper.Proposals.Set(ctx, id, proposal))
	}
	require.NoError(t, gapp.GovKeeper.Proposals.Set(ctx, 4, govv1.Proposal{
		Id:     4,
		Status: govv1.StatusRejected,
		Title:  "ordinary historical text proposal",
	}))

	first, err := server.Proposals(ctx, &govv1.QueryProposalsRequest{
		Pagination: &query.PageRequest{Limit: 2},
	})
	require.NoError(t, err)
	require.Len(t, first.Proposals, 2)
	require.Equal(t, uint64(1), first.Proposals[0].Id)
	require.Equal(t, uint64(2), first.Proposals[1].Id)
	require.NotNil(t, first.Pagination)
	require.NotEmpty(t, first.Pagination.NextKey)

	second, err := server.Proposals(ctx, &govv1.QueryProposalsRequest{
		Pagination: &query.PageRequest{Limit: 2, Key: first.Pagination.NextKey},
	})
	require.NoError(t, err)
	require.Len(t, second.Proposals, 2)
	require.Equal(t, uint64(3), second.Proposals[0].Id)
	require.Equal(t, uint64(4), second.Proposals[1].Id)
	require.NotNil(t, second.Pagination)
	require.Empty(t, second.Pagination.NextKey)

	_, err = gapp.AppCodec().MarshalJSON(first)
	require.NoError(t, err)
	_, err = gapp.AppCodec().MarshalJSON(second)
	require.NoError(t, err)

	for i, fixture := range fixtures {
		single, err := server.Proposal(ctx, &govv1.QueryProposalRequest{ProposalId: uint64(i + 1)})
		require.NoError(t, err)

		msgs, err := single.Proposal.GetMsgs()
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		wire, err := gapp.AppCodec().Marshal(msgs[0])
		require.NoError(t, err)
		require.Equal(t, fixture.wireHex, hex.EncodeToString(wire))

		rendered, err := gapp.AppCodec().MarshalJSON(single.Proposal)
		require.NoError(t, err)
		var decoded struct {
			Messages []json.RawMessage `json:"messages"`
		}
		require.NoError(t, json.Unmarshal(rendered, &decoded))
		require.Len(t, decoded.Messages, 1)
		require.JSONEq(t, fixture.json, string(decoded.Messages[0]))

		// Decodable, never executable.
		require.Nil(t, gapp.MsgServiceRouter().HandlerByTypeURL(fixture.typeURL))
	}
}

// TestLegacyIBCWasmGovRejectsInvalidMessages checks that registering the
// retired types does not make the gov queries lenient: an unknown type URL or
// a corrupt message still fails the query.
func TestLegacyIBCWasmGovRejectsInvalidMessages(t *testing.T) {
	for _, tc := range []struct {
		name          string
		typeURL       string
		wire          []byte
		errorContains string
	}{
		{"unknown type", "/ibc.lightclients.wasm.v1.UnknownMessage", nil, "UnknownMessage"},
		{"truncated field", "/ibc.lightclients.wasm.v1.MsgStoreCode", []byte{0x12, 0x80}, "unexpected EOF"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gapp := Setup(t)
			ctx := gapp.NewContext(false)
			proposal := govv1.Proposal{
				Id:       1,
				Status:   govv1.StatusPassed,
				Messages: []*codectypes.Any{{TypeUrl: tc.typeURL, Value: tc.wire}},
			}
			require.NoError(t, gapp.GovKeeper.Proposals.Set(ctx, 1, proposal))
			server := govkeeper.NewQueryServer(&gapp.GovKeeper)

			result, err := server.Proposals(ctx, &govv1.QueryProposalsRequest{})
			require.Error(t, err)
			require.Equal(t, codes.Internal, status.Code(err))
			require.ErrorContains(t, err, tc.errorContains)
			require.Nil(t, result)

			_, err = server.Proposal(ctx, &govv1.QueryProposalRequest{ProposalId: 1})
			require.Error(t, err)
			require.Equal(t, codes.Internal, status.Code(err))
		})
	}
}
