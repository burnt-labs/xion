package types

import bytes "bytes"

// d line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// d line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	return nil
}

func (d *DkimPubKey) Equal(v interface{}) bool {
	if v == nil {
		return d == nil
	}

	v1, ok := v.(*DkimPubKey)
	if !ok {
		v2, ok := v.(DkimPubKey)
		if ok {
			v1 = &v2
		} else {
			return false
		}
	}
	if v1 == nil {
		return d == nil
	} else if d == nil {
		return false
	}
	if d.Domain != v1.Domain {
		return false
	}
	if d.PubKey != v1.PubKey {
		return false
	}
	if d.Selector != v1.Selector {
		return false
	}
	if d.Version != v1.Version {
		return false
	}
	if d.KeyType != v1.KeyType {
		return false
	}
	if !bytes.Equal(d.PoseidonHash, v1.PoseidonHash) {
		return false
	}
	return true
}
