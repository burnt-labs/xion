package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cli "github.com/burnt-labs/xion/x/dkim/client/cli"
)

func TestGetDkimPubicKey(t *testing.T) {
	testCases := []struct {
		name     string
		domain   string
		selector string
		result   string
	}{
		{
			domain:   "x.com",
			selector: "dkim-202308",
			result:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
		},
		// { //NOTE: test seems to be failing, it seems it can't retrieve the TXT
		//	domain:   "account.netflix.com",
		//	selector: "kk6c473czcop4fqv6yhfgiqupmfz3cm2",
		//	result:   "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCD8gKP5B1x0stqA0NhBw0PbvVjbQ98s07tAovJmUBLk9D/VsjNCVx8WAzZxyKI+lbs9Okua/Knq5kDzO2dxSbus/LaDHCHx7YYqNWL0xdaPCSjFL/sYqX7V4wq4N/OcBoASitk61eGJXVgmEfJBRNfNoi3iHDf9GvpCNBKTHYkewIDAQAB",
		// },
		{
			domain:   "email.slackhq.com",
			selector: "200608",
			result:   "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDGoQCNwAQdJBy23MrShs1EuHqK/dtDC33QrTqgWd9CJmtM3CK2ZiTYugkhcxnkEtGbzg+IJqcDRNkZHyoRezTf6QbinBB2dbyANEuwKI5DVRBFowQOj9zvM3IvxAEboMlb0szUjAoML94HOkKuGuCkdZ1gbVEi3GcVwrIQphal1QIDAQAB",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			pubKey, err := cli.GetDKIMPublicKey(tc.selector, tc.domain)
			require.NoError(t, err)
			require.EqualValues(t, tc.result, pubKey)
		})
	}
}
