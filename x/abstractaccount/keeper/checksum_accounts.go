package keeper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

// ChecksumAccountBackfill counts what BackfillChecksumAccountAddresses did with
// the contracts of one code ID.
type ChecksumAccountBackfill struct {
	// Registered entries written to the (sender, salt) registry.
	Registered int
	// Skipped contracts: not an abstract account, no recognised authenticator in
	// the INIT message, an address that does not derive from the code checksum,
	// or a (sender, salt) slot that is already taken.
	Skipped int
}

// BackfillChecksumAccountAddresses writes the (sender, salt) registry entry for
// abstract accounts registered before the registry existed at an address
// derived from their code's checksum rather than from the module's address
// derivation hash. For those accounts AccountAddress otherwise predicts the
// fixed-hash address as unregistered, and RegisterAccount would accept a second
// account for the same (sender, salt).
//
// Every contract instantiated with codeID is visited. Its salt is recovered
// from the authenticator in its INIT message (see checksumAccountSalt), and an
// entry is written only when the contract is an abstract account, its address
// equals the instantiate2 address of (checksum, creator, salt) and the slot is
// empty. Nothing is taken on trust, so a contract that fails any check is
// skipped, never registered. The call refuses to run when the stored code's
// checksum is not the expected one.
func (k Keeper) BackfillChecksumAccountAddresses(
	ctx sdk.Context,
	codeID uint64,
	checksum []byte,
) (ChecksumAccountBackfill, error) {
	var result ChecksumAccountBackfill

	codeInfo := k.vk.GetCodeInfo(ctx, codeID)
	if codeInfo == nil {
		return result, types.ErrCodeIDNotFound.Wrapf("checksum account code ID %d", codeID)
	}
	if !bytes.Equal(codeInfo.CodeHash, checksum) {
		return result, types.ErrInvalidAccountAddressRegistry.Wrapf(
			"code ID %d checksum %X, expected %X", codeID, codeInfo.CodeHash, checksum,
		)
	}

	var contracts []sdk.AccAddress
	k.vk.IterateContractsByCode(ctx, codeID, func(address sdk.AccAddress) bool {
		contracts = append(contracts, address)
		return false
	})

	for _, address := range contracts {
		if k.backfillChecksumAccount(ctx, codeID, checksum, address) {
			result.Registered++
		} else {
			result.Skipped++
		}
	}

	return result, nil
}

func (k Keeper) backfillChecksumAccount(
	ctx sdk.Context,
	codeID uint64,
	checksum []byte,
	address sdk.AccAddress,
) bool {
	if !k.IsAbstractAccount(ctx, address) {
		return false
	}
	init, found := contractInitEntry(k.vk.GetContractHistory(ctx, address))
	if !found || init.CodeID != codeID {
		return false
	}
	salt, ok := checksumAccountSalt(init.Msg)
	if !ok {
		return false
	}
	creator, err := sdk.AccAddressFromBech32(k.vk.GetContractInfo(ctx, address).Creator)
	if err != nil {
		return false
	}
	if !wasmkeeper.BuildContractAddressPredictable(checksum, creator, salt, nil).Equals(address) {
		return false
	}
	if _, taken := k.GetAccountAddress(ctx, creator, salt); taken {
		return false
	}

	k.SetAccountAddress(ctx, creator, salt, address)
	return true
}

func contractInitEntry(history []wasmtypes.ContractCodeHistoryEntry) (wasmtypes.ContractCodeHistoryEntry, bool) {
	for _, entry := range history {
		if entry.Operation == wasmtypes.ContractCodeHistoryOperationTypeInit {
			return entry, true
		}
	}
	return wasmtypes.ContractCodeHistoryEntry{}, false
}

// derivesFromInitCodeChecksum reports whether address is the instantiate2
// address of (checksum of the code the contract was instantiated with, sender,
// salt): the form of a registry entry written by
// BackfillChecksumAccountAddresses.
func (k Keeper) derivesFromInitCodeChecksum(ctx sdk.Context, sender sdk.AccAddress, salt []byte, address sdk.AccAddress) bool {
	init, found := contractInitEntry(k.vk.GetContractHistory(ctx, address))
	if !found {
		return false
	}
	codeInfo := k.vk.GetCodeInfo(ctx, init.CodeID)
	if codeInfo == nil || len(codeInfo.CodeHash) != wasmtypes.ContractAddrLen {
		return false
	}
	return wasmkeeper.BuildContractAddressPredictable(codeInfo.CodeHash, sender, salt, nil).Equals(address)
}

// checksumAccountSalt recovers the salt an account was registered with from
// the authenticator in its INIT message, following the convention of the
// account-abstraction API that registered these accounts: sha256 of the
// Ethereum address bytes (EthWallet), of the public key string as sent
// (Secp256K1), or of "aud.sub" (Jwt). Any other message yields no salt.
func checksumAccountSalt(msg []byte) ([]byte, bool) {
	var init struct {
		Authenticator map[string]json.RawMessage `json:"authenticator"`
	}
	if err := json.Unmarshal(msg, &init); err != nil || len(init.Authenticator) != 1 {
		return nil, false
	}

	for kind, raw := range init.Authenticator {
		var auth struct {
			Address string `json:"address"`
			Pubkey  string `json:"pubkey"`
			Aud     string `json:"aud"`
			Sub     string `json:"sub"`
		}
		if err := json.Unmarshal(raw, &auth); err != nil {
			return nil, false
		}

		var credential []byte
		switch kind {
		case "EthWallet":
			address, err := hex.DecodeString(strings.TrimPrefix(strings.ToLower(auth.Address), "0x"))
			if err != nil || len(address) != 20 {
				return nil, false
			}
			credential = address
		case "Secp256K1":
			credential = []byte(auth.Pubkey)
		case "Jwt":
			if auth.Aud == "" || auth.Sub == "" {
				return nil, false
			}
			credential = []byte(auth.Aud + "." + auth.Sub)
		default:
			return nil, false
		}
		if len(credential) == 0 {
			return nil, false
		}

		salt := sha256.Sum256(credential)
		return salt[:], true
	}

	return nil, false
}
