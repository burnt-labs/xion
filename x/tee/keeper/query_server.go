package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/google/go-tdx-guest/abi"
	pb "github.com/google/go-tdx-guest/proto/tdx"
	"github.com/google/go-tdx-guest/verify"

	"github.com/burnt-labs/xion/x/tee/types"
)

var _ types.QueryServer = Querier{}

// Querier implements the tee module query server.
type Querier struct {
	Keeper
}

// NewQuerier returns a new Querier instance.
func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

// noOpGetter is a safety-net HTTPSGetter that prevents any network calls.
type noOpGetter struct{}

func (n *noOpGetter) Get(url string) (map[string][]string, []byte, error) {
	return nil, nil, fmt.Errorf("network calls are not permitted during TDX quote verification")
}

// QuoteVerify verifies a TDX quote and returns the parsed header and body fields.
func (q Querier) QuoteVerify(goCtx context.Context, req *types.QueryQuoteVerifyRequest) (*types.QueryQuoteVerifyResponse, error) {
	if req == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "empty request")
	}

	if len(req.Quote) == 0 {
		return nil, errors.Wrap(types.ErrInvalidRequest, "quote cannot be empty")
	}

	if err := types.ValidateQuoteSize(req.Quote); err != nil {
		return nil, err
	}

	// Parse the raw quote bytes into a protobuf QuoteV4 structure.
	parsed, err := abi.QuoteToProto(req.Quote)
	if err != nil {
		return nil, errors.Wrap(types.ErrQuoteParseFailed, err.Error())
	}

	quoteV4, ok := parsed.(*pb.QuoteV4)
	if !ok {
		return nil, errors.Wrap(types.ErrQuoteParseFailed, "unsupported quote format: expected QuoteV4")
	}

	// Set up deterministic verification options:
	// - No network calls (GetCollateral: false, CheckRevocations: false, no-op Getter)
	// - Deterministic time from block context
	// - Embedded Intel root CA (TrustedRoots: nil uses library-embedded certs)
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	opts := &verify.Options{
		GetCollateral:  false,
		CheckRevocations: false,
		TrustedRoots:  nil,
		Now:            sdkCtx.BlockTime(),
		Getter:         &noOpGetter{},
	}

	verifyErr := verify.TdxQuote(quoteV4, opts)

	// Map the parsed QuoteV4 fields to our response types.
	header := mapHeader(quoteV4.GetHeader())
	body := mapTDQuoteBody(quoteV4.GetTdQuoteBody())

	return &types.QueryQuoteVerifyResponse{
		Verified:    verifyErr == nil,
		Header:      header,
		TdQuoteBody: body,
	}, nil
}

// mapHeader converts a go-tdx-guest Header to our protobuf QuoteHeader.
func mapHeader(h *pb.Header) *types.QuoteHeader {
	if h == nil {
		return nil
	}
	return &types.QuoteHeader{
		Version:            h.Version,
		AttestationKeyType: h.AttestationKeyType,
		TeeType:            h.TeeType,
		QeSvn:              h.QeSvn,
		PceSvn:             h.PceSvn,
		QeVendorId:         h.QeVendorId,
		UserData:           h.UserData,
	}
}

// mapTDQuoteBody converts a go-tdx-guest TDQuoteBody to our protobuf TDQuoteBody.
func mapTDQuoteBody(b *pb.TDQuoteBody) *types.TDQuoteBody {
	if b == nil {
		return nil
	}
	return &types.TDQuoteBody{
		TeeTcbSvn:      b.TeeTcbSvn,
		MrSeam:         b.MrSeam,
		MrSignerSeam:   b.MrSignerSeam,
		SeamAttributes: b.SeamAttributes,
		TdAttributes:   b.TdAttributes,
		Xfam:           b.Xfam,
		MrTd:           b.MrTd,
		MrConfigId:     b.MrConfigId,
		MrOwner:        b.MrOwner,
		MrOwnerConfig:  b.MrOwnerConfig,
		Rtmrs:          b.Rtmrs,
		ReportData:     b.ReportData,
	}
}
