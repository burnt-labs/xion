package feeabs

import "fmt"

// Feeabs params default values .
const (
	DefaultOsmosisQueryTwapPath = "/osmosis.twap.v1beta1.Query/ArithmeticTwapToNow"
	DefaultChainName            = "feeappd-t1"
	DefaultContractAddress      = ""
)

// Validate also validates params info.
func (p Params) Validate() error {
	if err := validateString(p.OsmosisQueryTwapPath); err != nil {
		return err
	}
	if err := validateString(p.NativeIbcedInOsmosis); err != nil {
		return err
	}
	if err := validateString(p.ChainName); err != nil {
		return err
	}
	if err := validateString(p.IbcTransferChannel); err != nil {
		return err
	}
	if err := validateString(p.IbcQueryIcqChannel); err != nil {
		return err
	}
	if err := validateString(p.OsmosisCrosschainSwapAddress); err != nil {
		return err
	}

	return nil
}

func validateString(i interface{}) error {
	_, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type string: %T", i)
	}

	return nil
}

func DefaultParams() Params {
	return Params{
		OsmosisQueryTwapPath: DefaultOsmosisQueryTwapPath,
		ChainName:            DefaultChainName,
		IbcTransferChannel:   "",
		IbcQueryIcqChannel:   "",
		NativeIbcedInOsmosis: "ibc/C053D637CCA2A2BA030E2C5EE1B28A16F71CCB0E45E8BE52766DC1B241B77878",
	}
}
