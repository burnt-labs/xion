package types_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

const (
	validRSAPubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	validPrivKey   = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/M
FsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojM
M7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6
a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhR
VNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30G
i5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQABAoIBAFfx0xR7Z5LBQ3xj
-----END RSA PRIVATE KEY-----` // #nosec G101 -- Test credential, not a real secret
)

func TestMsgAddDkimPubKeys(t *testing.T) {
	addrs := simtestutil.CreateIncrementalAccounts(1)
	addr := addrs[0]
	validAddress := addr.String()

	validDkimKey := types.DkimPubKey{
		Domain:   "example.com",
		Selector: "default",
		PubKey:   validRSAPubKey,
		Version:  types.Version_VERSION_DKIM1_UNSPECIFIED,
		KeyType:  types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
	}

	t.Run("NewMsgAddDkimPubKeys", func(t *testing.T) {
		msg := types.NewMsgAddDkimPubKeys(addr, []types.DkimPubKey{validDkimKey})
		require.NotNil(t, msg)
		require.Equal(t, validAddress, msg.Authority)
		require.Len(t, msg.DkimPubkeys, 1)
	})

	t.Run("Route", func(t *testing.T) {
		msg := types.MsgAddDkimPubKeys{}
		require.Equal(t, types.ModuleName, msg.Route())
	})

	t.Run("Type", func(t *testing.T) {
		msg := types.MsgAddDkimPubKeys{}
		require.Equal(t, "add_dkim_public_keys", msg.Type())
	})

	t.Run("GetSigners", func(t *testing.T) {
		msg := &types.MsgAddDkimPubKeys{Authority: validAddress}
		signers := msg.GetSigners()
		require.Len(t, signers, 1)
		require.Equal(t, addr, signers[0])
	})

	t.Run("ValidateBasic - invalid address", func(t *testing.T) {
		msg := &types.MsgAddDkimPubKeys{
			Authority:   "invalid",
			DkimPubkeys: []types.DkimPubKey{validDkimKey},
		}
		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid authority address")
	})

	t.Run("ValidateBasic - invalid dkim key", func(t *testing.T) {
		invalidKey := validDkimKey
		invalidKey.PubKey = "invalid"
		msg := &types.MsgAddDkimPubKeys{
			Authority:   validAddress,
			DkimPubkeys: []types.DkimPubKey{invalidKey},
		}
		err := msg.ValidateBasic()
		require.Error(t, err)
	})
}

func TestMsgRemoveDkimPubKey(t *testing.T) {
	addrs := simtestutil.CreateIncrementalAccounts(1)
	addr := addrs[0]
	validAddress := addr.String()

	dkimKey := types.DkimPubKey{
		Domain:   "example.com",
		Selector: "default",
	}

	t.Run("NewMsgRemoveDkimPubKey", func(t *testing.T) {
		msg := types.NewMsgRemoveDkimPubKey(addr, dkimKey)
		require.NotNil(t, msg)
		require.Equal(t, validAddress, msg.Authority)
		require.Equal(t, "example.com", msg.Domain)
		require.Equal(t, "default", msg.Selector)
	})

	t.Run("Route", func(t *testing.T) {
		msg := types.MsgRemoveDkimPubKey{}
		require.Equal(t, types.ModuleName, msg.Route())
	})

	t.Run("Type", func(t *testing.T) {
		msg := types.MsgRemoveDkimPubKey{}
		require.Equal(t, "remove_dkim_public_keys", msg.Type())
	})

	t.Run("GetSigners", func(t *testing.T) {
		msg := &types.MsgRemoveDkimPubKey{Authority: validAddress}
		signers := msg.GetSigners()
		require.Len(t, signers, 1)
		require.Equal(t, addr, signers[0])
	})

	t.Run("ValidateBasic - valid", func(t *testing.T) {
		msg := &types.MsgRemoveDkimPubKey{
			Authority: validAddress,
			Domain:    "example.com",
			Selector:  "default",
		}
		err := msg.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("ValidateBasic - invalid address", func(t *testing.T) {
		msg := &types.MsgRemoveDkimPubKey{
			Authority: "invalid",
			Domain:    "example.com",
			Selector:  "default",
		}
		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid authority address")
	})
}

func TestMsgRevokeDkimPubKey(t *testing.T) {
	addrs := simtestutil.CreateIncrementalAccounts(1)
	addr := addrs[0]
	validAddress := addr.String()

	t.Run("NewMsgRevokeDkimPubKey", func(t *testing.T) {
		msg := types.NewMsgRevokeDkimPubKey(addr, "example.com", []byte(validPrivKey))
		require.NotNil(t, msg)
		require.Equal(t, validAddress, msg.Signer)
		require.Equal(t, "example.com", msg.Domain)
		require.NotEmpty(t, msg.PrivKey)
	})

	t.Run("Route", func(t *testing.T) {
		msg := types.MsgRevokeDkimPubKey{}
		require.Equal(t, types.ModuleName, msg.Route())
	})

	t.Run("Type", func(t *testing.T) {
		msg := types.MsgRevokeDkimPubKey{}
		require.Equal(t, "remove_dkim_public_keys", msg.Type())
	})

	t.Run("GetSigners", func(t *testing.T) {
		msg := &types.MsgRevokeDkimPubKey{Signer: validAddress}
		signers := msg.GetSigners()
		require.Len(t, signers, 1)
		require.Equal(t, addr, signers[0])
	})

	t.Run("ValidateBasic - invalid domain URL", func(t *testing.T) {
		msg := &types.MsgRevokeDkimPubKey{
			Signer:  validAddress,
			Domain:  "ht tp://invalid url",
			PrivKey: []byte(validPrivKey),
		}
		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "dkim url key parsing failed")
	})

	t.Run("ValidateBasic - invalid PEM", func(t *testing.T) {
		msg := &types.MsgRevokeDkimPubKey{
			Signer:  validAddress,
			Domain:  "https://example.com",
			PrivKey: []byte("not a pem key"),
		}
		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode pem private key")
	})

	t.Run("ValidateBasic - valid PKCS8 key", func(t *testing.T) {
		// Generate a test RSA private key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Marshal to PKCS8 format
		pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
		require.NoError(t, err)

		// Encode as PEM
		pkcs8PEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: pkcs8Bytes,
		})

		msg := &types.MsgRevokeDkimPubKey{
			Signer:  validAddress,
			Domain:  "https://example.com",
			PrivKey: pkcs8PEM,
		}
		err = msg.ValidateBasic()
		require.NoError(t, err)
	})
}

func TestMsgUpdateParams(t *testing.T) {
	addrs := simtestutil.CreateIncrementalAccounts(1)
	addr := addrs[0]

	t.Run("ValidateBasic valid", func(t *testing.T) {
		msg := &types.MsgUpdateParams{
			Authority: addr.String(),
			Params:    types.DefaultParams(),
		}
		require.NoError(t, msg.ValidateBasic())
	})

	t.Run("ValidateBasic invalid authority", func(t *testing.T) {
		msg := &types.MsgUpdateParams{
			Authority: "not-bech32",
			Params:    types.DefaultParams(),
		}
		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid authority address")
	})

	t.Run("ValidateBasic invalid params", func(t *testing.T) {
		msg := &types.MsgUpdateParams{
			Authority: addr.String(),
			Params:    types.Params{MaxPubkeySizeBytes: 0},
		}
		require.Error(t, msg.ValidateBasic())
	})

	t.Run("GetSigners", func(t *testing.T) {
		msg := &types.MsgUpdateParams{Authority: addr.String()}
		require.Equal(t, []sdk.AccAddress{addr}, msg.GetSigners())
	})
}

func TestValidateDkimPubKeys(t *testing.T) {
	pkixKey, pkcs1Key := generateRSAPubKeyEncodings(t)
	params := types.Params{MaxPubkeySizeBytes: 2048}
	validKey := types.DkimPubKey{
		Domain:   "example.com",
		Selector: "default",
		PubKey:   pkixKey,
		Version:  types.Version_VERSION_DKIM1_UNSPECIFIED,
		KeyType:  types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
	}

	t.Run("valid key list", func(t *testing.T) {
		require.NoError(t, types.ValidateDkimPubKeys([]types.DkimPubKey{validKey}, params))
	})

	t.Run("invalid key type", func(t *testing.T) {
		invalid := validKey
		invalid.KeyType = types.KeyType(999)
		err := types.ValidateDkimPubKeys([]types.DkimPubKey{invalid}, params)
		require.ErrorIs(t, err, types.ErrInvalidKeyType)
	})

	t.Run("exceeds size limit", func(t *testing.T) {
		oversized := validKey
		oversized.PubKey = base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4, 5})
		err := types.ValidateDkimPubKeys([]types.DkimPubKey{oversized}, types.Params{MaxPubkeySizeBytes: 4})
		require.ErrorIs(t, err, types.ErrPubKeyTooLarge)
	})

	t.Run("invalid rsa key bytes", func(t *testing.T) {
		invalid := validKey
		invalid.PubKey = base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
		err := types.ValidateDkimPubKeys([]types.DkimPubKey{invalid}, params)
		require.ErrorIs(t, err, types.ErrInvalidPubKey)
	})

	t.Run("pkcs1 encoded key accepted", func(t *testing.T) {
		key := validKey
		key.PubKey = pkcs1Key
		require.NoError(t, types.ValidateDkimPubKeys([]types.DkimPubKey{key}, params))
	})
}

func TestValidateRSAPubKey(t *testing.T) {
	pkixKey, pkcs1Key := generateRSAPubKeyEncodings(t)

	t.Run("valid pkix", func(t *testing.T) {
		require.NoError(t, types.ValidateRSAPubKey(pkixKey))
	})

	t.Run("valid pkcs1", func(t *testing.T) {
		require.NoError(t, types.ValidateRSAPubKey(pkcs1Key))
	})

	t.Run("non rsa key", func(t *testing.T) {
		ecdsaKey := generateECDSAPubKeyEncoding(t)
		err := types.ValidateRSAPubKey(ecdsaKey)
		require.ErrorIs(t, err, types.ErrNotRSAKey)
	})

	t.Run("invalid base64", func(t *testing.T) {
		require.ErrorIs(t, types.ValidateRSAPubKey("$$"), types.ErrInvalidPubKey)
	})

	t.Run("invalid rsa bytes", func(t *testing.T) {
		raw := base64.StdEncoding.EncodeToString([]byte{9, 9, 9})
		require.ErrorIs(t, types.ValidateRSAPubKey(raw), types.ErrInvalidPubKey)
	})
}

func generateRSAPubKeyEncodings(t *testing.T) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pkixBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	pkcs1Bytes := x509.MarshalPKCS1PublicKey(&key.PublicKey)

	return base64.StdEncoding.EncodeToString(pkixBytes), base64.StdEncoding.EncodeToString(pkcs1Bytes)
}

func generateECDSAPubKeyEncoding(t *testing.T) string {
	t.Helper()
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	bz, err := x509.MarshalPKIXPublicKey(&ecKey.PublicKey)
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(bz)
}

func TestValidateDkimPubKeysWithRevocation(t *testing.T) {
	params := types.DefaultParams()

	validKey := types.DkimPubKey{
		Domain:   "example.com",
		Selector: "default",
		PubKey:   validRSAPubKey,
		Version:  types.Version_VERSION_DKIM1_UNSPECIFIED,
		KeyType:  types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
	}

	t.Run("valid keys with no revocation check", func(t *testing.T) {
		ctx := sdk.Context{}
		err := types.ValidateDkimPubKeysWithRevocation(
			ctx,
			[]types.DkimPubKey{validKey},
			params,
			nil,
		)
		require.NoError(t, err)
	})

	t.Run("valid keys not revoked", func(t *testing.T) {
		ctx := sdk.Context{}
		isRevoked := func(context.Context, string) (bool, error) {
			return false, nil
		}
		err := types.ValidateDkimPubKeysWithRevocation(
			ctx,
			[]types.DkimPubKey{validKey},
			params,
			isRevoked,
		)
		require.NoError(t, err)
	})

	t.Run("key is revoked", func(t *testing.T) {
		ctx := sdk.Context{}
		isRevoked := func(context.Context, string) (bool, error) {
			return true, nil
		}
		err := types.ValidateDkimPubKeysWithRevocation(
			ctx,
			[]types.DkimPubKey{validKey},
			params,
			isRevoked,
		)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvalidatedKey)
		require.Contains(t, err.Error(), "has been revoked")
	})

	t.Run("error checking revocation", func(t *testing.T) {
		ctx := sdk.Context{}
		testErr := types.ErrInvalidPubKey
		isRevoked := func(context.Context, string) (bool, error) {
			return false, testErr
		}
		err := types.ValidateDkimPubKeysWithRevocation(
			ctx,
			[]types.DkimPubKey{validKey},
			params,
			isRevoked,
		)
		require.Error(t, err)
		require.ErrorIs(t, err, testErr)
	})

	t.Run("invalid key metadata", func(t *testing.T) {
		ctx := sdk.Context{}
		invalidKey := validKey
		invalidKey.KeyType = types.KeyType(999) // invalid key type
		err := types.ValidateDkimPubKeysWithRevocation(
			ctx,
			[]types.DkimPubKey{invalidKey},
			params,
			nil,
		)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvalidKeyType)
	})

	t.Run("invalid pubkey encoding", func(t *testing.T) {
		ctx := sdk.Context{}
		invalidKey := validKey
		invalidKey.PubKey = "invalid_base64"
		err := types.ValidateDkimPubKeysWithRevocation(
			ctx,
			[]types.DkimPubKey{invalidKey},
			params,
			nil,
		)
		require.Error(t, err)
	})

	t.Run("invalid RSA key", func(t *testing.T) {
		ctx := sdk.Context{}
		invalidKey := validKey
		invalidKey.PubKey = generateECDSAPubKeyEncoding(t)
		err := types.ValidateDkimPubKeysWithRevocation(
			ctx,
			[]types.DkimPubKey{invalidKey},
			params,
			nil,
		)
		require.Error(t, err)
	})
}

func TestValidateDkimPubKey(t *testing.T) {
	validKey := types.DkimPubKey{
		Domain:   "example.com",
		Selector: "default",
		PubKey:   validRSAPubKey,
		Version:  types.Version_VERSION_DKIM1_UNSPECIFIED,
		KeyType:  types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
	}

	t.Run("valid key", func(t *testing.T) {
		err := types.ValidateDkimPubKey(validKey)
		require.NoError(t, err)
	})

	t.Run("invalid metadata", func(t *testing.T) {
		invalidKey := validKey
		invalidKey.Version = types.Version(999) // invalid version
		err := types.ValidateDkimPubKey(invalidKey)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvalidVersion)
	})

	t.Run("invalid base64 encoding", func(t *testing.T) {
		invalidKey := validKey
		invalidKey.PubKey = "not-valid-base64!@#$"
		err := types.ValidateDkimPubKey(invalidKey)
		require.Error(t, err)
	})

	t.Run("invalid RSA key bytes", func(t *testing.T) {
		invalidKey := validKey
		invalidKey.PubKey = base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
		err := types.ValidateDkimPubKey(invalidKey)
		require.Error(t, err)
	})

	t.Run("non-RSA key", func(t *testing.T) {
		invalidKey := validKey
		invalidKey.PubKey = generateECDSAPubKeyEncoding(t)
		err := types.ValidateDkimPubKey(invalidKey)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrNotRSAKey)
	})
}
