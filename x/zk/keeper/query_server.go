package keeper

import (
	"context"
	"fmt"

	"github.com/vocdoni/circom2gnark/parser"

	"github.com/burnt-labs/xion/x/zk/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
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
