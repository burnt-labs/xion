package keeper

import (
	"bytes"
	"context"
	"fmt"
	stdmath "math"
	"math/big"

	"github.com/vocdoni/circom2gnark/parser"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/dkim/types"
)

var _ types.QueryServer = Querier{}

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
		Version:      types.Version(dkimPubKey.Version),
		KeyType:      types.KeyType(dkimPubKey.KeyType),
	}}, nil
}

// DkimPubKeys implements types.QueryServer
func (k Querier) DkimPubKeys(ctx context.Context, msg *types.QueryDkimPubKeysRequest) (*types.QueryDkimPubKeysResponse, error) {
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
				Version:      types.Version(dkimPubKey.Version),
				KeyType:      types.KeyType(dkimPubKey.KeyType),
			}},
			Pagination: nil,
		}, nil
	}

	var ranger collections.Ranger[collections.Pair[string, string]]
	if msg.Domain != "" {
		ranger = collections.NewPrefixedPairRange[string, string](msg.Domain)
	}

	iter, err := k.Keeper.DkimPubKeys.Iterate(ctx, ranger)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var allPubKeys []*types.DkimPubKey
	for ; iter.Valid(); iter.Next() {
		dkimPubKey, err := iter.Value()
		if err != nil {
			return nil, err
		}

		if len(msg.PoseidonHash) > 0 && !bytes.Equal(dkimPubKey.PoseidonHash, msg.PoseidonHash) {
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
		allPubKeys = append(allPubKeys, &pubKey)
	}

	// Pagination
	offset, limit := uint64(0), uint64(100)
	if msg.Pagination != nil {
		if msg.Pagination.Offset != 0 {
			offset = msg.Pagination.Offset
		}
		if msg.Pagination.Limit != 0 {
			limit = msg.Pagination.Limit
		} else {
			limit = 100
		}
	}

	allPubKeysLen := uint64(len(allPubKeys))

	// Safe conversion: check if offset is within bounds
	if offset >= allPubKeysLen {
		return &types.QueryDkimPubKeysResponse{
			DkimPubKeys: []*types.DkimPubKey{},
			Pagination: &query.PageResponse{
				NextKey: nil,
				Total:   allPubKeysLen,
			},
		}, nil
	}

	// Safe addition: check for overflow and bounds
	endOffset := math.NewUint(offset).Add(math.NewUint(limit))
	var end uint64
	if endOffset.GT(math.NewUint(allPubKeysLen)) {
		end = allPubKeysLen
	} else {
		end = endOffset.Uint64()
	}

	// Safe conversion to int for slicing - validate against math.MaxInt to prevent overflow
	if offset > uint64(stdmath.MaxInt) || end > uint64(stdmath.MaxInt) {
		return nil, fmt.Errorf("pagination offset or end exceeds maximum int value")
	}
	offsetInt := int(offset)
	endInt := int(end)
	paginatedPubKeys := allPubKeys[offsetInt:endInt]

	// TODO: Implement key-based pagination for nextKey when needed
	nextKey := []byte(nil)

	pageRes := &query.PageResponse{
		NextKey: nextKey,
		Total:   allPubKeysLen,
	}

	return &types.QueryDkimPubKeysResponse{
		DkimPubKeys: paginatedPubKeys,
		Pagination:  pageRes,
	}, nil
}

func (k Querier) Authenticate(c context.Context, req *types.QueryAuthenticateRequest) (*types.AuthenticateResponse, error) {
	var verified bool
	if len(req.PublicInputs) < 38 {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "insufficient public inputs, need at least 38 elements for email hosts, got %d", len(req.PublicInputs))
	}

	if req.EmailHash != req.PublicInputs[32] {
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "email hash does not match public input, got %s, expected %s\n", req.EmailHash, req.PublicInputs[32])
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
		return nil, errors.Wrapf(types.ErrInvalidPublicInput, "no dkim pubkey found for domain %s and poseidon hash %s", dkimDomainPInput, dkimHashPInputBig.String())
	}

	emailHostFromPublicInputs, err := types.ConvertStringArrayToBigInt(req.PublicInputs[34:38])
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

func (k Querier) ProofVerify(c context.Context, req *types.QueryAuthenticateRequest) (*types.AuthenticateResponse, error) {
	return &types.AuthenticateResponse{Verified: false}, nil
}

func (k Querier) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	p, err := k.Keeper.Params.Get(ctx)
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

// func convertPageRequest(request *query.PageRequest) *queryv1beta1.PageRequest {
// 	if request != nil {
// 		pageRequest := queryv1beta1.PageRequest{}
// 		pageRequest.CountTotal = request.CountTotal
// 		pageRequest.Key = request.Key
// 		pageRequest.Offset = request.Offset
// 		pageRequest.Limit = request.Limit
// 		pageRequest.Reverse = request.Reverse
// 		return &pageRequest
// 	}
// 	return nil
// }

// func convertPageResponse(response *queryv1beta1.PageResponse) *query.PageResponse {
// 	if response != nil {
// 		pageResponse := query.PageResponse{}
// 		pageResponse.NextKey = response.NextKey
// 		pageResponse.Total = response.Total
// 		return &pageResponse
// 	}
// 	return nil
// }

// func consumeIteratorResults(iterator collections.Iterator[collections.Pair[string, string], apiv1.DkimPubKey], domain string, poseidonHash []byte) ([]*types.DkimPubKey, error) {
// 	defer iterator.Close()

// 	var output []*types.DkimPubKey
// 	for ; iterator.Valid(); iterator.Next() {
// 		dkimPubKey, err := iterator.Value()
// 		if err != nil {
// 			return nil, err
// 		}

// 		match := true
// 		if domain != "" && dkimPubKey.Domain != domain {
// 			match = false
// 		}
// 		if len(poseidonHash) > 0 && !bytes.Equal(dkimPubKey.PoseidonHash, poseidonHash) {
// 			match = false
// 		}

// 		if match {
// 			output = append(output, &types.DkimPubKey{
// 				Domain:       dkimPubKey.Domain,
// 				PubKey:       dkimPubKey.PubKey,
// 				Selector:     dkimPubKey.Selector,
// 				PoseidonHash: dkimPubKey.PoseidonHash,
// 				Version:      types.Version(dkimPubKey.Version),
// 				KeyType:      types.KeyType(dkimPubKey.KeyType),
// 			})
// 		}
// 	}

// 	return output, nil
// }
