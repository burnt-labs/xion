package keeper

import (
	"context"
	"fmt"

	"github.com/vocdoni/circom2gnark/parser"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/zk/types"
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

func (k Querier) ProofVerify(c context.Context, req *types.QueryVerifyRequest) (*types.ProofVerifyResponse, error) {
	snarkProof, err := parser.UnmarshalCircomProofJSON(req.Proof)
	if err != nil {
		return nil, err
	}

	snarkVk, err := parser.UnmarshalCircomVerificationKeyJSON(req.Vkey)
	if err != nil {
		return nil, err
	}

	verified, err := k.Verify(c, snarkProof, snarkVk, &req.PublicInputs)
	if err != nil {
		fmt.Printf("we have passed verifications with errors??: %s\n", err.Error())
		return nil, err
	}
	fmt.Println("we have passed verifications with no errors")
	return &types.ProofVerifyResponse{Verified: verified}, nil
}
