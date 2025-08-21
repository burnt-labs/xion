package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Feeabs params default values .
const (
	DefaultOsmosisQueryTwapPath = "/osmosis.twap.v1beta1.Query/ArithmeticTwapToNow"
	DefaultChainName            = "feeappd-t1"
	DefaultContractAddress      = ""
)

// Parameter keys store keys.
var (
	KeyOsmosisQueryTwapPath         = []byte("OsmosisQueryTwapPath")
	KeyNativeIbcedInOsmosis         = []byte("NativeIbcedInOsmosis")
	KeyChainName                    = []byte("ChainName")
	KeyIbcTransferChannel           = []byte("IbcTransferChannel")
	KeyIbcQueryIcqChannel           = []byte("IbcQueryIcqChannel")
	KeyOsmosisCrosschainSwapAddress = []byte("OsmosisCrosschainSwapAddress")

	_ paramtypes.ParamSet = &Params{}
)

// ParamTable for lockup module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyOsmosisQueryTwapPath, &p.OsmosisQueryTwapPath, validateString),
		paramtypes.NewParamSetPair(KeyNativeIbcedInOsmosis, &p.NativeIbcedInOsmosis, validateString),
		paramtypes.NewParamSetPair(KeyChainName, &p.ChainName, validateString),
		paramtypes.NewParamSetPair(KeyIbcTransferChannel, &p.IbcTransferChannel, validateString),
		paramtypes.NewParamSetPair(KeyIbcQueryIcqChannel, &p.IbcQueryIcqChannel, validateString),
		paramtypes.NewParamSetPair(KeyOsmosisCrosschainSwapAddress, &p.OsmosisCrosschainSwapAddress, validateString),
	}
}

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
