package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestComputePoseidonHash(t *testing.T) {
	testXDkimPubKey := types.DkimPubKey{
		PubKey: "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
	}
	testGDkimPubKey := types.DkimPubKey{
		PubKey: "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAntvSKT1hkqhKe0xcaZ0x+QbouDsJuBfby/S82jxsoC/SodmfmVs2D1KAH3mi1AqdMdU12h2VfETeOJkgGYq5ljd996AJ7ud2SyOLQmlhaNHH7Lx+Mdab8/zDN1SdxPARDgcM7AsRECHwQ15R20FaKUABGu4NTbR2fDKnYwiq5jQyBkLWP+LgGOgfUF4T4HZb2PY2bQtEP6QeqOtcW4rrsH24L7XhD+HSZb1hsitrE0VPbhJzxDwI4JF815XMnSVjZgYUXP8CxI1Y0FONlqtQYgsorZ9apoW1KPQe8brSSlRsi9sXB/tu56LmG7tEDNmrZ5XUwQYUUADBOu7t1niwXwIDAQAB",
	}
	xPubkeyHash, err := types.ComputePoseidonHash(testXDkimPubKey.PubKey)
	require.NoError(t, err)
	gPubkeyHash, err := types.ComputePoseidonHash(testGDkimPubKey.PubKey)
	require.NoError(t, err)
	require.Equal(t, xPubkeyHash.String(), "1983664618407009423875829639306275185491946247764487749439145140682408188330")
	require.Equal(t, gPubkeyHash.String(), "6632353713085157925504008443078919716322386156160602218536961028046468237192")
}

func TestDkimPubKeyValidate(t *testing.T) {
	valid := &types.DkimPubKey{
		Domain: "https://example.com",
		PubKey: "Zm9v",
	}
	require.NoError(t, valid.Validate())

	invalidDomain := *valid
	invalidDomain.Domain = "ht tp://bad"
	err := invalidDomain.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "dkim url key parsing failed")

	invalidPubKey := *valid
	invalidPubKey.PubKey = "***"
	err = invalidPubKey.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "dkim public key decoding failed")
}
