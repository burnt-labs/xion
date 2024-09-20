package keeper_test

import (
	"testing"

	apiv1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
	"github.com/stretchr/testify/require"
)

func TestORM(t *testing.T) {
	f := SetupTest(t)

	dt := f.k.OrmDB.DkimPubKeyTable()
	domain := "xion.burnt.com"
	pubKey := "xion1234567890"
	selector := "zkemail"

	err := dt.Insert(f.ctx, &apiv1.DkimPubKey{
		Domain:   domain,
		PubKey:   pubKey,
		Selector: selector,
	})
	require.NoError(t, err)

	d, err := dt.Has(f.ctx, selector, domain)
	require.NoError(t, err)
	require.True(t, d)

	res, err := dt.Get(f.ctx, selector, domain)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, pubKey, res.PubKey)
}
