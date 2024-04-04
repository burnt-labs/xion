package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

var (
	PlatformPercentageKey = []byte{0x00}
	// Keys for store prefixes
	// Items are stored with the following key: values
	//
	// - 0x01<grant_Bytes>: Grant
	// - 0x02<grant_expiration_Bytes>: GrantQueueItem
	GrantKey         = []byte{0x01} // prefix for each key
	GrantQueuePrefix = []byte{0x02}
)

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "xion"

	// StoreKey is the store key string for oracle
	StoreKey = ModuleName

	// RouterKey is the message route for oracle
	RouterKey = ModuleName

	// QuerierRoute is the querier route for oracle
	QuerierRoute = ModuleName
)

// grantStoreKey - return authorization store key
// Items are stored with the following key: values
//
// - 0x01<granterAddressLen (1 Byte)><granterAddress_Bytes><granteeAddressLen (1 Byte)><granteeAddress_Bytes><msgType_Bytes>: Grant
func GrantStoreKey(grantee sdk.AccAddress, granter sdk.AccAddress, msgType string) []byte {
	m := []byte(msgType)
	granter = address.MustLengthPrefix(granter)
	grantee = address.MustLengthPrefix(grantee)
	key := sdk.AppendLengthPrefixedBytes(GrantKey, granter, grantee, m)

	return key
}
