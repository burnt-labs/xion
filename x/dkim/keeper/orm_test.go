package keeper_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"

	dkimKeeper "github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestORM(t *testing.T) {
	f := SetupTest(t)

	domain := "xion.burnt.com"
	pubKey := "xion1234567890"
	selector := "zkemail"
	poseidonHash := []byte("poseidonHash")
	version := types.Version_DKIM1
	keyType := types.KeyType_RSA

	isSaved, err := dkimKeeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
		Domain:       domain,
		PubKey:       pubKey,
		Selector:     selector,
		PoseidonHash: poseidonHash,
		Version:      version,
		KeyType:      keyType,
	}, &f.k)

	require.NoError(t, err)
	require.True(t, isSaved)

	key := collections.Join(domain, selector)
	has, err := f.k.DkimPubKeys.Has(ctx, key)
	require.NoError(t, err)
	require.True(t, has)

	res, err := f.k.DkimPubKeys.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, pubKey, res.PubKey)
	require.EqualValues(t, poseidonHash, res.PoseidonHash)
	require.EqualValues(t, types.Version_DKIM1, res.Version)
	require.EqualValues(t, types.KeyType_RSA, res.KeyType)
}

func CreateNDkimPubKey(t *testing.T, domain string, pubKey string, version types.Version, keyType types.KeyType, count int) []types.DkimPubKey {
	var dkimPubKeys []types.DkimPubKey
	hash, err := types.ComputePoseidonHash(pubKey)
	require.NoError(t, err)
	for range count {
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

	count := 10
	domain := "xion.burnt.com"
	pubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	dkimPubKeys := CreateNDkimPubKey(t, domain, pubKey, types.Version_DKIM1, types.KeyType_RSA, count)
	isSaved, err := dkimKeeper.SaveDkimPubKeys(f.ctx, dkimPubKeys, &f.k)
	require.NoError(t, err)
	require.True(t, isSaved)

	iter, err := f.k.DkimPubKeys.Iterate(f.ctx, collections.RangeFull())
	require.NoError(t, err)
	defer iter.Close()
	kvs, err := iter.KeyValues()
	require.NoError(t, err)
	require.EqualValues(t, count, len(kvs))
}
