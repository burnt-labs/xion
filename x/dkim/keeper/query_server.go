package keeper

import (
	"bytes"
	"context"
	"fmt"
	stdmath "math"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
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

func (k Querier) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	p, err := k.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: &p}, nil
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

func (k Querier) ProofVerify(c context.Context, req *types.QueryVerifyRequest) (*types.ProofVerifyResponse, error) {
	var verified bool
	emailHash, err := fr.LittleEndian.Element((*[32]byte)(req.EmailHash))
	if err != nil {
		return nil, errors.Wrapf(types.ErrEncodingElement, "invalid email bytes got %s", err.Error())
	}
	dkimHash, err := fr.LittleEndian.Element((*[32]byte)(req.DkimHash))
	if err != nil {
		return nil, errors.Wrapf(types.ErrEncodingElement, "invalid Dkim Hash, got %s", err.Error())
	}
	// encodedTxBytes := b64.StdEncoding.EncodeToString(req.TxBytes)
	// txBz, err := CalculateTxBodyCommitment(encodedTxBytes)
	txBz, err := CalculateTxBodyCommitment(string(req.TxBytes))
	if err != nil {
		return nil, errors.Wrapf(types.ErrCalculatingPoseidon, "got %s", err.Error())
	}
	inputs := []string{txBz.String(), emailHash.String(), dkimHash.String()}
	snarkProof, err := parser.UnmarshalCircomProofJSON(req.Proof)
	if err != nil {
		return nil, err
	}

	p, err := k.Keeper.Params.Get(c)
	if err != nil {
		return nil, err
	}
	snarkVk, err := parser.UnmarshalCircomVerificationKeyJSON(p.Vkey)
	if err != nil {
		return nil, err
	}

	verified, err = k.Verify(c, snarkProof, snarkVk, &inputs)
	if err != nil {
		fmt.Printf("we have passed verifications with errors??: %s\n", err.Error())
		return nil, err
	}
	fmt.Println("we have passed verifications with no errors")
	return &types.ProofVerifyResponse{Verified: verified}, nil
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
