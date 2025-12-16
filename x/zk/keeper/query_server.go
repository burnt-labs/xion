package keeper

import (
	"context"
	"fmt"

	"github.com/vocdoni/circom2gnark/parser"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/zk/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

func (q Querier) ProofVerify(c context.Context, req *types.QueryVerifyRequest) (*types.ProofVerifyResponse, error) {
	if req == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "empty request")
	}

	// Validate proof bytes
	if len(req.Proof) == 0 {
		return nil, errors.Wrap(types.ErrInvalidRequest, "proof cannot be empty")
	}

	snarkProof, err := parser.UnmarshalCircomProofJSON(req.Proof)
	if err != nil {
		return nil, err
	}
	// Get the verification key by name or ID
	var snarkVk *parser.CircomVerificationKey

	switch {
	case req.VkeyName != "":
		// Retrieve by name
		snarkVk, err = q.GetCircomVKeyByName(c, req.VkeyName)
		if err != nil {
			return nil, errors.Wrap(types.ErrVKeyNotFound, fmt.Sprintf("failed to get vkey '%s': %v", req.VkeyName, err))
		}
	case req.VkeyId != 0:
		// Retrieve by ID
		snarkVk, err = q.GetCircomVKeyByID(c, req.VkeyId)
		if err != nil {
			return nil, errors.Wrap(types.ErrVKeyNotFound, fmt.Sprintf("failed to get vkey ID %d: %v", req.VkeyId, err))
		}
	default:
		return nil, errors.Wrap(types.ErrInvalidRequest, "either vkey_name or vkey_id must be provided")
	}
	verified, err := q.Verify(c, snarkProof, snarkVk, &req.PublicInputs)
	if err != nil {
		return nil, err
	}
	return &types.ProofVerifyResponse{Verified: verified}, nil
}

// VKey queries a verification key by ID
func (q Querier) VKey(goCtx context.Context, req *types.QueryVKeyRequest) (*types.QueryVKeyResponse, error) {
	if req == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "empty request")
	}

	vkey, err := q.GetVKeyByID(goCtx, req.Id)
	if err != nil {
		return nil, err
	}

	return &types.QueryVKeyResponse{
		Vkey: vkey,
	}, nil
}

// VKeyByName queries a verification key by name
func (q Querier) VKeyByName(goCtx context.Context, req *types.QueryVKeyByNameRequest) (*types.QueryVKeyByNameResponse, error) {
	if req == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "empty request")
	}

	if req.Name == "" {
		return nil, errors.Wrap(types.ErrInvalidRequest, "name cannot be empty")
	}

	// Get ID from name index
	id, err := q.VKeyNameIndex.Get(goCtx, req.Name)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return nil, errors.Wrapf(types.ErrVKeyNotFound, "verification key with name '%s' not found", req.Name)
		}
		return nil, err
	}

	// Get vkey
	vkey, err := q.GetVKeyByID(goCtx, id)
	if err != nil {
		return nil, err
	}

	return &types.QueryVKeyByNameResponse{
		Vkey: vkey,
		Id:   id,
	}, nil
}

// VKeys queries all verification keys with pagination
func (q Querier) VKeys(goCtx context.Context, req *types.QueryVKeysRequest) (*types.QueryVKeysResponse, error) {
	if req == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "empty request")
	}

	// Use collections pagination - it returns (results, pageResponse, error)
	vkeys, pageResp, err := query.CollectionPaginate(
		goCtx,
		q.Keeper.VKeys,
		req.Pagination,
		func(id uint64, vkey types.VKey) (types.VKeyWithID, error) {
			return types.VKeyWithID{
				Id:   id,
				Vkey: vkey,
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryVKeysResponse{
		Vkeys:      vkeys,
		Pagination: pageResp,
	}, nil
}

// HasVKey checks if a verification key exists by name
func (q Querier) HasVKey(goCtx context.Context, req *types.QueryHasVKeyRequest) (*types.QueryHasVKeyResponse, error) {
	if req == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "empty request")
	}

	if req.Name == "" {
		return nil, errors.Wrap(types.ErrInvalidRequest, "name cannot be empty")
	}

	// Check if name exists in index
	id, err := q.VKeyNameIndex.Get(goCtx, req.Name)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return &types.QueryHasVKeyResponse{
				Exists: false,
				Id:     0,
			}, nil
		}
		return nil, err
	}

	return &types.QueryHasVKeyResponse{
		Exists: true,
		Id:     id,
	}, nil
}

// NextVKeyID returns the next available verification key ID
func (q Querier) NextVKeyID(goCtx context.Context, req *types.QueryNextVKeyIDRequest) (*types.QueryNextVKeyIDResponse, error) {
	if req == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "empty request")
	}

	// Peek at the sequence without incrementing it
	nextID, err := q.Keeper.NextVKeyID.Peek(goCtx)
	if err != nil {
		return nil, err
	}

	return &types.QueryNextVKeyIDResponse{
		NextId: nextID,
	}, nil
}
