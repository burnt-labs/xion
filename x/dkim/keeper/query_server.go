package keeper

import (
	"bytes"
	"context"
	"math/big"

	"github.com/vocdoni/circom2gnark/parser"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/dkim/types"
)

var _ types.QueryServer = Querier{}

// MaxPaginationLimit is the maximum number of results that can be returned in a single query
const MaxPaginationLimit = 1000

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

// DkimPubKey implements types.QueryServer.
func (k Querier) DkimPubKey(ctx context.Context, msg *types.QueryDkimPubKeyRequest) (*types.QueryDkimPubKeyResponse, error) {
	key := collections.Join(msg.Domain, msg.Selector)
	dkimPubKey, err := k.Keeper.DkimPubKeys.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &types.QueryDkimPubKeyResponse{DkimPubKey: &types.DkimPubKey{
		Domain:       dkimPubKey.Domain,
		PubKey:       dkimPubKey.PubKey,
		Selector:     dkimPubKey.Selector,
		PoseidonHash: dkimPubKey.PoseidonHash,
		Version:      dkimPubKey.Version,
		KeyType:      dkimPubKey.KeyType,
	}}, nil
}

// DkimPubKeys implements types.QueryServer
func (k Querier) DkimPubKeys(ctx context.Context, msg *types.QueryDkimPubKeysRequest) (*types.QueryDkimPubKeysResponse, error) {
	// Direct lookup when both domain and selector are provided
	if msg.Domain != "" && msg.Selector != "" {
		key := collections.Join(msg.Domain, msg.Selector)
		dkimPubKey, err := k.Keeper.DkimPubKeys.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		return &types.QueryDkimPubKeysResponse{
			DkimPubKeys: []*types.DkimPubKey{{
				Domain:       dkimPubKey.Domain,
				PubKey:       dkimPubKey.PubKey,
				Selector:     dkimPubKey.Selector,
				PoseidonHash: dkimPubKey.PoseidonHash,
				Version:      dkimPubKey.Version,
				KeyType:      dkimPubKey.KeyType,
			}},
			Pagination: nil,
		}, nil
	}

	// Parse pagination parameters
	limit := uint64(100)
	offset := uint64(0)
	useKeyBasedPagination := false
	countTotal := false
	var paginationKey []byte

	if msg.Pagination != nil {
		if msg.Pagination.Limit != 0 {
			limit = msg.Pagination.Limit
			if limit > MaxPaginationLimit {
				limit = MaxPaginationLimit
			}
		}
		if len(msg.Pagination.Key) > 0 {
			useKeyBasedPagination = true
			paginationKey = msg.Pagination.Key
		} else if msg.Pagination.Offset != 0 {
			offset = msg.Pagination.Offset
		}
		countTotal = msg.Pagination.CountTotal
	}

	// Set up the iterator range
	var ranger collections.Ranger[collections.Pair[string, string]]
	if msg.Domain != "" {
		ranger = collections.NewPrefixedPairRange[string, string](msg.Domain)
	}

	iter, err := k.Keeper.DkimPubKeys.Iterate(ctx, ranger)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	keyCodec := k.Keeper.DkimPubKeys.KeyCodec()

	// For key-based pagination, fast-forward to the starting key
	if useKeyBasedPagination && len(paginationKey) > 0 {
		for ; iter.Valid(); iter.Next() {
			fullKey, err := iter.Key()
			if err != nil {
				return nil, err
			}
			buf := make([]byte, 256)
			n, err := keyCodec.EncodeNonTerminal(buf, fullKey)
			if err != nil {
				return nil, err
			}
			// Start from the first key >= pagination key
			if bytes.Compare(buf[:n], paginationKey) >= 0 {
				break
			}
		}
	}

	// Collect results
	var results []*types.DkimPubKey
	var lastKey collections.Pair[string, string]
	var hasLastKey bool
	skipped := uint64(0)
	collected := uint64(0)
	totalMatching := uint64(0)

	for ; iter.Valid(); iter.Next() {
		dkimPubKey, err := iter.Value()
		if err != nil {
			return nil, err
		}

		// Apply PoseidonHash filter if specified
		if len(msg.PoseidonHash) > 0 && !bytes.Equal(dkimPubKey.PoseidonHash, msg.PoseidonHash) {
			continue
		}

		totalMatching++

		// For offset-based pagination (without key), skip first 'offset' matching records
		if !useKeyBasedPagination && skipped < offset {
			skipped++
			continue
		}

		// Check if we've collected enough
		if collected >= limit {
			// We have one more record, so there's a next page
			if !hasLastKey {
				lastKey, _ = iter.Key()
				hasLastKey = true
			}
			// Continue iterating to count total if needed (for offset-based pagination or CountTotal)
			if !countTotal && useKeyBasedPagination {
				break
			}
			continue
		}

		pubKey := types.DkimPubKey{
			Domain:       dkimPubKey.Domain,
			PubKey:       dkimPubKey.PubKey,
			Selector:     dkimPubKey.Selector,
			PoseidonHash: dkimPubKey.PoseidonHash,
			Version:      types.Version(dkimPubKey.Version),
			KeyType:      types.KeyType(dkimPubKey.KeyType),
		}
		results = append(results, &pubKey)
		collected++
	}

	// Generate NextKey if there are more results
	var nextKey []byte
	if hasLastKey {
		buf := make([]byte, 256)
		n, err := keyCodec.EncodeNonTerminal(buf, lastKey)
		if err != nil {
			return nil, err
		}
		nextKey = buf[:n]
	}

	pageRes := &query.PageResponse{
		NextKey: nextKey,
		Total:   totalMatching,
	}

	return &types.QueryDkimPubKeysResponse{
		DkimPubKeys: results,
		Pagination:  pageRes,
	}, nil
}

func (k Querier) Authenticate(c context.Context, req *types.QueryAuthenticateRequest) (*types.AuthenticateResponse, error) {
	var verified bool
	if len(req.PublicInputs) < 38 {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "insufficient public inputs, need at least 38 elements for email hosts, got %d", len(req.PublicInputs))
	}

	if req.EmailHash != req.PublicInputs[32] {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "email hash does not match public input, got %s, expected %s", req.EmailHash, req.PublicInputs[32])
	}

	// Verify tx_bytes match public inputs [12:32]
	txPartsFromPublicInputs, err := types.ConvertStringArrayToBigInt(req.PublicInputs[12:32])
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "failed to convert tx bytes public inputs to big int: %s", err.Error())
	}
	txBytesFromPublicInputs, err := types.ConvertBigIntArrayToString(txPartsFromPublicInputs)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "failed to convert tx bytes public inputs to string: %s", err.Error())
	}
	if !bytes.Equal(req.TxBytes, []byte(txBytesFromPublicInputs)) {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "tx bytes do not match public inputs [12:32]")
	}

	dkimDomainPInputBz, err := types.ConvertStringArrayToBigInt(req.PublicInputs[0:9])
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "failed to convert dkim domain public inputs: %s", err.Error())
	}
	dkimDomainPInput, err := types.ConvertBigIntArrayToString(dkimDomainPInputBz)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "failed to convert dkim domain public inputs to string: %s", err.Error())
	}

	dkimHashPInput := req.PublicInputs[9]
	dkimHashPInputBig, ok := new(big.Int).SetString(dkimHashPInput, 10)
	if !ok {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "failed to parse dkim hash public input")
	}

	res, err := k.DkimPubKeys(c, &types.QueryDkimPubKeysRequest{
		Domain:       dkimDomainPInput,
		PoseidonHash: dkimHashPInputBig.Bytes(),
		Pagination:   nil,
	})
	if err != nil {
		return nil, err
	}
	if len(res.DkimPubKeys) == 0 {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "no dkim pubkey found for domain and poseidon hash")
	}

	emailHostFromPublicInputs, err := types.ConvertStringArrayToBigInt(req.PublicInputs[34:43])
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "failed to convert allowed email hosts to big int: %s", err.Error())
	}

	emailHostFromPublicInputsString, err := types.ConvertBigIntArrayToString(emailHostFromPublicInputs)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "failed to convert allowed email hosts to string: %s", err.Error())
	}

	// If public inputs have email hosts but allowedEmailHosts is empty, return error
	if emailHostFromPublicInputsString != "" && len(req.AllowedEmailHosts) == 0 {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "email host from public inputs %s is not present in allowed email hosts list", emailHostFromPublicInputsString)
	}

	// Check if the email host from public inputs is present in the allowedEmailHosts list
	if !IsSubset([]string{emailHostFromPublicInputsString}, req.AllowedEmailHosts) {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "email host from public inputs %s is not present in allowed email hosts list: %s", emailHostFromPublicInputsString, req.AllowedEmailHosts)
	}

	emailSubjectFromPublicInputs, err := types.ConvertStringArrayToBigInt(req.PublicInputs[43:52])
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "failed to convertemail subject to big int: %s", err.Error())
	}

	emailSubjectFromPublicInputsString, err := types.ConvertBigIntArrayToString(emailSubjectFromPublicInputs)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "failed to convert email subject to string: %s", err.Error())
	}

	// Validate email subject for security and format compliance
	if !types.ValidateForcedSubject(emailSubjectFromPublicInputsString) {
		return nil, errors.Wrapf(types.ErrInvalidEmailSubject, "email subject validation failed: %s", emailSubjectFromPublicInputsString)
	}

	snarkProof, err := parser.UnmarshalCircomProofJSON(req.Proof)
	if err != nil {
		return nil, err
	}

	params, err := k.Keeper.Params.Get(c)
	if err != nil {
		return nil, err
	}

	vkey, err := k.ZkKeeper.GetCircomVKeyByID(c, params.VkeyIdentifier)
	if err != nil {
		return nil, err
	}

	verified, err = k.ZkKeeper.Verify(c, snarkProof, vkey, &req.PublicInputs)
	if err != nil {
		return nil, err
	}

	return &types.AuthenticateResponse{Verified: verified}, nil
}

func (k Querier) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	p, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: &p}, nil
}

// IsSubset returns true if all elements in A are contained in B
func IsSubset[T comparable](a, b []T) bool {
	// Build a set from B for O(1) lookups
	set := make(map[T]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}

	// Check if every element in A exists in B
	for _, v := range a {
		if _, exists := set[v]; !exists {
			return false
		}
	}
	return true
}
