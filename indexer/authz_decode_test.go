package indexer

import (
	"testing"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestAuthzDecode(t *testing.T) {
	// This test creates a proper collections-encoded key and verifies it can be decoded
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	msgType := "/cosmos.bank.v1beta1.MsgSend"

	// Encode using collections codec
	codec := collections.TripleKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey, collections.StringKey)
	triple := collections.Join3(granter, grantee, msgType)
	size := codec.Size(triple)
	buf := make([]byte, size)
	_, err := codec.Encode(buf, triple)
	if err != nil {
		t.Fatal(err)
	}

	// Add prefix
	key := append([]byte{0x01}, buf...)

	// Decode using our function
	decodedGranter, decodedGrantee, decodedMsgType := parseGrantStoreKey(key)

	if decodedGranter.String() != granter.String() {
		t.Errorf("granter mismatch: got %s, want %s", decodedGranter.String(), granter.String())
	}
	if decodedGrantee.String() != grantee.String() {
		t.Errorf("grantee mismatch: got %s, want %s", decodedGrantee.String(), grantee.String())
	}
	if decodedMsgType != msgType {
		t.Errorf("msgType mismatch: got %q (len=%d), want %q (len=%d)", decodedMsgType, len(decodedMsgType), msgType, len(msgType))
		t.Logf("Key hex: %x", key)
		t.Logf("Buf hex: %x", buf)
	}
}
