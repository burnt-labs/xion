package keeper

import (
	"context"
	"fmt"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/vocdoni/circom2gnark/parser"

	"cosmossdk.io/errors"

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
