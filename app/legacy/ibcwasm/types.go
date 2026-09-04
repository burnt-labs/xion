package ibcwasm

import (
	"errors"
	"fmt"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// Wasm is the 02-client client type of light clients created by the 08-wasm
// module. It doubles as the client-identifier prefix, and core IBC genesis
// validation rejects a client whose state disagrees with its identifier, so this
// must stay "08-wasm".
const Wasm = "08-wasm"

// checksumLen is the length of a SHA-256 digest, the only checksum 08-wasm ever
// stored.
const checksumLen = 32

var (
	_ exported.ClientState    = (*ClientState)(nil)
	_ exported.ConsensusState = (*ConsensusState)(nil)
	_ exported.ClientMessage  = (*ClientMessage)(nil)
)

// ClientType implements exported.ClientState.
func (ClientState) ClientType() string {
	return Wasm
}

// Validate implements exported.ClientState. It reproduces the checks 08-wasm
// applied, so that a genesis exported from a chain still holding these records
// passes 02-client genesis validation unchanged.
func (cs ClientState) Validate() error {
	if len(cs.Data) == 0 {
		return errors.New("wasm client state data cannot be empty")
	}

	if len(cs.Checksum) != checksumLen {
		return fmt.Errorf("wasm client state checksum: expected %d bytes, got %d", checksumLen, len(cs.Checksum))
	}

	return nil
}

// ClientType implements exported.ConsensusState.
func (ConsensusState) ClientType() string {
	return Wasm
}

// GetTimestamp implements exported.ConsensusState. 08-wasm kept the timestamp
// inside the opaque contract payload, where core IBC cannot read it, and so
// always reported zero here.
func (ConsensusState) GetTimestamp() uint64 {
	return 0
}

// ValidateBasic implements exported.ConsensusState.
func (cs ConsensusState) ValidateBasic() error {
	if len(cs.Data) == 0 {
		return errors.New("wasm consensus state data cannot be empty")
	}

	return nil
}

// ClientType implements exported.ClientMessage.
func (ClientMessage) ClientType() string {
	return Wasm
}

// ValidateBasic implements exported.ClientMessage.
func (c ClientMessage) ValidateBasic() error {
	if len(c.Data) == 0 {
		return errors.New("wasm client message data cannot be empty")
	}

	return nil
}
