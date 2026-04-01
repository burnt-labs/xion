package keeper

import (
	"bytes"
	"context"
	"math/big"
	"strings"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/vocdoni/circom2gnark/parser"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/errors"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/zk/types"
)

type Keeper struct {
	cdc codec.BinaryCodec

	logger log.Logger

	// state management
	Schema        collections.Schema
	VKeys         collections.Map[uint64, types.VKey]
	NextVKeyID    collections.Sequence
	VKeyNameIndex collections.Map[string, uint64]
	Params        collections.Item[types.Params]

	authority string
}

// NewKeeper creates a new Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	logger log.Logger,
	authority string,
) Keeper {
	logger = logger.With(log.ModuleKey, "x/"+types.ModuleName)

	sb := collections.NewSchemaBuilder(storeService)

	if authority == "" {
		authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	}

	k := Keeper{
		cdc:    cdc,
		logger: logger,
		VKeys: collections.NewMap(
			sb,
			types.VKeyPrefix,
			"vkeys",
			collections.Uint64Key,
			codec.CollValue[types.VKey](cdc)),

		NextVKeyID: collections.NewSequence(
			sb,
			types.VkeySeqPrefix,
			"vkey_sequence"),
		VKeyNameIndex: collections.NewMap(
			sb,
			types.VkeyNameIndexPrefix, "vkey_name_index",
			collections.StringKey,
			collections.Uint64Value),
		Params: collections.NewItem(
			sb,
			types.ParamsKey,
			"params",
			codec.CollValue[types.Params](cdc),
		),
		authority: authority,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.Schema = schema

	return k
}

func (k Keeper) Logger() log.Logger {
	return k.logger
}

// InitGenesis initializes the module's state from a genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	params := gs.Params
	if params == (types.Params{}) {
		params = types.DefaultParams()
	}

	// Backfill newly-added Groth16 and UltraHonk proof/public-input size params when upgrading old genesis files.
	params = params.WithMaxLimitDefaults()

	if err := params.Validate(); err != nil {
		panic(err)
	}

	if err := k.SetParams(ctx, params); err != nil {
		panic(err)
	}

	// Import all vkeys
	for _, vkeyWithID := range gs.Vkeys {
		if err := types.ValidateVKeyBytes(vkeyWithID.Vkey.KeyBytes, params.MaxVkeySizeBytes); err != nil {
			panic(err)
		}

		vkey := vkeyWithID.Vkey
		if vkey.Authority == "" {
			vkey.Authority = k.authority
		}

		// Set the vkey
		if err := k.VKeys.Set(ctx, vkeyWithID.Id, vkey); err != nil {
			panic(err)
		}

		// Set the name index
		if err := k.VKeyNameIndex.Set(ctx, vkey.Name, vkeyWithID.Id); err != nil {
			panic(err)
		}

		// Update the sequence to be after the highest ID
		currentSeq, err := k.NextVKeyID.Peek(ctx)
		if err != nil {
			panic(err)
		}
		if vkeyWithID.Id >= currentSeq {
			// Set the sequence to be one more than the highest ID
			if err := k.NextVKeyID.Set(ctx, vkeyWithID.Id+1); err != nil {
				panic(err)
			}
		}
	}
}

// ExportGenesis returns the module's exported genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	vkeys, err := k.ListVKeys(ctx)
	if err != nil {
		panic(err)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	// Convert to VKeyWithID format
	vkeysWithID := make([]types.VKeyWithID, 0, len(vkeys))
	for _, vkey := range vkeys {
		// Get the ID from the name index
		id, err := k.VKeyNameIndex.Get(ctx, vkey.Name)
		if err != nil {
			panic(err)
		}

		vkeysWithID = append(vkeysWithID, types.VKeyWithID{
			Id:   id,
			Vkey: vkey,
		})
	}

	return &types.GenesisState{
		Vkeys:  vkeysWithID,
		Params: params,
	}
}

// GetParams returns the current zk module parameters, defaulting when unset.
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return types.DefaultParams(), nil
		}
		return types.Params{}, err
	}

	// Ensure newly-added fields are always populated with sane defaults.
	return params.WithMaxLimitDefaults(), nil
}

// SetParams validates and persists zk module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	// Backfill newly-added max limit params (e.g. Groth16 and UltraHonk) when not provided.
	params = params.WithMaxLimitDefaults()

	if err := params.Validate(); err != nil {
		return err
	}

	return k.Params.Set(ctx, params)
}

func (k Keeper) ensureVKeySize(params types.Params, size int) (uint64, error) {
	gasCost, err := params.GasCostForSize(uint64(size))
	return gasCost, err
}

// bn254ScalarPrime is the BN254 scalar field modulus r. All Groth16 public inputs
// must be strictly less than this value — circom2gnark's SetBigInt silently reduces
// values >= p modulo p, so p+x would verify identically to x.
var bn254ScalarPrime, _ = new(big.Int).SetString(
	"21888242871839275222246405745257275088548364400416034343698204186575808495617", 10)

func (k *Keeper) Verify(ctx context.Context, proof *parser.CircomProof, vkey *parser.CircomVerificationKey, inputs *[]string) (bool, error) {
	// Validate all public inputs are canonical BN254 scalar field elements.
	// This check must live here so that ALL callers (ProofVerify query, DKIM
	// Authenticate, and any future callers) are protected — not just the query layer.
	for i, inp := range *inputs {
		s := inp
		base := 10
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			s = s[2:]
			base = 16
		}
		v, ok := new(big.Int).SetString(s, base)
		if !ok || v.Sign() < 0 || v.Cmp(bn254ScalarPrime) >= 0 {
			return false, errors.Wrapf(types.ErrInvalidRequest,
				"public input[%d] is not a canonical BN254 scalar field element", i)
		}
	}

	// Wrap gnark calls with panic recovery — circom2gnark/gnark may panic on
	// malformed proofs or VKeys that pass JSON parsing but have invalid curve points.
	var (
		verified  bool
		verifyErr error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				k.logger.Error("panic during groth16 verification", "panic", r)
				verifyErr = errors.Wrap(types.ErrInvalidRequest, "internal error during proof verification")
			}
		}()
		gnarkProof, err := parser.ConvertCircomToGnark(vkey, proof, *inputs)
		if err != nil {
			verifyErr = err
			return
		}
		verified, verifyErr = parser.VerifyProof(gnarkProof)
	}()
	return verified, verifyErr
}

// VerifyGnark verifies a gnark native Groth16 proof (BN254) using the provided vkey bytes and public inputs.
// proofBytes: serialized groth16.Proof (binary, gnark native format)
// vkeyBytes: serialized groth16.VerifyingKey (binary, gnark native format)
// publicInputsBytes: serialized public witness (gnark witness binary format)
//
// The public inputs must be serialized using gnark's witness.MarshalBinary() on the public part
// of the witness (obtained via witness.Public()).
func (k *Keeper) VerifyGnark(ctx context.Context, proofBytes []byte, vkeyBytes []byte, publicInputsBytes []byte) (bool, error) {
	// Deserialize verification key
	vk := groth16.NewVerifyingKey(ecc.BN254)
	if _, err := vk.ReadFrom(bytes.NewReader(vkeyBytes)); err != nil {
		return false, errors.Wrapf(types.ErrInvalidVKey, "failed to parse gnark vkey: %v", err)
	}

	// Deserialize proof
	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofBytes)); err != nil {
		return false, errors.Wrapf(types.ErrInvalidRequest, "failed to parse gnark proof: %v", err)
	}

	// Create and unmarshal public witness
	publicWitness, err := witness.New(ecc.BN254.ScalarField())
	if err != nil {
		return false, errors.Wrapf(types.ErrInvalidRequest, "failed to create witness: %v", err)
	}

	if err := publicWitness.UnmarshalBinary(publicInputsBytes); err != nil {
		return false, errors.Wrapf(types.ErrInvalidRequest, "failed to unmarshal public inputs: %v", err)
	}

	// Verify the proof
	if err := groth16.Verify(proof, vk, publicWitness); err != nil {
		// Verification failed - this is not an error, just means the proof is invalid
		return false, nil
	}

	return true, nil
}

// AddVKey adds a new verification key to the store.
// keyBytes is Groth16/Circom JSON (proofSystem groth16) or Barretenberg binary (proofSystem ultrahonk).
// proofSystem should be types.ProofSystem_PROOF_SYSTEM_GROTH16 or types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK; unspecified defaults to groth16.
func (k Keeper) AddVKey(ctx sdk.Context, authority string, name string, keyBytes []byte, description string, proofSystem types.ProofSystem) (uint64, error) {
	if proofSystem == types.ProofSystem_PROOF_SYSTEM_UNSPECIFIED {
		proofSystem = types.ProofSystem_PROOF_SYSTEM_GROTH16
	}

	// Check if name already exists
	has, err := k.VKeyNameIndex.Has(ctx, name)
	if err != nil {
		return 0, err
	}
	if has {
		return 0, errors.Wrapf(types.ErrVKeyExists, "verification key with name '%s' already exists", name)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	gasCost, err := k.ensureVKeySize(params, len(keyBytes))
	if err != nil {
		return 0, err
	}
	// charge gas for vkey size
	ctx.GasMeter().ConsumeGas(gasCost, "zk/AddVKey: vkey size cost")

	if err := types.ValidateVKeyForProofSystem(keyBytes, params.MaxVkeySizeBytes, proofSystem); err != nil {
		return 0, errors.Wrapf(types.ErrInvalidVKey, "vkey validation: %v", err)
	}

	// Generate new ID
	id, err := k.NextVKeyID.Next(ctx)
	if err != nil {
		return 0, errors.Wrap(types.ErrIncreaseSequenceID, err.Error())
	}

	// Create VKey
	vkey := &types.VKey{
		KeyBytes:    keyBytes,
		Name:        name,
		Description: description,
		Authority:   authority,
		ProofSystem: proofSystem,
	}

	// Store vkey
	if err := k.VKeys.Set(ctx, id, *vkey); err != nil {
		return 0, err
	}

	// Update name index
	if err := k.VKeyNameIndex.Set(ctx, name, id); err != nil {
		return 0, err
	}

	return id, nil
}

// GetVKeyByID retrieves a verification key by its numeric ID
func (k Keeper) GetVKeyByID(ctx context.Context, id uint64) (types.VKey, error) {
	vkey, err := k.VKeys.Get(ctx, id)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return types.VKey{}, errors.Wrapf(types.ErrVKeyNotFound, "verification key with ID %d not found", id)
		}
		return types.VKey{}, err
	}
	return vkey, nil
}

// GetVKeyByName retrieves a verification key by its name
func (k Keeper) GetVKeyByName(ctx context.Context, name string) (types.VKey, error) {
	id, err := k.VKeyNameIndex.Get(ctx, name)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return types.VKey{}, errors.Wrapf(types.ErrVKeyNotFound, "verification key with name '%s' not found", name)
		}
		return types.VKey{}, err
	}

	return k.GetVKeyByID(ctx, id)
}

// GetCircomVKeyByName retrieves and unmarshals a verification key for use in verification
// This is the method you'll use in ProofVerify
func (k Keeper) GetCircomVKeyByName(ctx context.Context, name string) (*parser.CircomVerificationKey, error) {
	vkey, err := k.GetVKeyByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return types.UnmarshalVKey(&vkey)
}

// GetCircomVKeyByID retrieves and unmarshals a verification key for use in verification
func (k Keeper) GetCircomVKeyByID(ctx context.Context, id uint64) (*parser.CircomVerificationKey, error) {
	vkey, err := k.GetVKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return types.UnmarshalVKey(&vkey)
}

// UpdateVKey updates an existing verification key.
// proofSystem should be types.ProofSystem_PROOF_SYSTEM_GROTH16 or types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK; unspecified defaults to groth16.
func (k Keeper) UpdateVKey(ctx sdk.Context, authority string, name string, keyBytes []byte, description string, proofSystem types.ProofSystem) error {
	if proofSystem == types.ProofSystem_PROOF_SYSTEM_UNSPECIFIED {
		proofSystem = types.ProofSystem_PROOF_SYSTEM_GROTH16
	}

	// Get existing ID
	id, err := k.VKeyNameIndex.Get(ctx, name)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return errors.Wrapf(types.ErrVKeyNotFound, "verification key '%s' not found", name)
		}
		return err
	}

	storedVKey, err := k.VKeys.Get(ctx, id)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return errors.Wrapf(types.ErrVKeyNotFound, "verification key '%s' not found", name)
		}
		return err
	}

	storedAuthority := storedVKey.Authority
	if storedAuthority == "" {
		storedAuthority = k.authority
	}

	if storedAuthority != authority {
		return errors.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", storedAuthority, authority)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	gasCost, err := k.ensureVKeySize(params, len(keyBytes))
	if err != nil {
		return err
	}
	// charge gas for vkey size
	ctx.GasMeter().ConsumeGas(gasCost, "zk/UpdateVKey: vkey size cost")

	if err := types.ValidateVKeyForProofSystem(keyBytes, params.MaxVkeySizeBytes, proofSystem); err != nil {
		return errors.Wrapf(types.ErrInvalidVKey, "vkey validation: %v", err)
	}

	// Update vkey
	updatedVKey := types.VKey{
		KeyBytes:    keyBytes,
		Name:        name,
		Description: description,
		Authority:   storedAuthority,
		ProofSystem: proofSystem,
	}

	if err := k.VKeys.Set(ctx, id, updatedVKey); err != nil {
		return err
	}

	return nil
}

// RemoveVKey removes a verification key by name
func (k Keeper) RemoveVKey(ctx context.Context, authority string, name string) error {
	// Get the ID from the name index
	id, err := k.VKeyNameIndex.Get(ctx, name)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return errors.Wrapf(types.ErrVKeyNotFound, "verification key '%s' not found", name)
		}
		return err
	}

	storedVKey, err := k.VKeys.Get(ctx, id)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return errors.Wrapf(types.ErrVKeyNotFound, "verification key '%s' not found", name)
		}
		return err
	}

	storedAuthority := storedVKey.Authority
	if storedAuthority == "" {
		storedAuthority = k.authority
	}

	if storedAuthority != authority {
		return errors.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", storedAuthority, authority)
	}

	// Remove from primary storage
	if err := k.VKeys.Remove(ctx, id); err != nil {
		return err
	}

	// Remove from name index
	if err := k.VKeyNameIndex.Remove(ctx, name); err != nil {
		return err
	}

	return nil
}

// ListVKeys returns all verification keys
func (k Keeper) ListVKeys(ctx context.Context) ([]types.VKey, error) {
	var vkeys []types.VKey

	err := k.VKeys.Walk(ctx, nil, func(id uint64, vkey types.VKey) (bool, error) {
		vkeys = append(vkeys, vkey)
		return false, nil
	})

	return vkeys, err
}

// HasVKey checks if a verification key exists by name
// This is a read operation - no authority check needed
func (k Keeper) HasVKey(ctx context.Context, name string) (bool, error) {
	return k.VKeyNameIndex.Has(ctx, name)
}

// IterateVKeys iterates over all verification keys with a callback
// This is a read operation - no authority check needed
func (k Keeper) IterateVKeys(ctx context.Context, cb func(id uint64, vkey types.VKey) (stop bool, err error)) error {
	return k.VKeys.Walk(ctx, nil, cb)
}

// GetAuthority returns the module's authority address
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Add these getter methods to your keeper if they don't exist
func (k Keeper) GetCodec() codec.BinaryCodec {
	return k.cdc
}
