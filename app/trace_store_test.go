package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log/v2"

	"github.com/cosmos/cosmos-sdk/client/flags"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type fakePopper struct{ next []*storetypes.StoreKVPair }

func (f *fakePopper) PopStateCache() []*storetypes.StoreKVPair {
	out := f.next
	f.next = nil
	return out
}

type recordingListener struct {
	finalized []int64
	commits   [][]*storetypes.StoreKVPair
	err       error
}

func (r *recordingListener) ListenFinalizeBlock(_ context.Context, req abci.RequestFinalizeBlock, _ abci.ResponseFinalizeBlock) error {
	r.finalized = append(r.finalized, req.Height)
	return r.err
}

func (r *recordingListener) ListenCommit(_ context.Context, _ abci.ResponseCommit, changeSet []*storetypes.StoreKVPair) error {
	r.commits = append(r.commits, changeSet)
	return r.err
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

func TestStoreTracerWritesTraceKVLines(t *testing.T) {
	popper := &fakePopper{next: []*storetypes.StoreKVPair{
		{StoreKey: "bank", Key: []byte{0x02, 0x01}, Value: []byte("coins")},
		{StoreKey: "wasm", Key: []byte{0x03}, Delete: true},
	}}
	var out strings.Builder
	tracer := newStoreTracer(popper, &out, nil)

	require.NoError(t, tracer.ListenFinalizeBlock(context.Background(), abci.RequestFinalizeBlock{Height: 42}, abci.ResponseFinalizeBlock{}))

	// The SDK tracekv store's line format, byte for byte.
	require.Equal(t,
		`{"operation":"write","key":"AgE=","value":"Y29pbnM=","metadata":{"blockHeight":42,"store_name":"bank"}}`+"\n"+
			`{"operation":"delete","key":"Aw==","value":"","metadata":{"blockHeight":42,"store_name":"wasm"}}`+"\n",
		out.String())
}

func TestStoreTracerForwardsWholeChangeSetAtCommit(t *testing.T) {
	popper := &fakePopper{next: []*storetypes.StoreKVPair{
		{StoreKey: "feegrant", Key: []byte("a"), Value: []byte("1")},
		{StoreKey: "authz", Key: []byte("b"), Value: []byte("2")},
	}}
	inner := &recordingListener{}
	var out strings.Builder
	tracer := newStoreTracer(popper, &out, []storetypes.ABCIListener{inner})

	require.NoError(t, tracer.ListenFinalizeBlock(context.Background(), abci.RequestFinalizeBlock{Height: 7}, abci.ResponseFinalizeBlock{}))
	require.Equal(t, []int64{7}, inner.finalized)
	require.Empty(t, inner.commits, "wrapped listeners get the change set at Commit, as without the tracer")

	// A write that reaches the store after FinalizeBlock arrives in Commit's change set.
	late := &storetypes.StoreKVPair{StoreKey: "bank", Key: []byte("c"), Value: []byte("3")}
	require.NoError(t, tracer.ListenCommit(context.Background(), abci.ResponseCommit{}, []*storetypes.StoreKVPair{late}))

	require.Len(t, inner.commits, 1)
	var got []string
	for _, p := range inner.commits[0] {
		got = append(got, p.StoreKey+"/"+string(p.Key))
	}
	require.Equal(t, []string{"authz/b", "bank/c", "feegrant/a"}, got)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 3)
	require.Contains(t, lines[2], `"blockHeight":7,"store_name":"bank"`)

	// The next block starts with an empty change set.
	require.NoError(t, tracer.ListenFinalizeBlock(context.Background(), abci.RequestFinalizeBlock{Height: 8}, abci.ResponseFinalizeBlock{}))
	require.NoError(t, tracer.ListenCommit(context.Background(), abci.ResponseCommit{}, nil))
	require.Empty(t, inner.commits[1])
}

func TestStoreTracerReportsWrappedListenerErrors(t *testing.T) {
	failing := &recordingListener{err: errors.New("indexer down")}
	ok := &recordingListener{}
	tracer := newStoreTracer(&fakePopper{}, &strings.Builder{}, []storetypes.ABCIListener{failing, ok})

	require.ErrorContains(t, tracer.ListenFinalizeBlock(context.Background(), abci.RequestFinalizeBlock{Height: 1}, abci.ResponseFinalizeBlock{}), "indexer down")
	require.ErrorContains(t, tracer.ListenCommit(context.Background(), abci.ResponseCommit{}, nil), "indexer down")
	require.Equal(t, []int64{1}, ok.finalized, "one failing listener doesn't stop the others")
	require.Len(t, ok.commits, 1)
}

func TestStoreTracerPanicsWhenTheTraceCannotBeWritten(t *testing.T) {
	popper := &fakePopper{next: []*storetypes.StoreKVPair{{StoreKey: "bank", Key: []byte("k"), Value: []byte("v")}}}
	tracer := newStoreTracer(popper, failingWriter{}, nil)

	// Panicking in FinalizeBlock stops the node before the block is committed,
	// so the block is executed and traced again after a restart.
	require.PanicsWithError(t, "store tracer: height 3: broken pipe", func() {
		_ = tracer.ListenFinalizeBlock(context.Background(), abci.RequestFinalizeBlock{Height: 3}, abci.ResponseFinalizeBlock{})
	})
}

func TestStoreTracerCloseIsIdempotent(t *testing.T) {
	f, err := openTraceStore(filepath.Join(t.TempDir(), "trace.out"))
	require.NoError(t, err)
	tracer := newStoreTracer(&fakePopper{}, f, nil)
	require.NoError(t, tracer.Close())
	require.NoError(t, tracer.Close())
}

func TestOpenTraceStoreRejectsMissingDirectory(t *testing.T) {
	_, err := openTraceStore(filepath.Join(t.TempDir(), "missing", "trace.pipe"))
	require.ErrorContains(t, err, "open trace store")
}

func TestNewWasmAppTracesStoreWrites(t *testing.T) {
	traceFile := filepath.Join(t.TempDir(), "trace.out")
	gapp := NewWasmAppWithCustomOptions(t, false, SetupOptions{
		Logger: log.NewNopLogger(),
		DB:     dbm.NewMemDB(),
		AppOpts: simtestutil.AppOptionsMap{
			flags.FlagHome:    t.TempDir(),
			"indexer.enabled": true,
			FlagTraceStore:    traceFile,
		},
	})
	t.Cleanup(func() { require.NoError(t, gapp.Close()) })

	// The tracer is the only top-level listener and wraps the xion indexer.
	manager := gapp.StreamingManager()
	require.Len(t, manager.ABCIListeners, 1)
	tracer, ok := manager.ABCIListeners[0].(*storeTracer)
	require.True(t, ok)
	require.Len(t, tracer.inner, 1)
	require.Same(t, gapp.IndexerService(), tracer.inner[0])

	for h := int64(1); h <= 2; h++ {
		_, err := gapp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: h})
		require.NoError(t, err)
		_, err = gapp.Commit()
		require.NoError(t, err)
	}

	f, err := os.Open(traceFile)
	require.NoError(t, err)
	defer f.Close()

	// Block 1 carries the genesis state, so the genesis balances are in the
	// trace under the bank store.
	heights := map[int64]int{}
	stores := map[string]bool{}
	foundBalance := false
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for scanner.Scan() {
		var op struct {
			Operation string `json:"operation"`
			Key       string `json:"key"`
			Value     string `json:"value"`
			Metadata  struct {
				BlockHeight int64  `json:"blockHeight"`
				StoreName   string `json:"store_name"`
			} `json:"metadata"`
		}
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &op))
		require.Contains(t, []string{"write", "delete"}, op.Operation)
		require.NotEmpty(t, op.Metadata.StoreName)
		heights[op.Metadata.BlockHeight]++
		stores[op.Metadata.StoreName] = true
		if op.Metadata.StoreName == banktypes.StoreKey && op.Metadata.BlockHeight == 1 {
			key, err := base64.StdEncoding.DecodeString(op.Key)
			require.NoError(t, err)
			if bytes.HasPrefix(key, banktypes.BalancesPrefix) {
				foundBalance = true
			}
		}
	}
	require.NoError(t, scanner.Err())

	require.Positive(t, heights[1])
	require.Positive(t, heights[2], "every block's writes are traced")
	require.Len(t, heights, 2)
	require.True(t, stores[banktypes.StoreKey])
	require.True(t, stores["wasm"])
	require.True(t, foundBalance, "the genesis balance write is traced under the bank store")
}

func TestNewWasmAppWithoutTraceStoreKeepsListeners(t *testing.T) {
	gapp := NewWasmAppWithCustomOptions(t, false, SetupOptions{
		Logger:  log.NewNopLogger(),
		DB:      dbm.NewMemDB(),
		AppOpts: simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()},
	})
	t.Cleanup(func() { require.NoError(t, gapp.Close()) })

	require.Empty(t, gapp.StreamingManager().ABCIListeners)
	require.Nil(t, gapp.storeTracer)
}
