package types_test

import (
	"testing"

	"github.com/burnt-labs/xion/x/dkim/types"
	"github.com/stretchr/testify/require"
)

func TestComputePoseidonHash(t *testing.T) {
	testDkimPubKey := types.DkimPubKey{
		Domain: "xion.burnt.com",
		PubKey: `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwe34ubzrMzM9sT0XVkcc
3UXd7W+EHCyHoqn70l2AxXox52lAZzH/UnKwAoO+5qsuP7T9QOifIJ9ddNH9lEQ9
5Y/GdHBsPLGdgSJIs95mXNxscD6MSyejpenMGL9TPQAcxfqY5xPViZ+1wA1qcryj
dZKRqf1f4fpMY+x3b8k7H5Qyf/Smz0sv4xFsx1r+THNIz0rzk2LO3GvE0f1ybp6P
+5eAelYU4mGeZQqsKw/eB20I3jHWEyGrXuvzB67nt6ddI+N2eD5K38wg/aSytOsb
5O+bUSEe7P0zx9ebRRVknCD6uuqG3gSmQmttlD5OrMWSXzrPIXe8eTBaaPd+e/jf
xwIDAQAB
-----END PUBLIC KEY-----
`,
		Selector: "zkemail",
	}
	err := testDkimPubKey.ComputePoseidonHash()
	require.NoError(t, err)
}
