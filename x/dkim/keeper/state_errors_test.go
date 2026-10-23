package keeper_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"

	"github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
)

// undecodableValue is not a valid protobuf encoding: a length-delimited field
// tag with no length byte following it.
var undecodableValue = []byte{0x0a}

// corruptParams overwrites the stored params with bytes the value codec cannot decode.
func (f *TestFixture) corruptParams() {
	f.ctx.KVStore(f.dkimKey).Set(types.ParamsKey.Bytes(), undecodableValue)
}

// corruptDkimPubKey writes undecodable bytes under the map key for domain/selector.
func (f *TestFixture) corruptDkimPubKey(t *testing.T, domain, selector string) {
	t.Helper()
	pair := collections.Join(domain, selector)
	keyCodec := f.k.DkimPubKeys.KeyCodec()
	buf := make([]byte, keyCodec.Size(pair))
	n, err := keyCodec.Encode(buf, pair)
	require.NoError(t, err)
	rawKey := append(append([]byte{}, types.DkimPrefix.Bytes()...), buf[:n]...)
	f.ctx.KVStore(f.dkimKey).Set(rawKey, undecodableValue)
}

// genRSAKeyPair returns a PEM-encoded PKCS1 private key and the base64 DER
// of the matching public key, as stored in a DkimPubKey.
func genRSAKeyPair(t *testing.T) (privPEM []byte, pubKeyB64 string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	return privPEM, base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(&privateKey.PublicKey))
}

func genRSAPrivKeyPEM(t *testing.T) []byte {
	t.Helper()
	privPEM, _ := genRSAKeyPair(t)
	return privPEM
}

func TestGetParamsUndecodable(t *testing.T) {
	f := SetupTest(t)
	f.corruptParams()

	_, err := f.k.GetParams(f.ctx)
	require.Error(t, err)

	res, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
	require.Error(t, err)
	require.Nil(t, res)

	_, err = f.queryServer.Authenticate(f.ctx, &types.QueryAuthenticateRequest{})
	require.Error(t, err)

	_, err = f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
		Authority:   f.govModAddr,
		DkimPubkeys: []types.DkimPubKey{{Domain: "x.com", Selector: "s", PubKey: testPubKey}},
	})
	require.Error(t, err)

	pemKey := genRSAPrivKeyPEM(t)
	_, err = f.msgServer.RevokeDkimPubKey(f.ctx, &types.MsgRevokeDkimPubKey{
		Signer:  f.addrs[0].String(),
		Domain:  "x.com",
		PrivKey: pemKey,
	})
	require.Error(t, err)

	require.Panics(t, func() {
		f.k.ExportGenesis(f.ctx)
	})
}

func TestAuthenticateWithoutParams(t *testing.T) {
	f := SetupTest(t)
	require.NoError(t, f.k.Params.Remove(f.ctx))

	res, err := f.queryServer.Authenticate(f.ctx, &types.QueryAuthenticateRequest{})
	require.ErrorIs(t, err, collections.ErrNotFound)
	require.Nil(t, res)
}

func TestUndecodableDkimPubKeyEntry(t *testing.T) {
	f := SetupTest(t)
	f.corruptDkimPubKey(t, "corrupt.com", "s1")

	res, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{Domain: "corrupt.com"})
	require.Error(t, err)
	require.Nil(t, res)

	pemKey := genRSAPrivKeyPEM(t)
	_, err = f.msgServer.RevokeDkimPubKey(f.ctx, &types.MsgRevokeDkimPubKey{
		Signer:  f.addrs[0].String(),
		Domain:  "corrupt.com",
		PrivKey: pemKey,
	})
	require.Error(t, err)

	require.Panics(t, func() {
		f.k.ExportGenesis(f.ctx)
	})
}

func TestRevokeDkimPubKeyStoredKeyUnparseable(t *testing.T) {
	pemKey := genRSAPrivKeyPEM(t)

	t.Run("stored key is not base64", func(t *testing.T) {
		f := SetupTest(t)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:   "bad.com",
			Selector: "s1",
			PubKey:   "!!not-base64!!",
		}, &f.k)
		require.NoError(t, err)

		_, err = f.msgServer.RevokeDkimPubKey(f.ctx, &types.MsgRevokeDkimPubKey{
			Signer:  f.addrs[0].String(),
			Domain:  "bad.com",
			PrivKey: pemKey,
		})
		require.ErrorIs(t, err, types.ErrInvalidPubKey)
	})

	t.Run("stored key is not DER", func(t *testing.T) {
		f := SetupTest(t)
		_, err := keeper.SaveDkimPubKey(f.ctx, types.DkimPubKey{
			Domain:   "bad.com",
			Selector: "s1",
			PubKey:   base64.StdEncoding.EncodeToString([]byte("garbage bytes")),
		}, &f.k)
		require.NoError(t, err)

		_, err = f.msgServer.RevokeDkimPubKey(f.ctx, &types.MsgRevokeDkimPubKey{
			Signer:  f.addrs[0].String(),
			Domain:  "bad.com",
			PrivKey: pemKey,
		})
		require.ErrorIs(t, err, types.ErrInvalidPubKey)
	})
}

func TestRevokeDkimPubKeyPKCS8NonRSA(t *testing.T) {
	f := SetupTest(t)

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	_, err = f.msgServer.RevokeDkimPubKey(f.ctx, &types.MsgRevokeDkimPubKey{
		Signer:  f.addrs[0].String(),
		Domain:  "example.com",
		PrivKey: pemKey,
	})
	require.ErrorIs(t, err, types.ErrParsingPrivKey)
	require.Contains(t, err.Error(), "not an RSA private key")
}

func TestAddDkimPubKeysPoseidonHashFailure(t *testing.T) {
	f := SetupTest(t)

	// A trailing newline is ignored by the base64 decoder, so the key passes
	// validation, but it breaks the PEM re-encoding used for the poseidon hash.
	_, err := f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
		Authority:   f.govModAddr,
		DkimPubkeys: []types.DkimPubKey{{Domain: "example.com", Selector: "s1", PubKey: testPubKey + "\n"}},
	})
	require.ErrorIs(t, err, types.ErrInvalidPubKey)
	require.Contains(t, err.Error(), "failed to compute poseidon hash")
}

func TestUpdateParamsValidation(t *testing.T) {
	f := SetupTest(t)

	t.Run("zero max pubkey size falls back to default", func(t *testing.T) {
		p := types.DefaultParams()
		p.MaxPubkeySizeBytes = 0
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: f.govModAddr, Params: p})
		require.NoError(t, err)

		stored, err := f.k.GetParams(f.ctx)
		require.NoError(t, err)
		require.Equal(t, types.DefaultMaxPubKeySizeBytes, stored.MaxPubkeySizeBytes)
	})

	t.Run("invalid params are rejected", func(t *testing.T) {
		p := types.DefaultParams()
		p.VkeyIdentifier = 0
		_, err := f.msgServer.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: f.govModAddr, Params: p})
		require.ErrorIs(t, err, types.ErrInvalidParams)
	})
}

func TestSetParamsRejectsInvalid(t *testing.T) {
	f := SetupTest(t)

	p := types.DefaultParams()
	p.PublicInputIndices.MinLength = 0
	err := f.k.SetParams(f.ctx, p)
	require.ErrorIs(t, err, types.ErrInvalidParams)
}
