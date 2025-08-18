package types

import "fmt"

// DefaultGenesis returns the incentive module's default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: Params{
			OsmosisQueryTwapPath: DefaultOsmosisQueryTwapPath,
			NativeIbcedInOsmosis: "ibc/C053D637CCA2A2BA030E2C5EE1B28A16F71CCB0E45E8BE52766DC1B241B77878",
			ChainName:            DefaultChainName,
		},
		Epochs: []EpochInfo{NewGenesisEpochInfo(DefaultQueryEpochIdentifier, DefaultQueryPeriod), NewGenesisEpochInfo(DefaultSwapEpochIdentifier, DefaultSwapPeriod)},
		PortId: IBCPortID,
	}
}

// Validate performs basic genesis state validation, returning an error upon any failure.
func (gs GenesisState) Validate() error {
	err := gs.Params.Validate()
	if err != nil {
		return fmt.Errorf("invalid params %s", err)
	}

	// Validate epochs genesis
	for _, epoch := range gs.Epochs {
		err := epoch.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}
