package types_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

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
