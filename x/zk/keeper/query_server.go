package keeper

import (
	"context"
	goerrors "errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/burnt-labs/barretenberg-go/barretenberg"
	"github.com/vocdoni/circom2gnark/parser"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	if req.VkeyName == "" && req.VkeyId == 0 {
		return nil, errors.Wrap(types.ErrInvalidRequest, "either vkey_name or vkey_id must be provided")
	}

	params, err := q.GetParams(c)
	if err != nil {
		return nil, err
	}

	// No gas is charged for this whitelisted query — size limits (MaxGroth16ProofSizeBytes,
	// MaxGroth16PublicInputSizeBytes) and governance-controlled ceilings (MaxAllowedProofOrInputSizeBytes)
	// serve as the DoS governors, consistent with v28 behavior.
	if uint64(len(req.Proof)) > params.MaxGroth16ProofSizeBytes {
		return nil, errors.Wrapf(
			types.ErrProofTooLarge,
			"proof size %d > max %d bytes",
			len(req.Proof),
			params.MaxGroth16ProofSizeBytes,
		)
	}

	// Approximate public-input payload size as total UTF-8 byte length of all provided strings.
	var publicInputsSize uint64
	for _, in := range req.PublicInputs {
		inputSize := uint64(len(in))
		if publicInputsSize+inputSize < publicInputsSize {
			return nil, errors.Wrap(types.ErrPublicInputsTooLarge, "public inputs size overflow")
		}
		publicInputsSize += inputSize
	}
	if publicInputsSize > params.MaxGroth16PublicInputSizeBytes {
		return nil, errors.Wrapf(
			types.ErrPublicInputsTooLarge,
			"public inputs size %d > max %d bytes",
			publicInputsSize,
			params.MaxGroth16PublicInputSizeBytes,
		)
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
	// Reject any public input that is not a canonical BN254 scalar field element.
	// circom2gnark's fr.Element.SetBigInt silently reduces values >= p modulo p,
	// so an input p+x would verify identically to x, enabling proof forgery.
	if err := validatePublicInputsInScalarField(req.PublicInputs); err != nil {
		return nil, err
	}

	verified, err := q.Verify(c, snarkProof, snarkVk, &req.PublicInputs)
	if err != nil {
		return nil, err
	}
	return &types.ProofVerifyResponse{Verified: verified}, nil
}

// bn254ScalarFieldPrime is the BN254 scalar field modulus r.
// All Groth16 public inputs must be strictly less than this value.
var bn254ScalarFieldPrime, _ = new(big.Int).SetString(
	"21888242871839275222246405745257275088548364400416034343698204186575808495617", 10)

// validatePublicInputsInScalarField rejects any public input string whose numeric
// value is >= the BN254 scalar field prime.  Inputs may be decimal or 0x-prefixed hex,
// matching the formats accepted by circom2gnark's ConvertPublicInputs.
func validatePublicInputsInScalarField(inputs []string) error {
	for i, inp := range inputs {
		s := inp
		base := 10
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			s = s[2:]
			base = 16
		}
		v, ok := new(big.Int).SetString(s, base)
		if !ok || v.Sign() < 0 || v.Cmp(bn254ScalarFieldPrime) >= 0 {
			return errors.Wrapf(types.ErrInvalidRequest,
				"public input[%d] is not a canonical BN254 scalar field element", i)
		}
	}
	return nil
}

// ProofVerifyUltraHonk verifies an UltraHonk (Barretenberg) proof using a vkey looked up by name or ID.
func (q Querier) ProofVerifyUltraHonk(c context.Context, req *types.QueryVerifyUltraHonkRequest) (*types.ProofVerifyUltraHonkResponse, error) {
	if req == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "empty request")
	}
	if len(req.GetProof()) == 0 {
		return nil, errors.Wrap(types.ErrInvalidRequest, "proof cannot be empty")
	}

	params, err := q.GetParams(c)
	if err != nil {
		return nil, err
	}

	// No gas is charged for this whitelisted query — size limits (MaxUltraHonkProofSizeBytes,
	// MaxUltraHonkPublicInputSizeBytes) and governance-controlled ceilings (MaxAllowedProofOrInputSizeBytes)
	// serve as the DoS governors, consistent with v28 behavior.
	if uint64(len(req.GetProof())) > params.MaxUltraHonkProofSizeBytes {
		return nil, errors.Wrapf(
			types.ErrProofTooLarge,
			"proof size %d > max %d bytes",
			len(req.GetProof()),
			params.MaxUltraHonkProofSizeBytes,
		)
	}

	// UltraHonk public inputs are provided as raw bytes.
	if uint64(len(req.GetPublicInputs())) > params.MaxUltraHonkPublicInputSizeBytes {
		return nil, errors.Wrapf(
			types.ErrPublicInputsTooLarge,
			"public inputs size %d > max %d bytes",
			len(req.GetPublicInputs()),
			params.MaxUltraHonkPublicInputSizeBytes,
		)
	}

	// Resolve vkey by name or ID (prefer name when both are set, same as Groth16 verify-proof)
	var vkey types.VKey
	switch {
	case req.GetVkeyName() != "":
		vkey, err = q.GetVKeyByName(c, req.GetVkeyName())
	case req.GetVkeyId() != 0:
		vkey, err = q.GetVKeyByID(c, req.GetVkeyId())
	default:
		return nil, errors.Wrap(types.ErrInvalidRequest, "either vkey_name or vkey_id must be provided")
	}
	if err != nil {
		return nil, err
	}

	if vkey.ProofSystem != types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK {
		proofSystem := vkey.ProofSystem
		if proofSystem == 0 {
			proofSystem = types.ProofSystem_PROOF_SYSTEM_GROTH16
		}
		return nil, errors.Wrapf(types.ErrInvalidRequest, "verification key is not an UltraHonk key (proof_system=%v)", proofSystem)
	}

	publicInputs := req.GetPublicInputs()
	if len(publicInputs)%barretenberg.FieldElementSize != 0 {
		return nil, errors.Wrapf(types.ErrInvalidRequest, "public_inputs length %d is not a multiple of %d", len(publicInputs), barretenberg.FieldElementSize)
	}
	numChunks := len(publicInputs) / barretenberg.FieldElementSize
	chunks := make([][]byte, numChunks)
	for i := 0; i < numChunks; i++ {
		start := i * barretenberg.FieldElementSize
		chunks[i] = publicInputs[start : start+barretenberg.FieldElementSize]
	}

	// Wrap all Barretenberg CGo calls with panic recovery.
	// A panic in the C++ layer propagates as a Go panic through CGo; while a
	// true SIGSEGV cannot be caught here, Go-level panics from the CGo wrapper
	// (e.g. nil-dereference, bounds check) are recoverable and must not crash
	// the validator.
	var (
		resp    *types.ProofVerifyUltraHonkResponse
		callErr error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				q.logger.Error("panic during ultrahonk verification", "panic", r)
				callErr = status.Error(codes.Internal, "internal error during proof verification")
			}
		}()

		vk, e := barretenberg.ParseVerificationKey(vkey.KeyBytes)
		if e != nil {
			callErr = errors.Wrapf(types.ErrInvalidVKey, "ultrahonk vkey: %v", e)
			return
		}
		defer vk.Close()

		bproof, e := barretenberg.ParseProof(req.GetProof())
		if e != nil {
			callErr = errors.Wrapf(types.ErrInvalidRequest, "proof: %v", e)
			return
		}

		verifier, e := barretenberg.NewVerifier(vk)
		if e != nil {
			callErr = e
			return
		}
		defer verifier.Close()

		verified, e := verifier.VerifyWithBytes(bproof, chunks)
		if e != nil {
			if goerrors.Is(e, barretenberg.ErrVerificationFailed) ||
				goerrors.Is(e, barretenberg.ErrInvalidPublicInputs) ||
				goerrors.Is(e, barretenberg.ErrInternal) {
				resp = &types.ProofVerifyUltraHonkResponse{Verified: false}
				return
			}
			callErr = errors.Wrapf(types.ErrInvalidRequest, "verification: %v", e)
			return
		}
		resp = &types.ProofVerifyUltraHonkResponse{Verified: verified}
	}()
	if callErr != nil {
		return nil, callErr
	}
	return resp, nil
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

func (q Querier) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.GetParams(goCtx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}
