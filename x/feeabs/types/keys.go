package types

const (
	// Module name store the name of the module
	ModuleName = "feeabs"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the msg router key for the feeabs module
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_feeabs"

	// Contract: Coin denoms cannot contain this character
	KeySeparator = "|"
)

type (
	ByPassMsgKey               struct{}
	ByPassExceedMaxGasUsageKey struct{}
	GlobalFeeKey               struct{}
)

var (
	StoreExponentialBackoff = []byte{0x10} // sub store that records the next block to send an ibc query or cross - chain swap

	OsmosisTwapExchangeRate     = []byte{0x01} // Key for the exchange rate of osmosis (to native token)
	KeyChannelID                = []byte{0x02} // Key for IBC channel to osmosis
	KeyHostChainConfigByFeeAbs  = []byte{0x03} // Key for IBC channel to osmosis
	KeyHostChainConfigByOsmosis = []byte{0x04} // Key for IBC channel to osmosis
	KeyPrefixEpoch              = []byte{0x05} // KeyPrefixEpoch defines prefix key for storing epochs.
	KeyTokenDenomPair           = []byte{0x06} // Key store token denom pair on feeabs and osmosis
)

func GetKeyHostZoneConfigByFeeabsIBCDenom(feeabsIbcDenom string) []byte {
	return append(KeyHostChainConfigByFeeAbs, []byte(feeabsIbcDenom)...)
}

func GetKeyHostZoneConfigByOsmosisIBCDenom(osmosisIbcDenom string) []byte {
	return append(KeyHostChainConfigByOsmosis, []byte(osmosisIbcDenom)...)
}

func GetKeyTwapExchangeRate(ibcDenom string) []byte {
	return append(OsmosisTwapExchangeRate, []byte(ibcDenom)...)
}
