package keeper_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	queryv1beta1 "cosmossdk.io/api/cosmos/base/query/v1beta1"
	ormlist "cosmossdk.io/orm/model/ormlist"

	apiv1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
	dkimKeeper "github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestORM(t *testing.T) {
	f := SetupTest(t)

	dt := f.k.OrmDB.DkimPubKeyTable()
	domain := "xion.burnt.com"
	pubKey := "xion1234567890"
	selector := "zkemail"
	version := types.Version_DKIM1
	keyType := types.KeyType_RSA

	isSaved, err := dkimKeeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
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

func CreateNDkimPubKey(t *testing.T, domain string, pubKey string, version types.Version, keyType types.KeyType, count int) []types.DkimPubKey {
	var dkimPubKeys []types.DkimPubKey
	hash, err := types.ComputePoseidonHash(pubKey)
	require.NoError(t, err)
	for i := 0; i < count; i++ {
		selector := uuid.NewString()
		dkimPubKeys = append(dkimPubKeys, types.DkimPubKey{
			Domain:       domain,
			PubKey:       pubKey,
			Selector:     selector,
			Version:      version,
			KeyType:      keyType,
			PoseidonHash: hash.Bytes(),
		})
	}
	return dkimPubKeys
}

func TestORMMultipleInsert(t *testing.T) {
	f := SetupTest(t)

	dt := f.k.OrmDB.DkimPubKeyTable()
	count := 10
	domain := "xion.burnt.com"
	pubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	dkimPubKeys := CreateNDkimPubKey(t, domain, pubKey, types.Version_DKIM1, types.KeyType_RSA, count)
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
