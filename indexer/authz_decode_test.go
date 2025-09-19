package indexer

import (
	"encoding/hex"
	"log/slog"
	"testing"
)

func TestAuthzDecode(t *testing.T) {
	key := "01323032352d31312d30385431373a35303a32362e30303030303030303014badd8d136c8359cabdd9959c5d56c40a81dac00d2078face6e381ee98acfc1429f7720f84910c23967fee2a17160ad72e009fb0c8e"
	keyBz, err := hex.DecodeString(key)
	if err != nil {
		t.Fatal(err)
	}

	granterAddr, granteeAddr, msgType := parseGrantStoreKey(keyBz)
	slog.Info("granter", "granter", granterAddr.String(), "grantee", granteeAddr.String(), "msgType", msgType)
}
