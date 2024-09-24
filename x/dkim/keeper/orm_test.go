package keeper_test

import (
	"testing"

	queryv1beta1 "cosmossdk.io/api/cosmos/base/query/v1beta1"
	ormlist "cosmossdk.io/orm/model/ormlist"
	apiv1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
	dkimKeeper "github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestORM(t *testing.T) {
	f := SetupTest(t)

	dt := f.k.OrmDB.DkimPubKeyTable()
	domain := "xion.burnt.com"
	pubKey := "xion1234567890"
	selector := "zkemail"
	version := types.Version_DKIM1
	keyType := types.KeyType_RSA

	isSaved, err := dkimKeeper.SaveDkimPubKey(f.ctx, &types.DkimPubKey{
		Domain:   domain,
		PubKey:   pubKey,
		Selector: selector,
		Version:  version,
		KeyType:  keyType,
	}, f.k.OrmDB)

	require.NoError(t, err)
	require.True(t, isSaved)

	d, err := dt.Has(f.ctx, domain, selector)
	require.NoError(t, err)
	require.True(t, d)

	res, err := dt.Get(f.ctx, domain, selector)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, pubKey, res.PubKey)
	require.EqualValues(t, types.Version_DKIM1, res.Version)
	require.EqualValues(t, types.KeyType_RSA, res.KeyType)
}

func TestORMMultipleInsert(t *testing.T) {
	f := SetupTest(t)

	dt := f.k.OrmDB.DkimPubKeyTable()
	count := 10
	var dkimPubKeys []types.DkimPubKey
	for i := 0; i < count; i++ {
		domain := "xion.burnt.com"
		pubKey := "xion1234567890"
		selector := uuid.NewString()
		version := types.Version_DKIM1
		keyType := types.KeyType_RSA

		dkimPubKeys = append(dkimPubKeys, types.DkimPubKey{
			Domain:   domain,
			PubKey:   pubKey,
			Selector: selector,
			Version:  version,
			KeyType:  keyType,
		})
	}
	isSaved, err := dkimKeeper.SaveDkimPubKeys(f.ctx, dkimPubKeys, f.k.OrmDB)
	require.NoError(t, err)
	require.True(t, isSaved)
	allDkimPubKeys, err := dt.List(f.ctx, apiv1.DkimPubKeyDomainSelectorIndexKey{}, ormlist.Paginate(&queryv1beta1.PageRequest{Limit: 100, CountTotal: true}))
	require.NoError(t, err)
	require.NotNil(t, allDkimPubKeys)
	for allDkimPubKeys.Next() {
		_, err := allDkimPubKeys.Value()
		require.NoError(t, err)
	}
	require.EqualValues(t, count, allDkimPubKeys.PageResponse().Total)
}
