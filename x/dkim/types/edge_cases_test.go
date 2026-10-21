package types_test

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

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func ecdsaPKIXDER(t *testing.T) []byte {
	t.Helper()
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&ecKey.PublicKey)
	require.NoError(t, err)
	return der
}

func TestDecodePubKeyInvalidBase64(t *testing.T) {
	_, err := types.DecodePubKey("!!not base64!!")
	require.ErrorIs(t, err, types.ErrInvalidPubKey)
}

func TestParseRSAPublicKeyRejectsNonRSAPKIX(t *testing.T) {
	_, err := types.ParseRSAPublicKey(ecdsaPKIXDER(t))
	require.ErrorIs(t, err, types.ErrNotRSAKey)
}

func TestComputePoseidonHashRejects(t *testing.T) {
	t.Run("base64 that does not survive PEM re-encoding", func(t *testing.T) {
		// the decoder ignores the newline, but the padding appended after it
		// makes the PEM body undecodable
		_, err := types.ComputePoseidonHash("QUJD\n")
		require.Error(t, err)
		require.Contains(t, err.Error(), "public key is invalid")
	})

	t.Run("PKIX key that is not RSA", func(t *testing.T) {
		_, err := types.ComputePoseidonHash(base64.StdEncoding.EncodeToString(ecdsaPKIXDER(t)))
		require.ErrorIs(t, err, types.ErrParsingPubKey)
		require.Contains(t, err.Error(), "not an RSA public key")
	})
}

func TestGenesisStateValidateRejectsInvalidPubKey(t *testing.T) {
	gs := types.GenesisState{
		Params: types.DefaultParams(),
		DkimPubkeys: []types.DkimPubKey{{
			Domain:   "example.com",
			Selector: "s1",
			PubKey:   base64.StdEncoding.EncodeToString([]byte("not a key")),
		}},
	}
	require.ErrorIs(t, gs.Validate(), types.ErrInvalidPubKey)
}

func TestDkimPubKeyEqualFieldByField(t *testing.T) {
	base := types.DkimPubKey{
		Domain:       "example.com",
		PubKey:       "pub",
		PoseidonHash: []byte{1, 2, 3},
		Selector:     "selector",
		Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
	}

	t.Run("typed nil pointer only equals nil receiver", func(t *testing.T) {
		var nilKey, otherNilKey *types.DkimPubKey
		require.False(t, base.Equal(nilKey))
		require.True(t, nilKey.Equal(otherNilKey))
	})

	mutations := map[string]func(k *types.DkimPubKey){
		"pubkey":        func(k *types.DkimPubKey) { k.PubKey = "other" },
		"selector":      func(k *types.DkimPubKey) { k.Selector = "other" },
		"version":       func(k *types.DkimPubKey) { k.Version = types.Version(7) },
		"key type":      func(k *types.DkimPubKey) { k.KeyType = types.KeyType(7) },
		"poseidon hash": func(k *types.DkimPubKey) { k.PoseidonHash = []byte{9} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			other := base
			mutate(&other)
			require.False(t, base.Equal(&other))
			require.False(t, base.Equal(other))
		})
	}

	t.Run("identical pointer values are equal", func(t *testing.T) {
		other := base
		require.True(t, base.Equal(&other))
	})
}

func TestMsgAddDkimPubKeysValidateBasicAccepts(t *testing.T) {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	msg := types.NewMsgAddDkimPubKeys(authority, []types.DkimPubKey{{
		Domain:   "example.com",
		Selector: "s1",
		PubKey:   validRSAPubKey,
	}})
	require.NoError(t, msg.ValidateBasic())
	require.Equal(t, []sdk.AccAddress{authority}, msg.GetSigners())
}

func TestMsgRevokeDkimPubKeyValidateBasicKeyFormats(t *testing.T) {
	signer := authtypes.NewModuleAddress(govtypes.ModuleName)

	t.Run("PEM block that is neither PKCS1 nor PKCS8", func(t *testing.T) {
		msg := types.NewMsgRevokeDkimPubKey(signer, "example.com", pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: []byte("garbage"),
		}))
		err := msg.ValidateBasic()
		require.ErrorIs(t, err, types.ErrParsingPrivKey)
		require.Contains(t, err.Error(), "failed to parse private key")
	})

	t.Run("PKCS8 key that is not RSA", func(t *testing.T) {
		ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		der, err := x509.MarshalPKCS8PrivateKey(ecKey)
		require.NoError(t, err)
		msg := types.NewMsgRevokeDkimPubKey(signer, "example.com", pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		}))
		err = msg.ValidateBasic()
		require.ErrorIs(t, err, types.ErrParsingPrivKey)
		require.Contains(t, err.Error(), "not an RSA private key")
	})

	t.Run("PKCS1 RSA key is accepted", func(t *testing.T) {
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		msg := types.NewMsgRevokeDkimPubKey(signer, "example.com", pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
		}))
		require.NoError(t, msg.ValidateBasic())
	})
}

func TestParamsValidateRejectsBadIndices(t *testing.T) {
	p := types.DefaultParams()
	p.PublicInputIndices.EmailHashIndex = p.PublicInputIndices.MinLength
	err := p.Validate()
	require.ErrorIs(t, err, types.ErrInvalidParams)
	require.Contains(t, err.Error(), "email_hash_index")
}

func TestValidateForcedSubjectMIMEDecodesToBlank(t *testing.T) {
	// base64 of a single space: the decoded subject is empty after trimming
	require.False(t, types.ValidateForcedSubject("=?utf-8?B?IA==?="))
}
