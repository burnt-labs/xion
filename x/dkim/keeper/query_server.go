package keeper

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"fmt"

	queryv1beta1 "cosmossdk.io/api/cosmos/base/query/v1beta1"
	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/vocdoni/circom2gnark/parser"

	dkimv1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
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
	dkimPubKey, err := k.DkimPubKeys.Get(ctx, key)
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
	switch {
	case msg.Domain != "" && msg.Selector != "":
		// direct request for a pubKey
		key := collections.Join(msg.Domain, msg.Selector)
		dkimPubKey, err := k.DkimPubKeys.Get(ctx, key)
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
	case msg.Domain != "" && msg.PoseidonHash != nil:
		// all pubKeys for a domain with specific hash
		pageRequest := convertPageRequest(msg.Pagination)
		results, err := k.DkimPubKeys.List(ctx, pageRequest)
		if err != nil {
			return nil, err
		}
		pubKeys, err := consumeIteratorResults(results, msg.Domain, msg.PoseidonHash)
		if err != nil {
			return nil, err
		}
		return &types.QueryDkimPubKeysResponse{
			DkimPubKeys: pubKeys,
			Pagination:  convertPageResponse(results.PageResponse()),
		}, nil
	case msg.Domain != "":
		// all pubKeys for a domain
		pageRequest := convertPageRequest(msg.Pagination)
		results, err := k.DkimPubKeys.List(ctx, pageRequest)
		if err != nil {
			return nil, err
		}
		pubKeys, err := consumeIteratorResults(results, msg.Domain, nil)
		if err != nil {
			return nil, err
		}
		return &types.QueryDkimPubKeysResponse{
			DkimPubKeys: pubKeys,
			Pagination:  convertPageResponse(results.PageResponse()),
		}, nil
	default:
		// all pubKeys
		pageRequest := convertPageRequest(msg.Pagination)
		results, err := k.DkimPubKeys.List(ctx, pageRequest)
		if err != nil {
			return nil, err
		}
		pubKeys, err := consumeIteratorResults(results, "", nil)
		if err != nil {
			return nil, err
		}
		return &types.QueryDkimPubKeysResponse{
			DkimPubKeys: pubKeys,
			Pagination:  convertPageResponse(results.PageResponse()),
		}, nil
	}
}

func (k Querier) ProofVerify(c context.Context, req *types.QueryVerifyRequest) (*types.QueryVerifyResponse, error) {
	var verified bool
	emailHash, err := fr.LittleEndian.Element((*[32]byte)(req.EmailHash))
	if err != nil {
		return nil, errors.Wrapf(types.ErrEncodingElement, "invalid email bytes got %s", err.Error())
	}
	dkimHash, err := fr.LittleEndian.Element((*[32]byte)(req.DkimHash))
	if err != nil {
		return nil, errors.Wrapf(types.ErrEncodingElement, "invalid Dkim Hash, got %s", err.Error())
	}
	encodedTxBytes := b64.StdEncoding.EncodeToString(req.TxBytes)
	txBz, err := CalculateTxBodyCommitment(encodedTxBytes)
	// txBz, err := CalculateTxBodyCommitment(string(req.TxBytes))
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
	return &types.QueryVerifyResponse{Verified: verified}, nil
}

func convertPageRequest(request *query.PageRequest) *queryv1beta1.PageRequest {
	if request != nil {
		pageRequest := queryv1beta1.PageRequest{}
		pageRequest.CountTotal = request.CountTotal
		pageRequest.Key = request.Key
		pageRequest.Offset = request.Offset
		pageRequest.Limit = request.Limit
		pageRequest.Reverse = request.Reverse
		return &pageRequest
	}
	return nil
}

func convertPageResponse(response *queryv1beta1.PageResponse) *query.PageResponse {
	if response != nil {
		pageResponse := query.PageResponse{}
		pageResponse.NextKey = response.NextKey
		pageResponse.Total = response.Total
		return &pageResponse
	}
	return nil
}

func consumeIteratorResults(iterator collections.Iterator[collections.Pair[string, string], dkimv1.DkimPubKey], domain string, poseidonHash []byte) (output []*types.DkimPubKey, err error) {
	defer iterator.Close()

	for iterator.Next() {
		kv, err := iterator.Value()
		if err != nil {
			return nil, err
		}
		dkimPubKey := kv.Value

		// Filter by domain and/or poseidon hash if specified
		match := true
		if domain != "" && dkimPubKey.Domain != domain {
			match = false
		}
		if poseidonHash != nil && !bytes.Equal(dkimPubKey.PoseidonHash, poseidonHash) {
			match = false
		}

		if match {
			output = append(output, &types.DkimPubKey{
				Domain:       dkimPubKey.Domain,
				PubKey:       dkimPubKey.PubKey,
				Selector:     dkimPubKey.Selector,
				PoseidonHash: dkimPubKey.PoseidonHash,
				Version:      types.Version(dkimPubKey.Version),
				KeyType:      types.KeyType(dkimPubKey.KeyType),
			})
		}
	}

	return output, nil
}
