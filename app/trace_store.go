package app

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"

	abci "github.com/cometbft/cometbft/abci/types"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
)

// FlagTraceStore is the `xiond start` flag that writes every committed store
// write and delete to a file, one JSON object per line.
//
// Cosmos SDK v0.55 removed store tracing, and with it the SDK's own
// `--trace-store` flag. State indexers such as the DAO DAO indexer read that
// output from a FIFO, so xiond provides the flag itself and feeds it from the
// ADR-038 store listeners instead.
const FlagTraceStore = "trace-store"

// stateCachePopper is the part of the commit multistore the store tracer uses.
type stateCachePopper interface {
	PopStateCache() []*storetypes.StoreKVPair
}

// traceOperation is one line of trace output. The field order, names and
// encodings are those of the SDK's former tracekv store, so existing trace
// consumers keep working unchanged.
type traceOperation struct {
	Operation string         `json:"operation"`
	Key       string         `json:"key"`
	Value     string         `json:"value"`
	Metadata  map[string]any `json:"metadata"`
}

// storeTracer is an ABCIListener that writes each block's store changes to a
// trace file and passes the same change set on to the listeners it wraps.
//
// It writes during FinalizeBlock, before the block is committed: if the write
// fails, the node stops before Commit and re-executes the block on restart, so
// the trace has no gap. The SDK only hands the change set to listeners at
// Commit, so the tracer pops it from the multistore itself and gives the
// wrapped listeners the full change set at Commit.
//
// Only writes and deletes are traced. The SDK tracer also traced reads and
// iterator access, which trace consumers ignore.
type storeTracer struct {
	cms       stateCachePopper
	out       *bufio.Writer
	file      io.Closer
	closeOnce sync.Once
	closeErr  error
	height    int64
	pending   []*storetypes.StoreKVPair
	inner     []storetypes.ABCIListener
}

var _ storetypes.ABCIListener = (*storeTracer)(nil)

// openTraceStore opens the trace file the way the SDK did: write-only, append,
// created if missing. Opening a FIFO blocks until a reader opens it.
func openTraceStore(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o666) //nolint:gosec // operator-supplied path, same mode as the SDK's former trace writer
	if err != nil {
		return nil, fmt.Errorf("open trace store %q: %w", path, err)
	}
	return f, nil
}

func newStoreTracer(cms stateCachePopper, w io.Writer, inner []storetypes.ABCIListener) *storeTracer {
	t := &storeTracer{
		cms:   cms,
		out:   bufio.NewWriterSize(w, 1<<20),
		inner: inner,
	}
	if c, ok := w.(io.Closer); ok {
		t.file = c
	}
	return t
}

// ListenFinalizeBlock writes the block's store changes, then calls the wrapped
// listeners. A failed write panics, which stops the node before Commit.
func (t *storeTracer) ListenFinalizeBlock(ctx context.Context, req abci.RequestFinalizeBlock, res abci.ResponseFinalizeBlock) error {
	pairs := t.cms.PopStateCache()
	t.mustTrace(req.Height, pairs)
	t.height = req.Height
	t.pending = append(t.pending, pairs...)

	var errs []error
	for _, l := range t.inner {
		if err := l.ListenFinalizeBlock(ctx, req, res); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ListenCommit traces any changes made after FinalizeBlock and hands the
// block's whole change set to the wrapped listeners.
func (t *storeTracer) ListenCommit(ctx context.Context, res abci.ResponseCommit, changeSet []*storetypes.StoreKVPair) error {
	height := blockHeight(ctx)
	if height == 0 {
		height = t.height
	}
	t.mustTrace(height, changeSet)

	all := append(t.pending, changeSet...)
	t.pending = nil
	sort.SliceStable(all, func(i, j int) bool { return all[i].StoreKey < all[j].StoreKey })

	var errs []error
	for _, l := range t.inner {
		if err := l.ListenCommit(ctx, res, all); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Close closes the trace file. It is idempotent: SDK v0.55 server shutdown
// calls app.Close() twice.
func (t *storeTracer) Close() error {
	t.closeOnce.Do(func() {
		if t.file != nil {
			t.closeErr = t.file.Close()
		}
	})
	return t.closeErr
}

func (t *storeTracer) mustTrace(height int64, pairs []*storetypes.StoreKVPair) {
	if err := t.trace(height, pairs); err != nil {
		panic(fmt.Errorf("store tracer: height %d: %w", height, err))
	}
}

func (t *storeTracer) trace(height int64, pairs []*storetypes.StoreKVPair) error {
	if len(pairs) == 0 {
		return nil
	}
	for _, p := range pairs {
		op := traceOperation{
			Operation: "write",
			Key:       base64.StdEncoding.EncodeToString(p.Key),
			Value:     base64.StdEncoding.EncodeToString(p.Value),
			Metadata: map[string]any{
				"blockHeight": height,
				"store_name":  p.StoreKey,
			},
		}
		if p.Delete {
			op.Operation = "delete"
			op.Value = ""
		}
		raw, err := json.Marshal(op)
		if err != nil {
			return err
		}
		if _, err := t.out.Write(raw); err != nil {
			return err
		}
		if err := t.out.WriteByte('\n'); err != nil {
			return err
		}
	}
	return t.out.Flush()
}

func blockHeight(ctx context.Context) int64 {
	if h, ok := ctx.(interface{ BlockHeight() int64 }); ok {
		return h.BlockHeight()
	}
	return 0
}
