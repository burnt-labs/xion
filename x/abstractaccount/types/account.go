// Two distinct public-key interfaces are in play here. They declare identical
// method sets, but the Go compiler treats them as separate types, so a value
// satisfying one does not substitute for the other:
//   - github.com/cosmos/cosmos-sdk/crypto/types.PubKey
//   - github.com/cometbft/cometbft/crypto.PubKey

package types

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.AccountI       = (*AbstractAccount)(nil)
	_ cryptotypes.PubKey = (*NilPubKey)(nil)
)

// ------------------------------ AbstractAccount ------------------------------

func NewAbstractAccount(address string, accountNum, seq uint64) *AbstractAccount {
	return &AbstractAccount{
		Address:       address,
		AccountNumber: accountNum,
		Sequence:      seq,
	}
}

func NewAbstractAccountFromAccount(acc sdk.AccountI) *AbstractAccount {
	return NewAbstractAccount(acc.GetAddress().String(), acc.GetAccountNumber(), acc.GetSequence())
}

func (acc *AbstractAccount) GetAddress() sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(acc.Address)
	return addr
}

func (acc *AbstractAccount) SetAddress(addr sdk.AccAddress) error {
	if len(acc.Address) != 0 {
		return errors.New("cannot override AbstractAccount address")
	}

	acc.Address = addr.String()

	return nil
}

func (acc *AbstractAccount) GetPubKey() cryptotypes.PubKey {
	return NewNilPubKey(acc.GetAddress())
}

func (acc *AbstractAccount) SetPubKey(_ cryptotypes.PubKey) error {
	return errors.New("cannot set pubkey for AbstractAccount")
}

func (acc *AbstractAccount) GetAccountNumber() uint64 {
	return acc.AccountNumber
}

func (acc *AbstractAccount) SetAccountNumber(accNumber uint64) error {
	acc.AccountNumber = accNumber

	return nil
}

func (acc *AbstractAccount) GetSequence() uint64 {
	return acc.Sequence
}

func (acc *AbstractAccount) SetSequence(seq uint64) error {
	acc.Sequence = seq

	return nil
}

func (acc *AbstractAccount) Validate() error {
	if len(acc.Address) == 0 {
		return errors.New("address cannot be empty")
	}

	// GetAddress discards the parse error and yields a nil address for a
	// malformed string, so reject it here instead of letting decode and genesis
	// paths admit an account that silently has no address.
	if _, err := sdk.AccAddressFromBech32(acc.Address); err != nil {
		return fmt.Errorf("invalid address %q: %w", acc.Address, err)
	}

	return nil
}

// --------------------------------- NilPubKey ---------------------------------

func NewNilPubKey(bz []byte) *NilPubKey {
	return &NilPubKey{AddressBytes: bz}
}

func (pk *NilPubKey) Address() cryptotypes.Address {
	return cryptotypes.Address(pk.AddressBytes)
}

func (pk *NilPubKey) Bytes() []byte {
	return nil
}

func (pk *NilPubKey) VerifySignature(_, _ []byte) bool {
	panic("NilPubKey.VerifySignature should never be invoked")
}

func (pk *NilPubKey) Equals(other cryptotypes.PubKey) bool {
	otherPk, ok := other.(*NilPubKey)
	if !ok {
		return false
	}

	return bytes.Equal(pk.AddressBytes, otherPk.AddressBytes)
}

func (pk *NilPubKey) Type() string {
	return "/" + proto.MessageName(pk)
}

func (pk *NilPubKey) String() string {
	return sdk.AccAddress(pk.AddressBytes).String()
}
